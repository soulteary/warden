package cache

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/soulteary/warden/internal/define"
	"github.com/stretchr/testify/require"
)

const usersMember = `","users":`

// The renewal reads the revision at a fixed position only because every
// envelope begins with the same bytes. This fails if the envelope's leading
// fields are reordered, renamed or encoded differently; the script then falls
// back to decoding, so the result stays correct and only the cost returns.
func TestEnvelopeHeadIsTheBeginningOfEveryPublishedEnvelope(t *testing.T) {
	store, mr := snapshotStoreFixture(t)
	ctx := context.Background()
	lease, err := store.Acquire(ctx, time.Minute)
	require.NoError(t, err)
	published, err := lease.Publish(ctx, sourceSnapshot(time.Now()))
	require.NoError(t, err)
	raw, err := mr.Get(store.key)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(raw, store.head+published.Revision+usersMember+"["), "an envelope begins with the head, the revision and the users member")
	require.Equal(t, `{"schema_version":2,"dataset":"test-dataset","policy":"`+strings.Repeat("a", 64)+`","revision":"`, store.head)
}

// The envelope as releases since v1.5.0 store it, written out by hand so the
// test does not depend on the current struct.
func storedEnvelopeLayout(revision, users string) string {
	digest := strings.Repeat("b", 64)
	return `{"schema_version":2,"dataset":"test-dataset","policy":"` + strings.Repeat("a", 64) + `","revision":"` + revision +
		`","users":` + users + `,"effective_version":"` + digest + `","input_version":"` + digest +
		`","input_count":1,"origin":"remote","source_succeeded_at":"2026-01-01T12:00:00Z","published_at":"2026-01-01T12:00:01Z"}`
}

func renewCases(t *testing.T, store *RedisSnapshotStore, client *redis.Client) {
	t.Helper()
	ctx := context.Background()
	lease, err := store.Acquire(ctx, time.Minute)
	require.NoError(t, err)
	require.NotNil(t, lease)
	held, other := strings.Repeat("1", 32), strings.Repeat("2", 32)
	head := envelopeHead(store.dataset, store.policy)
	layout := strings.ReplaceAll(storedEnvelopeLayout(held, `[{"phone":"","mail":"member@example.com","user_id":"member","status":"active","scope":["read"],"role":""}]`), `"dataset":"test-dataset"`, `"dataset":"`+store.dataset+`"`)
	require.True(t, strings.HasPrefix(layout, head+held+usersMember))
	var indented bytes.Buffer
	require.NoError(t, json.Indent(&indented, []byte(layout), "", "  "))
	revisionLast := strings.TrimSuffix(strings.Replace(layout, `"revision":"`+held+`",`, "", 1), "}") + `,"revision":"` + held + `"}`

	otherPolicy := strings.Replace(layout, store.policy, strings.Repeat("c", 64), 1)
	otherDataset := strings.Replace(layout, `"dataset":"`+store.dataset+`"`, `"dataset":"another-dataset"`, 1)
	member := func(name, value string) string {
		return strings.Replace(layout, `"`+name+`":`, `"`+name+`":`+value+`,"was_`+name+`":`, 1)
	}
	without := func(name string) string {
		return strings.Replace(layout, `"`+name+`":`, `"was_`+name+`":`, 1)
	}

	renewed, conflict, invalid, dataset, policy := "renewed", "conflict", "invalid", "dataset", "policy"
	for name, tc := range map[string]struct{ stored, heldOutcome, otherOutcome string }{
		// Values that begin as a published envelope does: no decode.
		"stored_layout":            {layout, renewed, conflict},
		"revision_text_in_a_user":  {storedEnvelopeLayout(held, `[{"mail":"x","name":"\"revision\":\"`+other+`\",\"users\":"}]`), renewed, conflict},
		"revision_member_in_users": {head + held + usersMember + `[{"revision":"` + other + `","users":[]}]}`, renewed, conflict},
		"head_repeated_in_users":   {head + held + usersMember + `["` + strings.ReplaceAll(head+other+usersMember, `"`, `\"`) + `"]}`, renewed, conflict},
		// Values with another beginning are decoded. An object with this
		// schema version, dataset and policy and a revision of the expected
		// form is a publication and is compared by revision.
		"indented":             {indented.String(), renewed, conflict},
		"revision_last":        {revisionLast, renewed, conflict},
		"leading_space":        {" " + layout, renewed, conflict},
		"revision_then_others": {head + held + `","other":1,"users":[]}`, renewed, conflict},
		// A value that states a later schema version is left to the release
		// that wrote it, whatever else it holds.
		"later_schema":             {member("schema_version", "3"), conflict, conflict},
		"later_schema_alone":       {`{"schema_version":3}`, conflict, conflict},
		"later_schema_other_shape": {`{"schema_version":12,"snapshot":{"id":"` + held + `"}}`, conflict, conflict},
		"later_schema_not_decoded": {`{"schema_version":3,"dataset":"` + store.dataset + `","users":[ not decoded`, conflict, conflict},
		"later_schema_spaced":      {`{ "schema_version" : 3, "revision" : "` + held + `" }`, conflict, conflict},
		"later_schema_fraction":    {`{"schema_version":2.5}`, conflict, conflict},
		// An envelope of another dataset or policy is reported as such.
		"other_policy":  {otherPolicy, policy, policy},
		"other_dataset": {otherDataset, dataset, dataset},
		// Anything else is not an envelope.
		"empty_object":         {`{}`, invalid, invalid},
		"empty_array":          {`[]`, invalid, invalid},
		"array_of_envelopes":   {`[` + layout + `]`, invalid, invalid},
		"revision_only":        {`{"revision":"` + held + `"}`, invalid, invalid},
		"no_schema_version":    {without("schema_version"), invalid, invalid},
		"earlier_schema":       {member("schema_version", "1"), invalid, invalid},
		"schema_zero":          {member("schema_version", "0"), invalid, invalid},
		"schema_as_text":       {member("schema_version", `"2"`), invalid, invalid},
		"later_schema_as_text": {member("schema_version", `"3"`), invalid, invalid},
		"no_dataset":           {without("dataset"), invalid, invalid},
		"dataset_not_text":     {member("dataset", "7"), invalid, invalid},
		"no_policy":            {without("policy"), invalid, invalid},
		"policy_not_text":      {member("policy", "[]"), invalid, invalid},
		"no_revision":          {without("revision"), invalid, invalid},
		"revision_null":        {member("revision", "null"), invalid, invalid},
		"revision_number":      {member("revision", "7"), invalid, invalid},
		"revision_object":      {member("revision", `{"value":"`+held+`"}`), invalid, invalid},
		"revision_upper_case":  {strings.Replace(strings.ReplaceAll(layout, held, strings.Repeat("A", 32)), "{", "{ ", 1), invalid, invalid},
		"head_upper_case":      {head + strings.Repeat("A", 32) + usersMember + `[]}`, invalid, invalid},
		"head_not_hexadecimal": {head + strings.Repeat("g", 32) + usersMember + `[]}`, invalid, invalid},
		"head_quote_in_place":  {head + strings.Repeat("1", 31) + `"` + usersMember + `[]}`, invalid, invalid},
		"short_revision":       {head + held[:31] + usersMember + `[]}`, invalid, invalid},
		"long_revision":        {head + held + "1" + usersMember + `[]}`, invalid, invalid},
		"not_json":             {`{`, invalid, invalid},
		"head_only":            {head, invalid, invalid},
		"head_and_revision":    {head + held, invalid, invalid},
		"schema_head_only":     {`{"schema_version":2,`, invalid, invalid},
	} {
		tc.stored = strings.ReplaceAll(tc.stored, `"dataset":"test-dataset"`, `"dataset":"`+store.dataset+`"`)
		for revision, outcome := range map[string]string{held: tc.heldOutcome, other: tc.otherOutcome} {
			require.NoError(t, client.Set(ctx, store.key, tc.stored, time.Minute).Err())
			renewErr := lease.Renew(ctx, revision)
			if outcome == renewed {
				require.NoError(t, renewErr, name)
				ttl, ttlErr := client.PTTL(ctx, store.key).Result()
				require.NoError(t, ttlErr)
				require.Greater(t, ttl, REDIS_CACHE_TTL-time.Minute, name)
			} else {
				require.ErrorIs(t, renewErr, ErrSnapshotConflict, name)
				require.Equal(t, map[string]string{conflict: "revision_conflict", invalid: "invalid_envelope", dataset: "dataset_rejected", policy: "policy_rejected"}[outcome], SnapshotFailureReason(renewErr), name)
				require.Equal(t, outcome != conflict, SnapshotRenewalFoundNoPublication(renewErr), "only a value that is not a publication may be replaced: %s", name)
			}
			after, getErr := client.Get(ctx, store.key).Result()
			require.NoError(t, getErr)
			require.Equal(t, tc.stored, after, "a renewal never changes the value: %s", name)
		}
	}

	// The revision of a value that begins as a published envelope is read from
	// its beginning alone: what follows the users member is not decoded.
	require.NoError(t, client.Set(ctx, store.key, head+held+usersMember+"[ not decoded", time.Minute).Err())
	require.NoError(t, lease.Renew(ctx, held))
	require.ErrorIs(t, lease.Renew(ctx, other), ErrSnapshotConflict)

	require.NoError(t, client.Del(ctx, store.key).Err())
	require.ErrorIs(t, lease.Renew(ctx, held), ErrSnapshotMissing)
	require.NoError(t, client.Set(ctx, store.key, strings.Repeat("x", define.MAX_ENVELOPE_JSON_SIZE+1), time.Minute).Err())
	require.ErrorIs(t, lease.Renew(ctx, held), ErrSnapshotTooLarge)
	require.NoError(t, client.Set(ctx, store.key, layout, time.Minute).Err())
	require.NoError(t, lease.Release(ctx))
	require.ErrorIs(t, lease.Renew(ctx, held), ErrSnapshotLeaseLost)
}

func TestRenewReadsTheRevisionFromTheEnvelopeHead(t *testing.T) {
	store, _ := snapshotStoreFixture(t)
	renewCases(t, store, store.client)
}

func TestRedisSnapshotIntegrationRenewReadsTheRevisionFromTheEnvelopeHead(t *testing.T) {
	store, client := realSnapshotStore(t)
	renewCases(t, store, client)
}

// A published snapshot whose user records carry text that looks like the
// envelope's own members renews by its real revision only.
func TestRenewIsNotMisledByPublishedUserData(t *testing.T) {
	store, mr := snapshotStoreFixture(t)
	ctx := context.Background()
	lease, err := store.Acquire(ctx, time.Minute)
	require.NoError(t, err)
	fake := strings.Repeat("f", 32)
	users := []define.AllowListUser{
		{Mail: store.head + fake + usersMember, UserID: `"revision":"` + fake + `"`, Status: "active", Scope: []string{`{"revision":"` + fake + `"}`}},
		{Mail: "member@example.com", Name: `","revision":"` + fake + `","users":[`},
	}
	envelope := &SnapshotEnvelope{Users: users, EffectiveVersion: HashUserList(users), InputVersion: HashUserList(users), InputCount: len(users), Origin: "remote", SourceSucceededAt: time.Now()}
	published, err := lease.Publish(ctx, envelope)
	require.NoError(t, err)
	before, err := mr.Get(store.key)
	require.NoError(t, err)
	require.Contains(t, before, fake)

	require.ErrorIs(t, lease.Renew(ctx, fake), ErrSnapshotConflict)
	mr.SetTTL(store.key, time.Second)
	require.NoError(t, lease.Renew(ctx, published.Revision))
	require.Equal(t, REDIS_CACHE_TTL, mr.TTL(store.key))
	after, err := mr.Get(store.key)
	require.NoError(t, err)
	require.Equal(t, before, after)
	read, err := store.Read(ctx)
	require.NoError(t, err)
	require.Equal(t, published, read)
}
