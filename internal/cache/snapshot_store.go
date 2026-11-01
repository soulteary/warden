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
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/soulteary/warden/internal/define"
)

// Snapshot protocol constants follow the project's constant naming convention.
//
//nolint:revive // ALL_CAPS is the project convention
const SNAPSHOT_SCHEMA_V2 = 2

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
}

// NewRedisSnapshotStore rejects implicit datasets and invalid policy fingerprints.
func NewRedisSnapshotStore(client *redis.Client, namespace, policy string) (*RedisSnapshotStore, error) {
	if client == nil || !snapshotNamespace.MatchString(namespace) || !snapshotDigest.MatchString(policy) {
		return nil, ErrSnapshotContract
	}
	key := "warden:{" + namespace + "}:snapshot:v2"
	return &RedisSnapshotStore{client: client, now: time.Now, key: key, lock: key + ":lease", dataset: namespace, policy: policy}, nil
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
	if e.Origin != "local" && e.Origin != "remote" && e.Origin != "merged" {
		return ErrSnapshotInvalid
	}
	return s.validateSnapshotTime(e)
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

var renewSnapshot = redis.NewScript(`
if redis.call('GET', KEYS[2]) ~= ARGV[1] then return 0 end
if redis.call('STRLEN', KEYS[1]) > tonumber(ARGV[4]) then return 4 end
local raw = redis.call('GET', KEYS[1])
if not raw then return 3 end
local ok, snapshot = pcall(cjson.decode, raw)
if not ok or type(snapshot) ~= 'table' then return 5 end
if snapshot.revision ~= ARGV[2] then return 2 end
redis.call('PEXPIRE', KEYS[1], ARGV[3])
return 1`)

// Renew extends only TTL. It never rewrites source time, publication time or users.
// A stale replica may not renew a different publication, even with identical users.
func (l *SnapshotLease) Renew(ctx context.Context, revision string) error {
	if !snapshotToken.MatchString(revision) {
		return ErrSnapshotConflict
	}
	result, err := renewSnapshot.Run(ctx, l.store.client, []string{l.store.key, l.store.lock}, l.token, revision, REDIS_CACHE_TTL.Milliseconds(), define.MAX_ENVELOPE_JSON_SIZE).Int()
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

func (s *RedisSnapshotStore) validateSnapshotTime(e *SnapshotEnvelope) error {
	latest := s.now().Add(5 * time.Second)
	if e.SourceSucceededAt.IsZero() || e.PublishedAt.IsZero() || e.SourceSucceededAt.After(latest) ||
		e.PublishedAt.After(latest) || e.SourceSucceededAt.After(e.PublishedAt.Add(5*time.Second)) {
		return ErrSnapshotTimestamp
	}
	return nil
}
