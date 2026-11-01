package prommetrics

import (
	"errors"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

type failingSnapshotGatherer struct{ err error }

func (g failingSnapshotGatherer) Gather() ([]*dto.MetricFamily, error) { return nil, g.err }
func TestSnapshotGathererReplacesOnlyCompatibilityGauges(t *testing.T) {
	base := prometheus.NewRegistry()
	age := prometheus.NewGauge(prometheus.GaugeOpts{Name: "warden_snapshot_age_seconds", Help: "old age"})
	age.Set(999)
	degraded := prometheus.NewGauge(prometheus.GaugeOpts{Name: "warden_snapshot_degraded", Help: "old state"})
	degraded.Set(1)
	unrelated := prometheus.NewGauge(prometheus.GaugeOpts{Name: "test_unrelated", Help: "retained"})
	unrelated.Set(7)
	base.MustRegister(age, degraded, unrelated)
	current := prometheus.NewRegistry()
	replacement := prometheus.NewGauge(prometheus.GaugeOpts{Name: "warden_snapshot_age_seconds", Help: "current age"})
	replacement.Set(5)
	current.MustRegister(replacement)
	families, err := (snapshotGatherer{base: base, snapshots: current}).Gather()
	require.NoError(t, err)
	values := map[string]float64{}
	for _, family := range families {
		require.NotContains(t, values, family.GetName())
		values[family.GetName()] = family.Metric[0].GetGauge().GetValue()
	}
	require.Equal(t, map[string]float64{"warden_snapshot_age_seconds": 5, "test_unrelated": 7}, values)
	unchanged, err := base.Gather()
	require.NoError(t, err)
	require.Len(t, unchanged, 3)
	sentinel := errors.New("gather failed")
	for _, g := range []snapshotGatherer{{base: failingSnapshotGatherer{sentinel}, snapshots: current}, {base: base, snapshots: failingSnapshotGatherer{sentinel}}} {
		_, err := g.Gather()
		require.ErrorIs(t, err, sentinel)
	}
}
func TestSnapshotHandlerCapturesPerScrapeWithoutMutatingGlobalRegistry(t *testing.T) {
	var calls atomic.Int64
	handler := HandlerWithSnapshot(func() SnapshotSample {
		calls.Add(1)
		return SnapshotSample{LoadedAt: time.Now().Add(-time.Minute), SourceSucceededAt: time.Now().Add(-time.Minute)}
	})
	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", http.NoBody))
		require.Equal(t, 200, rec.Code)
		require.Contains(t, rec.Body.String(), "warden_snapshot_source_time_known 1")
	}
	require.EqualValues(t, 2, calls.Load())
}
