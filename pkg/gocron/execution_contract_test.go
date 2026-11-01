package gocron

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type executionLocker struct {
	lockErr, unlockErr error
	key                string
	locks, unlocks     int
	locked             bool
}

func (l *executionLocker) Lock(key string) (bool, error) {
	l.locks++
	l.key = key
	return l.locked, l.lockErr
}
func (l *executionLocker) Unlock(key string) error {
	l.unlocks++
	if key != l.key {
		panic("different unlock key")
	}
	return l.unlockErr
}

func TestJobLockExecutionContract(t *testing.T) {
	old := locker
	t.Cleanup(func() { locker = old })
	//nolint:govet // Keep positional test cases grouped by expected outcome.
	for _, tc := range []struct {
		name           string
		l              *executionLocker
		calls, unlocks int
		wantErr        bool
	}{
		{"missing", nil, 0, 0, true},
		{"failure", &executionLocker{lockErr: errors.New("unavailable")}, 0, 0, true},
		{"held", &executionLocker{}, 0, 0, false},
		{"acquired", &executionLocker{locked: true}, 1, 1, false},
		{"release failure", &executionLocker{locked: true, unlockErr: errors.New("unavailable")}, 1, 1, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls int
			j := NewJob(1).Seconds().Lock()
			require.NoError(t, j.Do(func() { calls++ }))
			if tc.l == nil {
				locker = nil
			} else {
				locker = tc.l
			}
			err := j.run()
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.calls, calls)
			if tc.l != nil {
				require.Equal(t, 1, tc.l.locks)
				require.Equal(t, tc.unlocks, tc.l.unlocks)
				require.Len(t, tc.l.key, 64)
			}
		})
	}
}

func TestJobCancellationTimeoutAndParameterErrors(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	j := NewJob(1).Seconds().WithContext(ctx)
	require.NoError(t, j.Do(func() { t.Error("cancelled job executed") }))
	require.ErrorIs(t, j.run(), context.Canceled)
	//nolint:staticcheck // Verify the documented nil-context fallback.
	j = NewJob(1).Seconds().WithContext(nil).WithTimeout(time.Millisecond)
	require.NoError(t, j.Do(func(ctx context.Context) { <-ctx.Done() }))
	require.NoError(t, j.run()) // context-aware functions retain their own return semantics
	j = NewJob(1).Seconds()
	require.NoError(t, j.Do(func(int) {}))
	require.ErrorIs(t, j.run(), ErrParamsNotAdapted)
	j = NewJob(1).Seconds().WithTimeout(time.Millisecond)
	release, finished := make(chan struct{}), make(chan struct{})
	require.NoError(t, j.Do(func() { defer close(finished); <-release }))
	require.ErrorIs(t, j.run(), context.DeadlineExceeded)
	close(release)
	<-finished
}

func TestSafeJobHandlesPanicAndParameterMismatch(t *testing.T) {
	for _, fn := range []any{func() { panic("expected") }, func(int) {}} {
		j := NewJob(1).Seconds()
		require.NoError(t, j.DoSafely(fn))
		require.NoError(t, j.run())
	}
	j := NewJob(1)
	require.NoError(t, j.Do(func() {}))
	require.ErrorIs(t, j.Err(), ErrPeriodNotSpecified)
	for _, j := range []*Job{NewJob(2).Second(), NewJob(2).Minute(), NewJob(2).Hour(), NewJob(2).Day(), NewJob(2).Week(), NewJob(2).Monday()} {
		require.Error(t, j.Err())
	}
}

func TestJobBuilderErrorsAreRetained(t *testing.T) {
	j := NewJob(2).Second()
	require.Error(t, j.Do(func() {}))
	require.ErrorIs(t, NewJob(1).Seconds().Do("not a function"), ErrNotAFunction)
	duration, err := NewJob(2).Hours().periodDuration()
	require.NoError(t, err)
	require.Equal(t, 2*time.Hour, duration)
	//nolint:staticcheck // Verify the documented nil-context fallback.
	scheduler := NewSchedulerWithContext(nil)
	require.False(t, scheduler.Scheduled(func() {}))
	j = scheduler.Every(1).Seconds()
	j.Tag("keep")
	scheduler.RemoveByTag("missing")
	require.Len(t, scheduler.Jobs(), 1)
}
