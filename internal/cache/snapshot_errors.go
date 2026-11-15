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
// the snapshot key holds nothing a replica of this dataset and policy could
// have published or could restore: the key is missing (script result 3), its
// value is over define.MAX_ENVELOPE_JSON_SIZE (result 4), its value is not an
// envelope (result 5: not a JSON object, or an object without this schema
// version, a dataset, a policy and a revision of the expected form), or it is
// an envelope of another dataset (result 6) or policy (result 7). A
// publication of this dataset and policy with a different revision, and a
// value that states a later schema version, (result 2) are not in this set;
// either may be another writer's publication.
func SnapshotRenewalFoundNoPublication(err error) bool {
	return errors.Is(err, ErrSnapshotConflict) && (errors.Is(err, ErrSnapshotMissing) || errors.Is(err, ErrSnapshotInvalid) ||
		errors.Is(err, ErrSnapshotDataset) || errors.Is(err, ErrSnapshotPolicy))
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
