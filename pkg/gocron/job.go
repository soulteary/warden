package gocron

import (
	"context"
	"errors"
	"fmt"
	"log"
	"reflect"
	"sync"
	"time"
)

var (
	// ErrTimeFormat 时间格式错误
	ErrTimeFormat = errors.New("time format error")
	// ErrParamsNotAdapted 参数数量不匹配
	ErrParamsNotAdapted = errors.New("the number of params is not adapted")
	// ErrNotAFunction 不是函数错误，只有函数才能被调度到任务队列
	ErrNotAFunction = errors.New("only functions can be schedule into the job queue")
	// ErrPeriodNotSpecified 未指定任务周期错误
	ErrPeriodNotSpecified = errors.New("unspecified job period")
	// ErrParameterCannotBeNil 参数不能为 nil 错误
	ErrParameterCannotBeNil = errors.New("nil parameters cannot be used with reflection")
	// ErrJobTimeout 任务执行超时错误
	ErrJobTimeout = errors.New("job execution timeout")
	// ErrJobCancelled 任务执行被取消错误
	ErrJobCancelled = errors.New("job execution cancelled")
)

// Job struct keeping information about job
//
//nolint:govet // fieldalignment: 字段顺序已优化，但为了保持 API 兼容性，不进一步调整
type Job struct {
	mu       sync.RWMutex             // 24 bytes - mutex to protect concurrent access to lastRun and nextRun
	lastRun  time.Time                // 24 bytes - datetime of last run
	nextRun  time.Time                // 24 bytes - datetime of next run
	tags     []string                 // 24 bytes (8 pointer + 8 len + 8 cap) - allow the user to tag jobs with certain labels
	ctx      context.Context          // 16 bytes interface - optional context for job execution
	err      error                    // 16 bytes interface - error related to job
	jobFunc  string                   // 16 bytes - the job jobFunc to run, func[jobFunc]
	atTime   time.Duration            // 8 bytes - optional time at which this job runs
	timeout  time.Duration            // 8 bytes - optional timeout for job execution
	interval uint64                   // 8 bytes - pause interval * unit between runs
	funcs    map[string]interface{}   // 8 bytes pointer - Map for the function task store
	fparams  map[string][]interface{} // 8 bytes pointer - Map for function and params of function
	loc      *time.Location           // 8 bytes pointer - optional timezone that the atTime is in
	unit     timeUnit                 // 1 byte - time units, e.g. 'minutes', 'hours'...
	startDay time.Weekday             // 1 byte - Specific day of the week to start on
	lock     bool                     // 1 byte - lock the job from running at same time form multiple instances
}

// NewJob creates a new job with the time interval.
func NewJob(interval uint64) *Job {
	return &Job{
		interval: interval,
		loc:      loc,
		lastRun:  time.Unix(0, 0),
		nextRun:  time.Unix(0, 0),
		startDay: time.Sunday,
		funcs:    make(map[string]interface{}),
		fparams:  make(map[string][]interface{}),
		tags:     []string{},
		ctx:      context.Background(),
		timeout:  0, // 0 means no timeout
	}
}

// True if the job should be run now
func (j *Job) shouldRun() bool {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return time.Now().Unix() >= j.nextRun.Unix()
}

// Run the job and immediately reschedule it
func (j *Job) run() error {
	// Create execution context with timeout if specified
	execCtx := j.ctx
	if j.timeout > 0 {
		var cancel context.CancelFunc
		execCtx, cancel = context.WithTimeout(j.ctx, j.timeout)
		defer cancel()
	}

	// Check if context is already cancelled
	select {
	case <-execCtx.Done():
		return fmt.Errorf("job %s cancelled before execution: %w", j.jobFunc, execCtx.Err())
	default:
	}

	if j.lock {
		if locker == nil {
			return fmt.Errorf("trying to lock %s with nil locker", j.jobFunc)
		}
		key := getFunctionKey(j.jobFunc)

		locked, err := locker.Lock(key)
		if err != nil {
			return fmt.Errorf("failed to lock job %s: %w", j.jobFunc, err)
		}
		if !locked {
			// Job is already running in another instance, skip execution
			return nil
		}
		defer func() {
			if err := locker.Unlock(key); err != nil {
				log.Printf("解锁失败: %v", err)
			}
		}()
	}

	// Execute job with context support
	_, err := j.runWithContext(execCtx)
	if err != nil {
		return err
	}
	return nil
}

// runWithContext executes the job function with context support
func (j *Job) runWithContext(ctx context.Context) ([]reflect.Value, error) {
	jobFunc := j.funcs[j.jobFunc]
	funcType := reflect.TypeOf(jobFunc)
	originalParams := j.fparams[j.jobFunc]

	// Check if function accepts context as first parameter
	if funcType.NumIn() > 0 {
		firstParamType := funcType.In(0)
		contextType := reflect.TypeOf((*context.Context)(nil)).Elem()

		// Check if first parameter is context.Context
		if firstParamType == contextType {
			// Function accepts context, create new params with context
			params := make([]interface{}, 0, len(originalParams)+1)
			params = append(params, ctx)
			params = append(params, originalParams...)
			return callJobFuncWithParams(jobFunc, params)
		}
	}

	// Function doesn't accept context, run normally but check context cancellation
	done := make(chan []reflect.Value, 1)
	errChan := make(chan error, 1)

	go func() {
		result, err := callJobFuncWithParams(jobFunc, originalParams)
		if err != nil {
			errChan <- err
			return
		}
		done <- result
	}()

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("job %s execution cancelled: %w", j.jobFunc, ctx.Err())
	case err := <-errChan:
		return nil, err
	case result := <-done:
		return result, nil
	}
}

// Err should be checked to ensure an error didn't occur creating the job
func (j *Job) Err() error {
	return j.err
}

// Do specifies the jobFunc that should be called every time the job runs
func (j *Job) Do(jobFun interface{}, params ...interface{}) error {
	if j.err != nil {
		return j.err
	}

	typ := reflect.TypeOf(jobFun)
	if typ.Kind() != reflect.Func {
		return ErrNotAFunction
	}
	fname := getFunctionName(jobFun)
	j.funcs[fname] = jobFun
	j.fparams[fname] = params
	j.jobFunc = fname

	now := time.Now().In(j.loc)
	j.mu.RLock()
	shouldSchedule := !j.nextRun.After(now)
	j.mu.RUnlock()
	if shouldSchedule {
		if err := j.scheduleNextRun(); err != nil {
			j.err = err
		}
	}

	return nil
}

// DoSafely does the same thing as Do, but logs unexpected panics, instead of unwinding them up the chain
//
// Deprecated: DoSafely exists due to historical compatibility and will be removed soon. Use Do instead
func (j *Job) DoSafely(jobFun interface{}, params ...interface{}) error {
	recoveryWrapperFunc := func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Internal panic occurred: %s", r)
			}
		}()

		_, err := callJobFuncWithParams(jobFun, params)
		if err != nil {
			log.Printf("执行任务失败: %v", err)
		}
	}

	return j.Do(recoveryWrapperFunc)
}

// At schedules job at specific time of day
//
//	s.Every(1).Day().At("10:30:01").Do(task)
//	s.Every(1).Monday().At("10:30:01").Do(task)
func (j *Job) At(t string) *Job {
	hour, minute, sec, err := formatTime(t)
	if err != nil {
		j.err = ErrTimeFormat
		return j
	}
	// save atTime start as duration from midnight
	j.atTime = time.Duration(hour)*time.Hour + time.Duration(minute)*time.Minute + time.Duration(sec)*time.Second
	return j
}

// GetAt returns the specific time of day the job will run at
//
//	s.Every(1).Day().At("10:30").GetAt() == "10:30"
func (j *Job) GetAt() string {
	return fmt.Sprintf("%d:%d", j.atTime/time.Hour, (j.atTime%time.Hour)/time.Minute)
}

// Loc sets the location for which to interpret "At"
//
//	s.Every(1).Day().At("10:30").Loc(time.UTC).Do(task)
func (j *Job) Loc(loc *time.Location) *Job {
	j.loc = loc
	return j
}

// Tag allows you to add labels to a job
// they don't impact the functionality of the job.
func (j *Job) Tag(t string, others ...string) {
	j.tags = append(j.tags, t)
	j.tags = append(j.tags, others...)
}

// Untag removes a tag from a job
func (j *Job) Untag(t string) {
	newTags := []string{}
	for _, tag := range j.tags {
		if t != tag {
			newTags = append(newTags, tag)
		}
	}

	j.tags = newTags
}

// Tags returns the tags attached to the job
func (j *Job) Tags() []string {
	return j.tags
}

func (j *Job) periodDuration() (time.Duration, error) {
	// #nosec G115 -- 转换是安全的，interval 是 uint64
	interval := time.Duration(j.interval)
	var periodDuration time.Duration

	switch j.unit {
	case seconds:
		periodDuration = interval * time.Second
	case minutes:
		periodDuration = interval * time.Minute
	case hours:
		periodDuration = interval * time.Hour
	case days:
		periodDuration = interval * time.Hour * 24
	case weeks:
		periodDuration = interval * time.Hour * 24 * 7
	default:
		return 0, ErrPeriodNotSpecified
	}
	return periodDuration, nil
}

// roundToMidnight truncate time to midnight
func (j *Job) roundToMidnight(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, j.loc)
}

// scheduleNextRun Compute the instant when this job should run next
func (j *Job) scheduleNextRun() error {
	j.mu.Lock()
	defer j.mu.Unlock()

	now := time.Now()
	if j.lastRun.Equal(time.Unix(0, 0)) {
		j.lastRun = now
	}

	periodDuration, err := j.periodDuration()
	if err != nil {
		return err
	}

	switch j.unit {
	case seconds, minutes, hours:
		j.nextRun = j.lastRun.Add(periodDuration)
	case days:
		j.nextRun = j.roundToMidnight(j.lastRun)
		j.nextRun = j.nextRun.Add(j.atTime)
	case weeks:
		j.nextRun = j.roundToMidnight(j.lastRun)
		dayDiff := int(j.startDay)
		dayDiff -= int(j.nextRun.Weekday())
		if dayDiff != 0 {
			j.nextRun = j.nextRun.Add(time.Duration(dayDiff) * 24 * time.Hour)
		}
		j.nextRun = j.nextRun.Add(j.atTime)
	}

	// advance to next possible schedule
	for j.nextRun.Before(now) || j.nextRun.Before(j.lastRun) {
		j.nextRun = j.nextRun.Add(periodDuration)
	}

	return nil
}

// NextScheduledTime returns the time of when this job is to run next
func (j *Job) NextScheduledTime() time.Time {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return j.nextRun
}

// set the job's unit with seconds,minutes,hours...
// #nosec G107 -- expected 参数总是为 1，这是设计上的要求
//
//nolint:unparam // expected 参数总是为 1，这是设计上的要求
func (j *Job) mustInterval(expected uint64) error {
	if j.interval != expected {
		return fmt.Errorf("interval must be %d", expected)
	}
	return nil
}

// From schedules the next run of the job
func (j *Job) From(t *time.Time) *Job {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.nextRun = *t
	return j
}

// setUnit sets unit type
func (j *Job) setUnit(unit timeUnit) *Job {
	j.unit = unit
	return j
}

// Seconds set the unit with seconds
func (j *Job) Seconds() *Job {
	return j.setUnit(seconds)
}

// Minutes set the unit with minute
func (j *Job) Minutes() *Job {
	return j.setUnit(minutes)
}

// Hours set the unit with hours
func (j *Job) Hours() *Job {
	return j.setUnit(hours)
}

// Days set the job's unit with days
func (j *Job) Days() *Job {
	return j.setUnit(days)
}

// Weeks sets the units as weeks
func (j *Job) Weeks() *Job {
	return j.setUnit(weeks)
}

// Second sets the unit with second
func (j *Job) Second() *Job {
	if err := j.mustInterval(1); err != nil {
		j.err = err
	}
	return j.Seconds()
}

// Minute sets the unit  with minute, which interval is 1
func (j *Job) Minute() *Job {
	if err := j.mustInterval(1); err != nil {
		j.err = err
	}
	return j.Minutes()
}

// Hour sets the unit with hour, which interval is 1
func (j *Job) Hour() *Job {
	if err := j.mustInterval(1); err != nil {
		j.err = err
	}
	return j.Hours()
}

// Day sets the job's unit with day, which interval is 1
func (j *Job) Day() *Job {
	if err := j.mustInterval(1); err != nil {
		j.err = err
	}
	return j.Days()
}

// Week sets the job's unit with week, which interval is 1
func (j *Job) Week() *Job {
	if err := j.mustInterval(1); err != nil {
		j.err = err
	}
	return j.Weeks()
}

// Weekday start job on specific Weekday
func (j *Job) Weekday(startDay time.Weekday) *Job {
	if err := j.mustInterval(1); err != nil {
		j.err = err
	}
	j.startDay = startDay
	return j.Weeks()
}

// GetWeekday returns which day of the week the job will run on
// This should only be used when .Weekday(...) was called on the job.
func (j *Job) GetWeekday() time.Weekday {
	return j.startDay
}

// Monday set the start day with Monday
// - s.Every(1).Monday().Do(task)
func (j *Job) Monday() (job *Job) {
	return j.Weekday(time.Monday)
}

// Tuesday sets the job start day Tuesday
func (j *Job) Tuesday() *Job {
	return j.Weekday(time.Tuesday)
}

// Wednesday sets the job start day Wednesday
func (j *Job) Wednesday() *Job {
	return j.Weekday(time.Wednesday)
}

// Thursday sets the job start day Thursday
func (j *Job) Thursday() *Job {
	return j.Weekday(time.Thursday)
}

// Friday sets the job start day Friday
func (j *Job) Friday() *Job {
	return j.Weekday(time.Friday)
}

// Saturday sets the job start day Saturday
func (j *Job) Saturday() *Job {
	return j.Weekday(time.Saturday)
}

// Sunday sets the job start day Sunday
func (j *Job) Sunday() *Job {
	return j.Weekday(time.Sunday)
}

// Lock prevents job to run from multiple instances of gocron
func (j *Job) Lock() *Job {
	j.lock = true
	return j
}

// WithContext sets the context for job execution
// If the job function accepts context.Context as its first parameter,
// it will be passed automatically. Otherwise, the context is used for
// cancellation and timeout control.
//
//	scheduler.Every(1).Minute().WithContext(ctx).Do(task)
func (j *Job) WithContext(ctx context.Context) *Job {
	if ctx == nil {
		ctx = context.Background()
	}
	j.ctx = ctx
	return j
}

// WithTimeout sets a timeout for job execution
// If the job doesn't complete within the timeout, it will be cancelled.
//
//	scheduler.Every(1).Minute().WithTimeout(30*time.Second).Do(task)
func (j *Job) WithTimeout(timeout time.Duration) *Job {
	j.timeout = timeout
	return j
}
