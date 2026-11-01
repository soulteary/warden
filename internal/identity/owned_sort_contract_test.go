package identity

import (
	"encoding/json"
	"fmt"
	"github.com/soulteary/warden/internal/define"
	"github.com/stretchr/testify/require"
	"math/rand/v2"
	"runtime"
	"testing"
)

func checkOwnedPermutation(t *testing.T, order []int) {
	t.Helper()
	rows := make([]define.AllowListUser, len(order))
	for rank, source := range order {
		rows[source] = define.AllowListUser{UserID: fmt.Sprintf("rank-%03d", rank), Mail: fmt.Sprintf("member%d@example.com", source), Status: "active", Name: fmt.Sprint(source), Scope: []string{"read", fmt.Sprint(source)}}
	}
	before, err := json.Marshal(rows)
	require.NoError(t, err)
	want := make([]define.AllowListUser, len(order))
	for i, source := range order {
		want[i] = rows[source]
	}
	public, err := ValidateAndIndexUsers(rows, Options{})
	require.NoError(t, err)
	require.Equal(t, want, public.Users, "order %v", order)
	require.NotNil(t, public.Users)
	require.Equal(t, len(rows), cap(public.Users))
	if len(public.Users) > 0 {
		public.Users[0].Name = "changed result"
	}
	after, err := json.Marshal(rows)
	require.NoError(t, err)
	require.Equal(t, before, after, "public result must not alias the caller outer slice")
	storage := make([]define.AllowListUser, len(rows)+3)
	for i := range storage {
		storage[i].Name = "spare"
	}
	copy(storage, rows)
	owned, err := ValidateAndIndexOwnedUsers(storage[:len(rows)], Options{})
	require.NoError(t, err)
	require.Equal(t, want, owned.Users, "order %v", order)
	require.NotNil(t, owned.Users)
	require.Equal(t, len(rows), cap(owned.Users))
	for i := len(rows); i < len(storage); i++ {
		require.Equal(t, "spare", storage[i].Name, "do not write beyond input length")
	}
	after, err = json.Marshal(rows)
	require.NoError(t, err)
	require.Equal(t, before, after, "existing Scope contents must not be modified")
}

func TestOwnedSortPermutationContract(t *testing.T) {
	for size := 0; size <= 7; size++ {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			order := make([]int, size)
			for i := range order {
				order[i] = i
			}
			var visit func(int)
			visit = func(position int) {
				if position == len(order) {
					checkOwnedPermutation(t, order)
					return
				}
				for i := position; i < len(order); i++ {
					order[position], order[i] = order[i], order[position]
					visit(position + 1)
					order[position], order[i] = order[i], order[position]
				}
			}
			visit(0)
		})
	}
}

func TestOwnedSortNilEmptyAndCapacity(t *testing.T) {
	for _, rows := range [][]define.AllowListUser{nil, {}, make([]define.AllowListUser, 0, 4), {{UserID: "one", Mail: "one@example.com", Scope: []string{}}}} {
		for _, validate := range []func([]define.AllowListUser, Options) (Result, error){ValidateAndIndexUsers, ValidateAndIndexOwnedUsers} {
			result, err := validate(rows, Options{})
			require.NoError(t, err)
			require.NotNil(t, result.Users)
			require.Equal(t, len(rows), len(result.Users))
			require.Equal(t, len(rows), cap(result.Users))
		}
	}
}

func TestOwnedValidationErrorIndexContract(t *testing.T) {
	tests := []struct {
		name          string
		field         FieldType
		rows          []define.AllowListUser
		first, second int
		explicit      bool
	}{
		{name: "phone before mail and id", field: FieldPhone, first: 1, second: 2, rows: []define.AllowListUser{{UserID: "z", Phone: "13800000001", Mail: "one@example.com"}, {UserID: "a", Phone: "13800000002", Mail: "two@example.com"}, {UserID: "a", Phone: "13800000002", Mail: "two@example.com"}}},
		{name: "mail", field: FieldMail, first: 0, second: 2, rows: []define.AllowListUser{{UserID: "z", Mail: " DUP@example.com "}, {UserID: "a", Mail: "other@example.com"}, {UserID: "b", Mail: "dup@example.com"}}},
		{name: "id", field: FieldUserID, first: 0, second: 2, rows: []define.AllowListUser{{UserID: "z", Mail: "one@example.com"}, {UserID: "a", Mail: "two@example.com"}, {UserID: "z", Mail: "three@example.com"}}},
		{name: "missing", second: 1, explicit: true, rows: []define.AllowListUser{{UserID: "z", Mail: "one@example.com"}, {UserID: " ", Mail: "two@example.com"}, {UserID: "a", Mail: "three@example.com"}}},
	}
	for _, tc := range tests {
		for _, report := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/report=%t", tc.name, report), func(t *testing.T) {
				for i := range tc.rows {
					tc.rows[i].Name = fmt.Sprint(i)
				}
				before, err := json.Marshal(tc.rows)
				require.NoError(t, err)
				opts := Options{RequireExplicitUserID: tc.explicit, ReportOnly: report}
				want, wantErr := legacyValidateAndIndexUsers(tc.rows, opts)
				public, publicErr := ValidateAndIndexUsers(tc.rows, opts)
				require.Equal(t, wantErr, publicErr)
				require.Equal(t, want, public)
				private := append([]define.AllowListUser{}, tc.rows...)
				got, gotErr := ValidateAndIndexOwnedUsers(private, opts)
				require.Equal(t, wantErr, gotErr)
				require.Equal(t, want, got)
				after, err := json.Marshal(tc.rows)
				require.NoError(t, err)
				require.Equal(t, before, after)
				if report {
					return
				}
				checkOwnedErrorPosition(t, gotErr, tc.field, tc.first, tc.second)
				for i := range private {
					require.Equal(t, fmt.Sprint(i), private[i].Name, "errors retain original positions")
					require.Equal(t, "inactive", private[i].Status, "normalization precedes validation")
				}
			})
		}
	}
}
func checkOwnedErrorPosition(t *testing.T, err error, field FieldType, first, second int) {
	t.Helper()
	if field == "" {
		var missing *MissingUserIDError
		require.ErrorAs(t, err, &missing)
		require.Equal(t, second, missing.Index)
		return
	}
	var conflict *ConflictError
	require.ErrorAs(t, err, &conflict)
	require.Equal(t, field, conflict.Field)
	require.Equal(t, first, conflict.FirstIndex)
	require.Equal(t, second, conflict.SecondIndex)
}

func FuzzOwnedValidationContract(f *testing.F) {
	for _, raw := range [][]byte{nil, {}, {1, 2, 3, 4}, {255, 0, 255, 0}} {
		f.Add(raw)
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		if len(raw) > 128 {
			return
		}
		rows := make([]define.AllowListUser, len(raw))
		for i, value := range raw {
			rows[i] = define.AllowListUser{UserID: fmt.Sprint(value % 7), Mail: fmt.Sprintf("Member%d@Example.com", value%11), Name: fmt.Sprint(i), Scope: []string{fmt.Sprint(i)}}
			if value%5 == 0 {
				rows[i].UserID = " "
			}
		}
		before, err := json.Marshal(rows)
		require.NoError(t, err)
		for _, report := range []bool{false, true} {
			opts := Options{RequireExplicitUserID: true, ReportOnly: report}
			want, wantErr := legacyValidateAndIndexUsers(rows, opts)
			owned := append([]define.AllowListUser{}, rows...)
			got, gotErr := ValidateAndIndexOwnedUsers(owned, opts)
			require.Equal(t, wantErr, gotErr)
			require.Equal(t, want, got)
			after, e := json.Marshal(rows)
			require.NoError(t, e)
			require.Equal(t, before, after)
		}
	})
}

func BenchmarkOwnedValidationOrder(b *testing.B) {
	for _, pattern := range []string{"sorted", "reverse", "shuffled"} {
		rows := make([]define.AllowListUser, 10000)
		for i := range rows {
			rank := i
			if pattern == "reverse" {
				rank = 10000 - i
			}
			rows[i] = define.AllowListUser{UserID: fmt.Sprintf("member-%05d", rank), Mail: fmt.Sprintf("Member%05d@Example.com", i), Status: "active", Scope: []string{"read", "write"}}
		}
		// #nosec G404 -- deterministic benchmark fixture, not security-sensitive.
		if pattern == "shuffled" {
			rng := rand.New(rand.NewPCG(31, 53))
			rng.Shuffle(len(rows), func(i, j int) { rows[i], rows[j] = rows[j], rows[i] })
		}
		for _, api := range []struct {
			validate func([]define.AllowListUser, Options) (Result, error)
			name     string
		}{{ValidateAndIndexUsers, "Public"}, {ValidateAndIndexOwnedUsers, "Owned"}} {
			b.Run(pattern+"/"+api.name, func(b *testing.B) {
				scratch := make([]define.AllowListUser, len(rows))
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					copy(scratch, rows)
					result, err := api.validate(scratch, Options{})
					if err != nil {
						b.Fatal(err)
					}
					runtime.KeepAlive(result)
				}
			})
		}
	}
}
