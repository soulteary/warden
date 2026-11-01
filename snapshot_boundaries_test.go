package main

import (
	"fmt"
	"testing"

	"github.com/soulteary/warden/internal/remote"
	"github.com/stretchr/testify/require"
)

func TestRefreshReasonClassification(t *testing.T) {
	for _, tc := range []struct {
		err    error
		reason string
	}{
		{nil, "none"}, {remote.ErrEncryptionRequired, "encryption_required"}, {remote.ErrIntegrityCheckFailed, "integrity_failed"}, {remote.ErrUnsupportedEnvelopeVersion, "unsupported_format"}, {remote.ErrUnsupportedAlgorithm, "unsupported_format"}, {remote.ErrEnvelopeMalformed, "malformed"}, {remote.ErrPlaintextTooLarge, "too_large"}, {fmt.Errorf("no local source"), "no_sources"},
	} {
		var err error
		if tc.err != nil {
			err = fmt.Errorf("load: %w", tc.err)
		}
		require.Equal(t, tc.reason, classifyRefreshReason(err))
	}
}
func TestSnapshotZeroValueBoundaries(t *testing.T) {
	require.Nil(t, freezeSnapshot(nil))
	var store snapshotStore
	require.Nil(t, store.Load())
	failures, reason := store.RefreshFailure()
	require.Zero(t, failures)
	require.Empty(t, reason)
	var missing *snapshotStore
	require.False(t, missing.HasKnownGood())
	(&App{}).updateSnapshotMetrics()
	app := &App{}
	_, err := app.readRedisUserCache()
	require.Error(t, err)
	_, err = app.readRedisUserCacheExists()
	require.Error(t, err)
}
