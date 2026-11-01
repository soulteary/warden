package identity

import (
	"encoding/json"
	"fmt"
	"github.com/soulteary/warden/internal/define"
	"github.com/stretchr/testify/require"
	"math/rand/v2"
	"sort"
	"strings"
	"testing"
)

// Frozen from main bf8863e. Keep ordering, validation order and partial report
// indexes independent of the production sorting and indexing helpers.
func legacyStableSort(users []define.AllowListUser) []define.AllowListUser {
	out := make([]define.AllowListUser, len(users))
	copy(out, users)
	sort.SliceStable(out, func(i, j int) bool {
		ki, kj := out[i].UserID, out[j].UserID
		if ki != kj {
			return ki < kj
		}
		ci, cj := legacyCanonicalKey(&out[i]), legacyCanonicalKey(&out[j])
		if ci != cj {
			return ci < cj
		}
		return strings.ToLower(out[i].Mail) < strings.ToLower(out[j].Mail)
	})
	return out
}

func legacyCanonicalKey(u *define.AllowListUser) string {
	k := strings.TrimSpace(u.Phone)
	if k == "" {
		k = strings.ToLower(strings.TrimSpace(u.Mail))
	}
	return k
}

func legacyValidateAndIndexUsers(users []define.AllowListUser, opts Options) (Result, error) {
	normalized := make([]define.AllowListUser, len(users))
	copy(normalized, users)
	for i := range normalized {
		normalized[i].Normalize()
	}

	phoneIdx := make(map[string]int, len(normalized))
	mailIdx := make(map[string]int, len(normalized))
	userIDIdx := make(map[string]int, len(normalized))

	var res Result
	for i := range normalized {
		u := &normalized[i]
		phone := strings.TrimSpace(u.Phone)
		mail := strings.ToLower(strings.TrimSpace(u.Mail))
		uid := strings.TrimSpace(u.UserID)

		if opts.RequireExplicitUserID && uid == "" {
			mErr := &MissingUserIDError{Index: i, MaskedIdentifier: maskIdentifier(phone, mail)}
			if opts.ReportOnly {
				res.MissingIDCount++
				continue
			}
			return Result{}, mErr
		}

		err := legacyIndexIdentifiers(&res, opts, i, phone, mail, uid, phoneIdx, mailIdx, userIDIdx)
		if err != nil {
			return Result{}, err
		}
	}

	res.Users = legacyStableSort(normalized)
	return res, nil
}

func legacyHandleConflict(res *Result, opts Options, field FieldType, masked string, first, second int) error {
	if opts.ReportOnly {
		res.ConflictCount++
		return nil
	}
	return &ConflictError{Field: field, MaskedValue: masked, FirstIndex: first, SecondIndex: second}
}

func legacyIndexIdentifiers(res *Result, opts Options, index int, phone, mail, uid string, phoneIdx, mailIdx, userIDIdx map[string]int) error {
	fields := []struct {
		indexes map[string]int
		mask    func(string) string
		value   string
		field   FieldType
	}{
		{phoneIdx, MaskPhone, phone, FieldPhone},
		{mailIdx, MaskMail, mail, FieldMail},
		{userIDIdx, MaskUserID, uid, FieldUserID},
	}
	for _, field := range fields {
		if field.value == "" {
			continue
		}
		if previous, ok := field.indexes[field.value]; ok {
			return legacyHandleConflict(res, opts, field.field, field.mask(field.value), previous, index)
		}
		field.indexes[field.value] = index
	}
	return nil
}

func sortContractUsers() [][]define.AllowListUser {
	tied := make([]define.AllowListUser, 40)
	for i := range tied {
		tied[i] = define.AllowListUser{UserID: "same", Phone: "same", Mail: " SAME@Example.com ", Name: fmt.Sprint(i), Scope: []string{"read", fmt.Sprint(i)}}
	}
	return [][]define.AllowListUser{nil, {}, {{UserID: "one"}}, tied,
		{{UserID: "same", Phone: " 2 ", Mail: "Z@example.com"}, {UserID: "same", Phone: "1", Mail: "b@example.com"}, {UserID: "same", Phone: "1", Mail: "A@example.com"}},
		{{Mail: "É@Example.com", Scope: []string{}}, {Mail: "é@example.com", Scope: nil}, {Mail: "a\x00@example.com", UserID: "\xff"}},
		{{UserID: " x ", Phone: " 13800000001 "}, {UserID: "x", Phone: "13800000002"}},
		{{Phone: "13800000001", Mail: "same@example.com", UserID: "a"}, {Phone: "13800000001", Mail: "SAME@example.com", UserID: "a"}},
	}
}

func TestStableSortLegacyDifferential(t *testing.T) {
	cases := sortContractUsers()
	// #nosec G404 -- deterministic contract fixtures, not security-sensitive randomness.
	rng := rand.New(rand.NewPCG(101, 303))
	for sample := 0; sample < 400; sample++ {
		users := make([]define.AllowListUser, rng.IntN(150))
		for i := range users {
			users[i] = define.AllowListUser{UserID: fmt.Sprint(rng.IntN(5)), Phone: fmt.Sprint(rng.IntN(12)), Mail: fmt.Sprintf(" Member%d@Example.com ", rng.IntN(8)), Name: fmt.Sprint(i), Scope: []string{"write", "read", fmt.Sprint(i)}}
		}
		cases = append(cases, users)
	}
	for i, users := range cases {
		before, err := json.Marshal(users)
		require.NoError(t, err)
		require.Equal(t, legacyStableSort(users), stableSort(users), "case %d", i)
		after, err := json.Marshal(users)
		require.NoError(t, err)
		require.Equal(t, before, after, "caller input changed in case %d", i)
	}
}

func TestIdentityValidationLegacyDifferential(t *testing.T) {
	oldStrategy, oldExplicit := define.GetUserIDStrategy(), define.RequireExplicitUserID()
	t.Cleanup(func() { define.SetUserIDStrategy(oldStrategy); define.SetRequireExplicitUserID(oldExplicit) })
	for _, strategy := range []define.UserIDStrategy{define.UserIDStrategyLegacy, define.UserIDStrategySHA256_128} {
		define.SetUserIDStrategy(strategy)
		for _, explicit := range []bool{false, true} {
			define.SetRequireExplicitUserID(explicit)
			for _, report := range []bool{false, true} {
				for _, users := range sortContractUsers() {
					opts := Options{RequireExplicitUserID: explicit, ReportOnly: report}
					before, err := json.Marshal(users)
					require.NoError(t, err)
					want, wantErr := legacyValidateAndIndexUsers(users, opts)
					got, gotErr := ValidateAndIndexUsers(users, opts)
					require.Equal(t, wantErr, gotErr)
					require.Equal(t, want, got)
					after, err := json.Marshal(users)
					require.NoError(t, err)
					require.Equal(t, before, after)
				}
			}
		}
	}
}

func BenchmarkIdentityStableSort(b *testing.B) {
	for _, count := range []int{1, 1000, 10000} {
		for _, tied := range []bool{false, true} {
			users := make([]define.AllowListUser, count)
			for i := range users {
				users[i] = define.AllowListUser{UserID: fmt.Sprintf("id-%05d", i), Mail: fmt.Sprintf("Member%05d@Example.com", i), Scope: []string{"read", "write"}}
				if tied {
					users[i].UserID = "same"
				}
			}
			// #nosec G404 -- fixed-seed benchmark fixture.
			rng := rand.New(rand.NewPCG(11, 33))
			rng.Shuffle(len(users), func(i, j int) { users[i], users[j] = users[j], users[i] })
			if len(stableSort(users)) != count {
				b.Fatal("fixture lost rows")
			}
			b.Run(fmt.Sprintf("rows=%d/tied=%t", count, tied), func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					stableSort(users)
				}
			})
		}
	}
}
