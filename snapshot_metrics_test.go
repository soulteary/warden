package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/prommetrics"
	"github.com/stretchr/testify/require"
)

func TestSnapshotMetricSampleUsesPublishedGenerationAndOriginalTimes(t *testing.T) {
	app, _ := migrationFixture(t, "v2")
	source := time.Now().Add(-10 * time.Minute)
	pub := publishMigrationSource(t, app, source, []define.AllowListUser{{Mail: "member@example.com"}})
	app.snapshots.RecordRefreshFailure("timeout")
	sample := app.snapshotMetricSample()
	require.Equal(t, pub.SourceSucceededAt, sample.LoadedAt)
	require.Equal(t, pub.SourceSucceededAt, sample.SourceSucceededAt)
	require.Equal(t, pub.PublishedAt, sample.PublishedAt)
	require.True(t, sample.Degraded)
	// The original store remains attached to the handler after restoration.
	handler := prommetrics.HandlerWithSnapshot(app.snapshotMetricSample)
	require.True(t, app.bootstrapFromRedis())
	require.Equal(t, pub.SourceSucceededAt, app.snapshotMetricSample().LoadedAt)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", http.NoBody))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 1, strings.Count(rec.Body.String(), "\nwarden_snapshot_age_seconds "))
	require.Contains(t, rec.Body.String(), "warden_snapshot_source_time_known 1")
	require.Contains(t, rec.Body.String(), "warden_snapshot_restoration_time_known 1")
}

func TestSnapshotMetricSampleKeepsLegacyAgeBasis(t *testing.T) {
	app, _ := migrationFixture(t, "legacy")
	source := time.Now().Add(-time.Hour)
	loaded := time.Now().Add(-time.Minute)
	app.snapshots.Store(&Snapshot{LoadedAt: loaded, SourceSucceededAt: source})
	sample := app.snapshotMetricSample()
	require.Equal(t, loaded, sample.LoadedAt)
	require.Equal(t, source, sample.SourceSucceededAt)
	require.Zero(t, (&App{}).snapshotMetricSample())
}
