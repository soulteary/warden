package cache

import (
	"errors"
	"fmt"
)

// Detailed snapshot failures retain the existing broad errors for errors.Is.
var (
	ErrSnapshotSchema    = fmt.Errorf("%w: unsupported schema", ErrSnapshotContract)
	ErrSnapshotDataset   = fmt.Errorf("%w: dataset mismatch", ErrSnapshotContract)
	ErrSnapshotPolicy    = fmt.Errorf("%w: policy mismatch", ErrSnapshotContract)
	ErrSnapshotTimestamp = fmt.Errorf("%w: timestamp invalid", ErrSnapshotInvalid)
	ErrSnapshotTooLarge  = fmt.Errorf("%w: size limit", ErrSnapshotInvalid)
	ErrSnapshotMissing   = fmt.Errorf("%w: snapshot missing", ErrSnapshotConflict)
)

// SnapshotFailureReason is a bounded diagnostic code, never an error message.
func SnapshotFailureReason(err error) string {
	switch {
	case errors.Is(err, ErrSnapshotSchema):
		return "schema_rejected"
	case errors.Is(err, ErrSnapshotDataset):
		return "dataset_rejected"
	case errors.Is(err, ErrSnapshotPolicy):
		return "policy_rejected"
	case errors.Is(err, ErrSnapshotTimestamp):
		return "timestamp_rejected"
	case errors.Is(err, ErrSnapshotTooLarge):
		return "too_large"
	case errors.Is(err, ErrSnapshotMissing):
		return "missing"
	case errors.Is(err, ErrSnapshotInvalid):
		return "invalid_envelope"
	case errors.Is(err, ErrSnapshotContract):
		return "contract_rejected"
	case errors.Is(err, ErrSnapshotLeaseLost):
		return "lease_lost"
	case errors.Is(err, ErrSnapshotConflict):
		return "revision_conflict"
	default:
		return "io_error"
	}
}
