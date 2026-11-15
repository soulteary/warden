package cache

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/soulteary/warden/internal/define"
)

// Snapshot protocol constants follow the project's constant naming convention.
//
//nolint:revive // ALL_CAPS is the project convention
const SNAPSHOT_SCHEMA_V2 = 2

// SNAPSHOT_FUTURE_ALLOWANCE is how far ahead of the local clock a stored source
// or publication time may be. Replicas share one snapshot but not one clock, so
// the store and the read decision must apply the same bound: a snapshot the
// store accepts is one a reader may serve.
//
//nolint:revive // ALL_CAPS is the project convention
const SNAPSHOT_FUTURE_ALLOWANCE = 5 * time.Second

// Versioned snapshot errors never include users, keys or raw payloads.
var (
	ErrSnapshotInvalid   = errors.New("invalid versioned snapshot")
	ErrSnapshotContract  = errors.New("snapshot contract mismatch")
	ErrSnapshotLeaseLost = errors.New("snapshot writer lease lost")
	ErrSnapshotConflict  = errors.New("snapshot revision changed")
)

var snapshotNamespace = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9:_-]{0,63}$`)
var snapshotDigest = regexp.MustCompile(`^[a-f0-9]{64}$`)
var snapshotToken = regexp.MustCompile(`^[a-f0-9]{32}$`)

// SnapshotEnvelope stores effective data and its original validation evidence.
// RestoredAt is intentionally absent: recovery is a process-local event.
//
//nolint:govet // fieldalignment: preserve the protocol's logical field grouping
type SnapshotEnvelope struct {
	SchemaVersion     int                    `json:"schema_version"`
	Dataset           string                 `json:"dataset"`
	Policy            string                 `json:"policy"`
	Revision          string                 `json:"revision"`
	Users             []define.AllowListUser `json:"users"`
	EffectiveVersion  string                 `json:"effective_version"`
	InputVersion      string                 `json:"input_version"`
	InputCount        int                    `json:"input_count"`
	Origin            string                 `json:"origin"`
	SourceSucceededAt time.Time              `json:"source_succeeded_at"`
	PublishedAt       time.Time              `json:"published_at"`
}

// RedisSnapshotStore uses isolated, co-located Redis Cluster keys for schema v2.
type RedisSnapshotStore struct {
	client  *redis.Client
	now     func() time.Time
	key     string
	lock    string
	dataset string
	policy  string
	// head is every byte Publish writes before the revision's value.
	head string
}

// NewRedisSnapshotStore rejects implicit datasets and invalid policy fingerprints.
func NewRedisSnapshotStore(client *redis.Client, namespace, policy string) (*RedisSnapshotStore, error) {
	if client == nil || !snapshotNamespace.MatchString(namespace) || !snapshotDigest.MatchString(policy) {
		return nil, ErrSnapshotContract
	}
	key := "warden:{" + namespace + "}:snapshot:v2"
	return &RedisSnapshotStore{client: client, now: time.Now, key: key, lock: key + ":lease", dataset: namespace, policy: policy, head: envelopeHead(namespace, policy)}, nil
}

// envelopeHead returns the bytes with which every envelope Publish writes for
// this dataset and policy begins, up to the opening quote of the revision.
// encoding/json writes struct fields in declaration order without whitespace,
// and SnapshotEnvelope has had the same first five fields since the format was
// introduced. The namespace and policy patterns admit no character JSON
// escapes, so both appear verbatim.
func envelopeHead(namespace, policy string) string {
	return `{"schema_version":` + strconv.Itoa(SNAPSHOT_SCHEMA_V2) + `,"dataset":"` + namespace + `","policy":"` + policy + `","revision":"`
}

// ValidateSnapshotNamespace checks an operator-selected dataset without exposing it.
func ValidateSnapshotNamespace(namespace string) bool {
	return snapshotNamespace.MatchString(namespace)
}

var readSnapshot = redis.NewScript(`
if redis.call('STRLEN', KEYS[1]) > tonumber(ARGV[1]) then
  return redis.error_reply('snapshot exceeds size limit')
end
return redis.call('GET', KEYS[1])`)

// Read atomically distinguishes a missing key from a stored, valid empty set.
// Oversized values are rejected in Redis before being transferred to the client.
func (s *RedisSnapshotStore) Read(ctx context.Context) (*SnapshotEnvelope, error) {
	raw, err := readSnapshot.Run(ctx, s.client, []string{s.key}, define.MAX_ENVELOPE_JSON_SIZE).Text()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		if redis.HasErrorPrefix(err, "snapshot exceeds size limit") {
			return nil, ErrSnapshotTooLarge
		}
		return nil, fmt.Errorf("read versioned snapshot: %w", err)
	}
	var envelope SnapshotEnvelope
	decoder := json.NewDecoder(bytes.NewBufferString(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&envelope); err != nil {
		return nil, ErrSnapshotInvalid
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return nil, ErrSnapshotInvalid
	}
	if err := s.validate(&envelope); err != nil {
		return nil, err
	}
	return &envelope, nil
}

func (s *RedisSnapshotStore) validate(e *SnapshotEnvelope) error {
	if e.SchemaVersion != SNAPSHOT_SCHEMA_V2 {
		return ErrSnapshotSchema
	}
	if e.Dataset != s.dataset {
		return ErrSnapshotDataset
	}
	if e.Policy != s.policy {
		return ErrSnapshotPolicy
	}
	if e.Users == nil || e.InputCount < len(e.Users) || !snapshotDigest.MatchString(e.InputVersion) ||
		!snapshotDigest.MatchString(e.EffectiveVersion) || !snapshotToken.MatchString(e.Revision) ||
		e.EffectiveVersion != HashUserList(e.Users) {
		return ErrSnapshotInvalid
	}
	return s.validateSnapshotTimes(e)
}

// SnapshotLease authorizes writes only while its Redis owner token still matches.
type SnapshotLease struct {
	store *RedisSnapshotStore
	token string
}

func newSnapshotToken() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}

// Acquire returns nil when another replica holds the bounded writer lease.
// It never falls back to a process-local lock.
func (s *RedisSnapshotStore) Acquire(ctx context.Context, ttl time.Duration) (*SnapshotLease, error) {
	if ttl <= 0 {
		return nil, ErrSnapshotInvalid
	}
	token, err := newSnapshotToken()
	if err != nil {
		return nil, err
	}
	acquired, err := s.client.SetNX(ctx, s.lock, token, ttl).Result()
	if err != nil || !acquired {
		return nil, err
	}
	return &SnapshotLease{store: s, token: token}, nil
}

var publishSnapshot = redis.NewScript(`
if redis.call('GET', KEYS[2]) ~= ARGV[1] then return 0 end
redis.call('SET', KEYS[1], ARGV[2], 'PX', ARGV[3])
return 1`)

// Publish writes one complete envelope, assigning publication time and revision.
// A slow writer whose lease expired cannot overwrite the next owner's result.
func (l *SnapshotLease) Publish(ctx context.Context, input *SnapshotEnvelope) (*SnapshotEnvelope, error) {
	if input == nil {
		return nil, ErrSnapshotInvalid
	}
	envelope := *input
	envelope.Users = cloneUsers(input.Users)
	envelope.SourceSucceededAt = input.SourceSucceededAt.UTC()
	envelope.SchemaVersion = SNAPSHOT_SCHEMA_V2
	envelope.Dataset = l.store.dataset
	envelope.Policy = l.store.policy
	envelope.PublishedAt = l.store.now().UTC()
	revision, err := newSnapshotToken()
	if err != nil {
		return nil, err
	}
	envelope.Revision = revision
	if validationErr := l.store.validate(&envelope); validationErr != nil {
		return nil, validationErr
	}
	body, err := json.Marshal(envelope)
	if err != nil {
		return nil, ErrSnapshotInvalid
	}
	if len(body) > define.MAX_ENVELOPE_JSON_SIZE {
		return nil, ErrSnapshotTooLarge
	}
	written, err := publishSnapshot.Run(ctx, l.store.client, []string{l.store.key, l.store.lock}, l.token, body, REDIS_CACHE_TTL.Milliseconds()).Int()
	if err != nil {
		return nil, fmt.Errorf("publish versioned snapshot: %w", err)
	}
	if written != 1 {
		return nil, ErrSnapshotLeaseLost
	}
	return &envelope, nil
}

// renewSnapshot classifies the stored value before it compares revisions.
//
// A value is a publication of this dataset and policy when it begins exactly
// as an envelope written by Publish does (ARGV[5], then 32 characters of the
// revision alphabet, then the start of the users member), or when it decodes
// to an object whose schema_version, dataset and policy are this store's and
// whose revision has the form of a revision. Only such a value is compared by
// revision: 1 renews it, 2 leaves another writer's publication untouched.
//
// A value that states a later schema_version is also answered with 2. This
// release cannot judge it, and a newer release may be writing under the same
// key during an upgrade.
//
// Everything else is not a publication a replica of this dataset and policy
// could restore, and the caller may replace it: 3 no key, 4 over the size
// limit, 5 not an envelope, 6 an envelope of another dataset, 7 an envelope
// of another policy.
//
// The comparison with ARGV[5] is anchored at the first byte, so it ends before
// the users array begins and nothing inside user records can take part in it.
// cjson.decode of a complete envelope builds every user in Lua and blocks
// Redis for that long; it remains only for a value with any other beginning,
// which no release writes.
var renewSnapshot = redis.NewScript(`
if redis.call('GET', KEYS[2]) ~= ARGV[1] then return 0 end
if redis.call('STRLEN', KEYS[1]) > tonumber(ARGV[4]) then return 4 end
local raw = redis.call('GET', KEYS[1])
if not raw then return 3 end
local function token(value)
  return type(value) == 'string' and #value == 32 and string.find(value, '^[a-f0-9]+$') ~= nil
end
local schema = tonumber(ARGV[6])
local head = #ARGV[5]
local first = string.sub(raw, 1, head + 42)
local revision = string.sub(first, head + 1, head + 32)
if string.sub(first, 1, head) ~= ARGV[5] or string.sub(first, head + 33) ~= '","users":' or not token(revision) then
  local stated = string.match(first, '^{"schema_version":(%d+)[,}]')
  if stated and tonumber(stated) > schema then return 2 end
  local ok, snapshot = pcall(cjson.decode, raw)
  if not ok or type(snapshot) ~= 'table' then return 5 end
  stated = snapshot.schema_version
  if type(stated) == 'number' and stated > schema then return 2 end
  if stated ~= schema or type(snapshot.dataset) ~= 'string' or type(snapshot.policy) ~= 'string' then return 5 end
  if snapshot.dataset ~= ARGV[7] then return 6 end
  if snapshot.policy ~= ARGV[8] then return 7 end
  revision = snapshot.revision
  if not token(revision) then return 5 end
end
if revision ~= ARGV[2] then return 2 end
redis.call('PEXPIRE', KEYS[1], ARGV[3])
return 1`)

// Renew extends only TTL. It never rewrites source time, publication time or users.
// A stale replica may not renew a different publication, even with identical users.
func (l *SnapshotLease) Renew(ctx context.Context, revision string) error {
	if !snapshotToken.MatchString(revision) {
		return ErrSnapshotConflict
	}
	result, err := renewSnapshot.Run(ctx, l.store.client, []string{l.store.key, l.store.lock}, l.token, revision, REDIS_CACHE_TTL.Milliseconds(), define.MAX_ENVELOPE_JSON_SIZE, l.store.head, SNAPSHOT_SCHEMA_V2, l.store.dataset, l.store.policy).Int()
	if err != nil {
		return fmt.Errorf("renew versioned snapshot: %w", err)
	}
	if result == 0 {
		return ErrSnapshotLeaseLost
	}
	switch result {
	case 3:
		return ErrSnapshotMissing
	case 4:
		return fmt.Errorf("%w: %w", ErrSnapshotConflict, ErrSnapshotTooLarge)
	case 5:
		return fmt.Errorf("%w: %w", ErrSnapshotConflict, ErrSnapshotInvalid)
	case 6:
		return fmt.Errorf("%w: %w", ErrSnapshotConflict, ErrSnapshotDataset)
	case 7:
		return fmt.Errorf("%w: %w", ErrSnapshotConflict, ErrSnapshotPolicy)
	}
	if result != 1 {
		return ErrSnapshotConflict
	}
	return nil
}

var releaseSnapshot = redis.NewScript(`
if redis.call('GET', KEYS[1]) == ARGV[1] then return redis.call('DEL', KEYS[1]) end
return 0`)

// Release cannot delete a successor's lease after expiry.
func (l *SnapshotLease) Release(ctx context.Context) error {
	return releaseSnapshot.Run(ctx, l.store.client, []string{l.store.lock}, l.token).Err()
}

func (s *RedisSnapshotStore) validateSnapshotTimes(e *SnapshotEnvelope) error {
	if e.Origin != "local" && e.Origin != "remote" && e.Origin != "merged" {
		return ErrSnapshotInvalid
	}
	latest := s.now().Add(SNAPSHOT_FUTURE_ALLOWANCE)
	if e.SourceSucceededAt.IsZero() || e.PublishedAt.IsZero() || e.SourceSucceededAt.After(latest) ||
		e.PublishedAt.After(latest) || e.SourceSucceededAt.After(e.PublishedAt.Add(SNAPSHOT_FUTURE_ALLOWANCE)) {
		return ErrSnapshotTimestamp
	}
	return nil
}
