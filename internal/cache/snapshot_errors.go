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

// SnapshotRenewalFoundNoPublication reports whether a renewal failed because
// the snapshot key holds nothing a replica could have published and another
// could restore: the key is missing (script result 3), its value is over
// define.MAX_ENVELOPE_JSON_SIZE (result 4), or its value is not a JSON object
// or array (result 5). A value that decodes and carries a different revision
// (result 2) is not in this set; it may be another writer's publication.
func SnapshotRenewalFoundNoPublication(err error) bool {
	return errors.Is(err, ErrSnapshotConflict) && (errors.Is(err, ErrSnapshotMissing) || errors.Is(err, ErrSnapshotInvalid))
}

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
