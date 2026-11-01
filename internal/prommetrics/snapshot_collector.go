package prommetrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	metricskit "github.com/soulteary/metrics-kit/v3"
)

// SnapshotSample contains one atomically captured generation, without user data.
// LoadedAt retains the application's mode-specific age basis.
type SnapshotSample struct {
	LoadedAt          time.Time
	SourceSucceededAt time.Time
	PublishedAt       time.Time
	RestoredAt        time.Time
	Degraded          bool
}

type snapshotCollector struct {
	sample func() SnapshotSample
	now    func() time.Time
	descs  []*prometheus.Desc
}

func newSnapshotCollector(sample func() SnapshotSample, now func() time.Time) *snapshotCollector {
	names := []string{"snapshot_age_seconds", "snapshot_degraded", "snapshot_loaded_timestamp_seconds", "snapshot_age_known", "snapshot_source_succeeded_timestamp_seconds", "snapshot_source_time_known", "snapshot_published_timestamp_seconds", "snapshot_publication_time_known", "snapshot_restored_timestamp_seconds", "snapshot_restoration_time_known"}
	helps := []string{"Age in seconds of the current rule-set snapshot", "Whether the current snapshot is degraded (1) or healthy (0)", "Age-basis timestamp of the current snapshot, or zero if unknown", "Whether the snapshot age basis is known", "Accepted source validation timestamp, or zero if unknown", "Whether the source validation time is known", "Redis publication timestamp, or zero if unknown", "Whether Redis publication time is known", "Local restoration timestamp, or zero if unknown", "Whether local restoration time is known"}
	c := &snapshotCollector{sample: sample, now: now}
	for i, name := range names {
		c.descs = append(c.descs, prometheus.NewDesc("warden_"+name, helps[i], nil, nil))
	}
	return c
}

func (c *snapshotCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, desc := range c.descs {
		ch <- desc
	}
}
func (c *snapshotCollector) Collect(ch chan<- prometheus.Metric) {
	state := c.sample()
	now := c.now()
	age := 0.0
	if !state.LoadedAt.IsZero() {
		age = now.Sub(state.LoadedAt).Seconds()
		if age < 0 {
			age = 0
		}
	}
	degraded := 0.0
	if state.Degraded {
		degraded = 1
	}
	values := []float64{age, degraded}
	for _, at := range []time.Time{state.LoadedAt, state.SourceSucceededAt, state.PublishedAt, state.RestoredAt} {
		stamp, known := 0.0, 0.0
		if !at.IsZero() {
			stamp = float64(at.Unix()) + float64(at.Nanosecond())/1e9
			known = 1
		}
		values = append(values, stamp, known)
	}
	for i, value := range values {
		ch <- prometheus.MustNewConstMetric(c.descs[i], prometheus.GaugeValue, value)
	}
}

type snapshotGatherer struct {
	base      prometheus.Gatherer
	snapshots prometheus.Gatherer
}

func (g snapshotGatherer) Gather() ([]*dto.MetricFamily, error) {
	base, err := g.base.Gather()
	if err != nil {
		return nil, err
	}
	// Replace event-updated compatibility gauges only for this app's scrape.
	// Shared registry state is never mutated or re-registered by route setup.
	filtered := make([]*dto.MetricFamily, 0, len(base))
	for _, family := range base {
		if family.GetName() != "warden_snapshot_age_seconds" && family.GetName() != "warden_snapshot_degraded" {
			filtered = append(filtered, family)
		}
	}
	current, err := g.snapshots.Gather()
	if err != nil {
		return nil, err
	}
	return append(filtered, current...), nil
}

// HandlerWithSnapshot computes snapshot gauges on every scrape, even if refresh
// scheduling has stopped. Each handler owns its source; no global callback is swapped.
func HandlerWithSnapshot(sample func() SnapshotSample) http.Handler {
	registry := prometheus.NewRegistry()
	registry.MustRegister(newSnapshotCollector(sample, time.Now))
	return metricskit.HandlerForGatherer(snapshotGatherer{base: Registry.Gatherer(), snapshots: registry})
}
