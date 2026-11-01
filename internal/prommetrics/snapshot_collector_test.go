package prommetrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

func TestSnapshotCollectorAgesWithoutRefresh(t *testing.T) {
	now := time.Date(2026, 11, 1, 1, 0, 0, 0, time.UTC)
	source := now.Add(-time.Minute)
	calls := 0
	collector := newSnapshotCollector(func() SnapshotSample {
		calls++
		return SnapshotSample{LoadedAt: source, SourceSucceededAt: source, PublishedAt: source.Add(10 * time.Second), RestoredAt: now}
	}, func() time.Time { return now })
	registry := prometheus.NewRegistry()
	registry.MustRegister(collector)
	read := func() map[string]float64 {
		families, err := registry.Gather()
		require.NoError(t, err)
		values := map[string]float64{}
		for _, f := range families {
			values[f.GetName()] = f.Metric[0].GetGauge().GetValue()
		}
		return values
	}
	first := read()
	now = now.Add(30 * time.Second)
	second := read()
	require.Equal(t, 60.0, first["warden_snapshot_age_seconds"])
	require.Equal(t, 90.0, second["warden_snapshot_age_seconds"])
	require.Equal(t, first["warden_snapshot_source_succeeded_timestamp_seconds"], second["warden_snapshot_source_succeeded_timestamp_seconds"])
	require.Equal(t, 1.0, second["warden_snapshot_age_known"])
	require.Equal(t, 2, calls, "one root capture per collection")
}

func TestSnapshotCollectorUnknownAndFutureAge(t *testing.T) {
	now := time.Now()
	sample := SnapshotSample{}
	c := newSnapshotCollector(func() SnapshotSample { return sample }, func() time.Time { return now })
	r := prometheus.NewRegistry()
	r.MustRegister(c)
	families, err := r.Gather()
	require.NoError(t, err)
	for _, f := range families {
		require.Zero(t, f.Metric[0].GetGauge().GetValue(), f.GetName())
	}
	sample.LoadedAt = now.Add(time.Second)
	sample.Degraded = true
	families, err = r.Gather()
	require.NoError(t, err)
	for _, f := range families {
		switch f.GetName() {
		case "warden_snapshot_age_seconds":
			require.Zero(t, f.Metric[0].GetGauge().GetValue())
		case "warden_snapshot_degraded", "warden_snapshot_age_known":
			require.Equal(t, 1.0, f.Metric[0].GetGauge().GetValue())
		}
	}
}

func TestSnapshotCollectorsKeepSourcesIndependentUnderConcurrentScrapes(t *testing.T) {
	now := time.Now()
	var wg sync.WaitGroup
	for _, age := range []time.Duration{time.Minute, 2 * time.Minute} {
		c := newSnapshotCollector(func() SnapshotSample { return SnapshotSample{LoadedAt: now.Add(-age)} }, func() time.Time { return now })
		r := prometheus.NewRegistry()
		r.MustRegister(c)
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				families, err := r.Gather()
				if err != nil {
					t.Error(err)
					return
				}
				for _, f := range families {
					if f.GetName() == "warden_snapshot_age_seconds" && f.Metric[0].GetGauge().GetValue() != age.Seconds() {
						t.Error("collector used another app's source")
					}
				}
			}()
		}
	}
	wg.Wait()
}
