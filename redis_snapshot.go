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
	snap.Origin = res.Source
	snap.versioned = app.redisSnapshotMode == "shadow" || app.redisSnapshotMode == "v2"
	if app.redisSnapshotMode == "v2" {
		snap.LoadedAt = snap.SourceSucceededAt
	}
	return snap
}

func (app *App) acquireSnapshotLease() (writer bool, release func()) {
	if app.redisSnapshots == nil {
		return false, func() {}
	}
	ctx, cancel := context.WithTimeout(context.Background(), cache.REDIS_OPERATION_TIMEOUT)
	defer cancel()
	lease, err := app.redisSnapshots.Acquire(ctx, redisRefreshLockTTL)
	if err != nil || lease == nil {
		if err != nil {
			app.log.Warn().Err(err).Msg("Failed to acquire snapshot lease; skipping shared update")
		}
		return false, func() {}
	}
	app.snapshotLease = lease
	return true, func() {
		app.snapshotLease = nil
		releaseCtx, releaseCancel := context.WithTimeout(context.Background(), cache.REDIS_OPERATION_TIMEOUT)
		defer releaseCancel()
		if releaseErr := lease.Release(releaseCtx); releaseErr != nil {
			app.log.Warn().Err(releaseErr).Msg("Failed to release snapshot lease")
		}
	}
}

func (app *App) publishRedisSnapshots(users []define.AllowListUser) error {
	if app.redisSnapshotMode != "v2" {
		if err := app.updateRedisCacheWithRetry(users); err != nil {
			return err
		}
		if app.redisSnapshotMode != "shadow" {
			return nil
		}
	}
	if app.snapshotLease == nil {
		return nil
	}
	snap := app.snapshots.Load()
	if snap == nil || snap.SourceSucceededAt.IsZero() {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), cache.REDIS_OPERATION_TIMEOUT)
	defer cancel()
	if snap.RedisRevision != "" {
		return app.snapshotLease.Renew(ctx, snap.RedisRevision)
	}
	published, err := app.snapshotLease.Publish(ctx, &cache.SnapshotEnvelope{
		Users: snap.view.Get(), EffectiveVersion: snap.EffectiveVersion,
		InputVersion: snap.Version, InputCount: snap.Count, Origin: string(snap.Origin), SourceSucceededAt: snap.SourceSucceededAt,
	})
	if err != nil {
		return err
	}
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
	if app.redisSnapshots == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), cache.REDIS_OPERATION_TIMEOUT)
	defer cancel()
	envelope, err := app.redisSnapshots.Read(ctx)
	if err != nil || envelope == nil {
		prommetrics.CacheMisses.Inc()
		return false
	}
	prepared, prepareErr := app.prepareUsers(envelope.Users)
	if prepareErr != nil || prepared.ReadView().GetHash() != envelope.EffectiveVersion ||
		(app.emptyRulesetPolicy.KeepsLastKnownGood() && envelope.InputCount > 0 && prepared.ReadView().Len() == 0) {
		prommetrics.CacheMisses.Inc()
		return false
	}
	app.userCache.ApplyPrepared(prepared)
	app.snapshots.StorePreservingFailures(&Snapshot{
		versioned: true, view: prepared.ReadView(), Users: envelope.Users, EffectiveVersion: envelope.EffectiveVersion,
		EffectiveCount: prepared.ReadView().Len(), Count: envelope.InputCount, Version: envelope.InputVersion,
		Source: loader.SourceRedis, Origin: loader.Source(envelope.Origin), LoadedAt: envelope.SourceSucceededAt,
		SourceSucceededAt: envelope.SourceSucceededAt, PublishedAt: envelope.PublishedAt,
		RestoredAt: app.snapshotTime(), RedisRevision: envelope.Revision, Degraded: true, DegradedReason: "redis_bootstrap",
	})
	app.updateSnapshotMetrics()
	prommetrics.CacheHits.Inc()
	return true
}

// recoverVersionedBaseline fills only an uninitialized V2 replica after a failed
// source attempt. Recovery preserves source age and refresh-failure diagnostics.
func (app *App) recoverVersionedBaseline() bool {
	if app.redisSnapshotMode != "v2" || app.hasKnownGoodSnapshot() ||
		strings.EqualFold(strings.TrimSpace(app.appMode), "ONLY_LOCAL") {
		return false
	}
	return app.bootstrapFromRedis()
}
