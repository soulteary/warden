package cache

import (
	"context"
	"github.com/soulteary/warden/internal/define"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSnapshotStoreConnectionFailureContract(t *testing.T) {
	store, mr := snapshotStoreFixture(t)
	ctx := context.Background()
	lease, err := store.Acquire(ctx, time.Minute)
	require.NoError(t, err)
	mr.Close()
	_, err = store.Read(ctx)
	require.Error(t, err)
	_, err = store.Acquire(ctx, time.Minute)
	require.Error(t, err)
	_, err = lease.Publish(ctx, sourceSnapshot(time.Now()))
	require.Error(t, err)
	require.Error(t, lease.Renew(ctx, "not-a-revision"))
	require.Error(t, lease.Release(ctx))
}
func TestSnapshotStoreRejectsTrailingJSONAndInvalidOrigin(t *testing.T) {
	store, mr := snapshotStoreFixture(t)
	require.NoError(t, mr.Set(store.key, `{} {}`))
	_, err := store.Read(context.Background())
	require.ErrorIs(t, err, ErrSnapshotInvalid)
	lease, err := store.Acquire(context.Background(), time.Minute)
	require.NoError(t, err)
	input := sourceSnapshot(time.Now())
	input.Origin = "unknown"
	_, err = lease.Publish(context.Background(), input)
	require.ErrorIs(t, err, ErrSnapshotInvalid)
	require.NoError(t, lease.Release(context.Background()))
}

func TestSnapshotStoreRejectsInvalidLeaseAndNilPublication(t *testing.T) {
	store, _ := snapshotStoreFixture(t)
	require.True(t, ValidateSnapshotNamespace("dataset"))
	require.False(t, ValidateSnapshotNamespace("invalid namespace"))
	_, err := store.Acquire(context.Background(), 0)
	require.ErrorIs(t, err, ErrSnapshotInvalid)
	lease, err := store.Acquire(context.Background(), time.Minute)
	require.NoError(t, err)
	_, err = lease.Publish(context.Background(), nil)
	require.ErrorIs(t, err, ErrSnapshotInvalid)
	require.NoError(t, lease.Release(context.Background()))
}
func TestEmptyRecordValidationDoesNotPublish(t *testing.T) {
	require.Error(t, validateUser(define.AllowListUser{}))
	require.Nil(t, cloneUsers(nil))
}
