package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/cmd"
	"github.com/soulteary/warden/internal/config"
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/loader"
	"github.com/soulteary/warden/internal/prommetrics"
)

func snapshotPolicy(cfg *cmd.Config) string {
	mode, _ := config.ParseMergeMode(cfg.Mode)
	empty, _ := config.ParseEmptyRulesetPolicy(cfg.EmptyRulesetPolicy)
	format := strings.ToLower(strings.TrimSpace(cfg.RemoteEncryptionFormat))
	if format == "" {
		format = "auto"
	}
	// Increment the contract when normalization or validation semantics change.
	//nolint:govet // fieldalignment: logical contract grouping
	policy := struct {
		Contract                                 int
		Merge, Identity, Empty, EncryptionFormat string
		ExplicitID, Decrypt, EncryptionRequired  bool
	}{1, mode.String(), string(define.GetUserIDStrategy()), empty.String(), format,
		define.RequireExplicitUserID(), cfg.RemoteDecryptEnabled && (cfg.RemoteRSAPrivateKeyFile != "" || cfg.RemoteRSAPrivateKey != ""), cfg.RemoteEncryptionRequired}
	body, err := json.Marshal(policy)
	if err != nil {
		panic("snapshot policy contains unsupported fields")
	}
	digest := sha256.Sum256(body)
	return hex.EncodeToString(digest[:])
}

func (app *App) initRedisSnapshots(cfg *cmd.Config) {
	app.redisSnapshotMode = cmd.RedisSnapshotMode(cfg.RedisSnapshotMode)
	if app.redisSnapshotMode == "legacy" {
		return
	}
	if app.redisSnapshotMode != "shadow" && app.redisSnapshotMode != "v2" {
		app.log.Warn().Msg("Invalid Redis snapshot mode; shared writes disabled")
		app.redisSnapshotMode = "v2"
		return
	}
	if app.redisClient == nil {
		return
	}
	store, err := cache.NewRedisSnapshotStore(app.redisClient, cfg.RedisSnapshotNamespace, snapshotPolicy(cfg))
	if err != nil {
		app.log.Warn().Err(err).Msg("Versioned Redis snapshots unavailable")
		return
	}
	app.redisSnapshots = store
}

func (app *App) snapshotTime() time.Time {
	if app.snapshotNow != nil {
		return app.snapshotNow().UTC()
	}
	return time.Now().UTC()
}

func (app *App) sourceSnapshot(res *loader.LoadResult, view *cache.ReadView) *Snapshot {
	snap := snapshotFromResult(res, view)
	snap.SourceSucceededAt = app.snapshotTime()
	if res.Err == nil && res.Manifests != nil && view != nil {
		snap.acceptedSources = cloneAcceptedSources(&acceptedSourceEvidence{Generation: acceptedSourceGeneration.Add(1), AcceptedAt: snap.SourceSucceededAt, SourceSetID: res.SourceSetID, InputVersion: res.Version, EffectiveVersion: view.GetHash(), Manifests: res.Manifests, Attempts: res.Attempts})
	}
	snap.Origin = res.Source
	snap.versioned = app.redisSnapshotMode == "shadow" || app.redisSnapshotMode == "v2"
	if app.redisSnapshotMode == "v2" {
		snap.LoadedAt = snap.SourceSucceededAt
	}
	return snap
}

func (app *App) acquireSnapshotLease() (writer bool, release func()) {
	if app.redisSnapshots == nil {
		app.recordSnapshotResult("lease_acquire", "unavailable")
		return false, func() {}
	}
	ctx, cancel := context.WithTimeout(context.Background(), cache.REDIS_OPERATION_TIMEOUT)
	defer cancel()
	lease, err := app.redisSnapshots.Acquire(ctx, redisRefreshLockTTL)
	if err != nil || lease == nil {
		if err == nil {
			app.recordSnapshotResult("lease_acquire", "busy")
		} else {
			app.recordSnapshotResult("lease_acquire", cache.SnapshotFailureReason(err))
		}
		if err != nil {
			app.log.Warn().Err(err).Msg("Failed to acquire snapshot lease; skipping shared update")
		}
		return false, func() {}
	}
	app.recordSnapshotResult("lease_acquire", "acquired")
	app.snapshotLease = lease
	return true, func() {
		app.snapshotLease = nil
		releaseCtx, releaseCancel := context.WithTimeout(context.Background(), cache.REDIS_OPERATION_TIMEOUT)
		defer releaseCancel()
		if releaseErr := lease.Release(releaseCtx); releaseErr != nil {
			app.recordSnapshotResult("lease_release", cache.SnapshotFailureReason(releaseErr))
			app.log.Warn().Err(releaseErr).Msg("Failed to release snapshot lease")
		} else {
			app.recordSnapshotResult("lease_release", "released")
		}
	}
}

func (app *App) publishRedisSnapshots(users []define.AllowListUser) error {
	if app.redisSnapshotMode != "v2" {
		if err := app.updateRedisCacheWithRetry(users); err != nil {
			app.recordSnapshotResult("publish", "legacy_write_error")
			return err
		}
		if app.redisSnapshotMode != "shadow" {
			return nil
		}
	}
	if app.snapshotLease == nil {
		app.recordSnapshotResult("publish", "no_lease")
		return nil
	}
	snap := app.snapshots.Load()
	if snap == nil || snap.SourceSucceededAt.IsZero() {
		app.recordSnapshotResult("publish", "unknown_source_time")
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), cache.REDIS_OPERATION_TIMEOUT)
	defer cancel()
	if snap.RedisRevision != "" {
		err := app.snapshotLease.Renew(ctx, snap.RedisRevision)
		result := "renewed"
		if err != nil {
			result = cache.SnapshotFailureReason(err)
		}
		app.recordSnapshotResult("renew", result)
		return err
	}
	published, err := app.snapshotLease.Publish(ctx, &cache.SnapshotEnvelope{
		Users: snap.view.Get(), EffectiveVersion: snap.EffectiveVersion,
		InputVersion: snap.Version, InputCount: snap.Count, Origin: string(snap.Origin), SourceSucceededAt: snap.SourceSucceededAt,
	})
	if err != nil {
		app.recordSnapshotResult("publish", cache.SnapshotFailureReason(err))
		return err
	}
	app.recordSnapshotResult("publish", "published")
	app.snapshots.recordPublication(snap, published.PublishedAt, published.Revision)
	return nil
}

// Publication metadata joins the same generation without erasing concurrent diagnostics.
func (s *snapshotStore) recordPublication(expected *Snapshot, at time.Time, revision string) {
	for {
		previous := s.capture()
		if previous == nil || previous.snapshot != expected {
			return
		}
		snap := *expected
		snap.PublishedAt, snap.RedisRevision = at, revision
		next := &snapshotState{snapshot: &snap, reason: previous.reason, failures: previous.failures}
		if s.current.CompareAndSwap(previous, next) {
			return
		}
	}
}

func (app *App) bootstrapVersionedSnapshot() bool {
	return app.restoreVersionedSnapshot(nil)
}

// expected is non-nil only for opt-in follow recovery. Background callers hold
// refreshMu; the compare-and-swap additionally fences publication changes.
func (app *App) restoreVersionedSnapshot(expected *Snapshot) bool {
	operation := "read"
	if expected != nil {
		operation = "follow_read"
	}
	if app.redisSnapshots == nil {
		app.recordSnapshotResult(operation, "unavailable")
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), cache.REDIS_OPERATION_TIMEOUT)
	defer cancel()
	envelope, err := app.redisSnapshots.Read(ctx)
	if err != nil || envelope == nil {
		result := "missing"
		if err != nil {
			result = cache.SnapshotFailureReason(err)
		}
		app.recordSnapshotResult(operation, result)
		if err != nil && app.log != nil {
			app.log.Warn().Str("result", result).Msg("Versioned Redis snapshot recovery rejected")
		}
		prommetrics.CacheMisses.Inc()
		return false
	}
	prepared, prepareErr := app.prepareUsers(envelope.Users)
	if prepareErr != nil || prepared.ReadView().GetHash() != envelope.EffectiveVersion ||
		(app.emptyRulesetPolicy.KeepsLastKnownGood() && envelope.InputCount > 0 && prepared.ReadView().Len() == 0) {
		app.recordSnapshotResult(operation, "effective_rejected")
		prommetrics.CacheMisses.Inc()
		return false
	}
	restored := &Snapshot{
		versioned: true, view: prepared.ReadView(), Users: envelope.Users, EffectiveVersion: envelope.EffectiveVersion,
		EffectiveCount: prepared.ReadView().Len(), Count: envelope.InputCount, Version: envelope.InputVersion,
		Source: loader.SourceRedis, Origin: loader.Source(envelope.Origin), LoadedAt: envelope.SourceSucceededAt,
		SourceSucceededAt: envelope.SourceSucceededAt, PublishedAt: envelope.PublishedAt,
		RestoredAt: app.snapshotTime(), RedisRevision: envelope.Revision, Degraded: true, DegradedReason: "redis_bootstrap",
	}
	if expected != nil {
		now := app.snapshotTime()
		current := captureHealthState(app.snapshots, app.userCache)
		current.observedAt = now
		candidate := &healthReadState{state: &snapshotState{snapshot: restored}, view: restored.view, observedAt: now}
		if current.state.snapshot != expected || evaluateSnapshotRead(current, app.snapshotMaxAge).reason != readSourceExpired ||
			!restored.SourceSucceededAt.After(expected.SourceSucceededAt) || !snapshotReadFresh(candidate, app.snapshotMaxAge) {
			app.recordSnapshotResult(operation, "ineligible")
			return false
		}
		restored.DegradedReason = "redis_follow_recovery"
		if !app.snapshots.restoreIfCurrent(expected, restored) {
			app.recordSnapshotResult(operation, "superseded")
			return false
		}
		app.userCache.ApplyPrepared(prepared)
	} else {
		app.userCache.ApplyPrepared(prepared)
		app.snapshots.StorePreservingFailures(restored)
	}
	app.updateSnapshotMetrics()
	prommetrics.CacheHits.Inc()
	app.recordSnapshotResult(operation, "restored")
	return true
}

func (app *App) recordSnapshotResult(operation, result string) {
	if app.redisSnapshotMode == "shadow" || app.redisSnapshotMode == "v2" {
		prommetrics.RecordRedisSnapshotResult(app.redisSnapshotMode, operation, result)
	}
}

// recoverVersionedBaseline retains bootstrap recovery by default. The optional
// stale policy follows Redis only after a failed refresh of an expired local set.
func (app *App) recoverVersionedBaseline() bool {
	if app.redisSnapshotMode != "v2" || strings.EqualFold(strings.TrimSpace(app.appMode), "ONLY_LOCAL") {
		return false
	}
	if !app.hasKnownGoodSnapshot() {
		return app.bootstrapFromRedis()
	}
	if app.redisSnapshotRecovery != "stale" || app.redisStartupFailed {
		return false
	}
	pinned := captureHealthState(app.snapshots, app.userCache)
	pinned.observedAt = app.snapshotTime()
	if pinned.state.failures == 0 || evaluateSnapshotRead(pinned, app.snapshotMaxAge).reason != readSourceExpired {
		return false
	}
	return app.restoreVersionedSnapshot(pinned.state.snapshot)
}

// restoreIfCurrent keeps concurrent failure diagnostics, but never replaces a
// different publication, including a successful source refresh.
func (s *snapshotStore) restoreIfCurrent(expected, recovered *Snapshot) bool {
	frozen := freezeSnapshot(recovered)
	for {
		previous := s.capture()
		if previous == nil || previous.snapshot != expected {
			return false
		}
		next := &snapshotState{snapshot: frozen, reason: previous.reason, failures: previous.failures}
		if s.current.CompareAndSwap(previous, next) {
			return true
		}
	}
}
