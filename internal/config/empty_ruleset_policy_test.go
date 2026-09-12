package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseEmptyRulesetPolicy(t *testing.T) {
	cases := []struct {
		in    string
		want  EmptyRulesetPolicy
		wantY bool
	}{
		{"", DefaultEmptyRulesetPolicy, true},
		{"   ", DefaultEmptyRulesetPolicy, true},
		{"consistency-first", EmptyRulesetConsistencyFirst, true},
		{"CONSISTENCY-FIRST", EmptyRulesetConsistencyFirst, true},
		{"  Consistency-First  ", EmptyRulesetConsistencyFirst, true},
		{"consistency", EmptyRulesetConsistencyFirst, true},
		{"availability-first", EmptyRulesetAvailabilityFirst, true},
		{"AVAILABILITY-FIRST", EmptyRulesetAvailabilityFirst, true},
		{"availability", EmptyRulesetAvailabilityFirst, true},
		{"last-known-good", DefaultEmptyRulesetPolicy, false},
		{"true", DefaultEmptyRulesetPolicy, false},
	}

	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got, ok := ParseEmptyRulesetPolicy(tc.in)
			assert.Equal(t, tc.wantY, ok, "识别结果不符合预期")
			assert.Equal(t, tc.want, got, "解析结果不符合预期")
		})
	}
}

// TestParseEmptyRulesetPolicy_UnknownFallsBackToDefault pins the contract the startup path
// relies on: an unrecognized value must report false AND still hand back a usable policy,
// so a caller that only logs the warning cannot end up with an undefined value.
func TestParseEmptyRulesetPolicy_UnknownFallsBackToDefault(t *testing.T) {
	got, ok := ParseEmptyRulesetPolicy("no-such-policy")
	assert.False(t, ok)
	assert.Equal(t, EmptyRulesetConsistencyFirst, got)
	assert.True(t, got.Validate(), "回退值本身必须是合法策略")
}

func TestEmptyRulesetPolicy_Validate(t *testing.T) {
	assert.True(t, EmptyRulesetConsistencyFirst.Validate())
	assert.True(t, EmptyRulesetAvailabilityFirst.Validate())
	assert.False(t, EmptyRulesetPolicy("").Validate())
	assert.False(t, EmptyRulesetPolicy("consistency").Validate(), "别名不是规范值")
	assert.False(t, EmptyRulesetPolicy("whatever").Validate())
}

// TestEmptyRulesetPolicy_KeepsLastKnownGood is the branch backgroundTask keys off; getting
// it backwards would invert the whole feature.
func TestEmptyRulesetPolicy_KeepsLastKnownGood(t *testing.T) {
	assert.True(t, EmptyRulesetAvailabilityFirst.KeepsLastKnownGood(),
		"可用性优先必须保留上一次有效的规则集")
	assert.False(t, EmptyRulesetConsistencyFirst.KeepsLastKnownGood(),
		"一致性优先必须应用并发布空集")
	assert.False(t, DefaultEmptyRulesetPolicy.KeepsLastKnownGood(),
		"默认策略必须保持历史行为")
}

func TestEmptyRulesetPolicy_String(t *testing.T) {
	assert.Equal(t, "consistency-first", EmptyRulesetConsistencyFirst.String())
	assert.Equal(t, "availability-first", EmptyRulesetAvailabilityFirst.String())
}

// TestParseEmptyRulesetPolicy_RoundTrip guards against a canonical value that Parse cannot
// read back — the shape of bug that makes a documented setting silently inert.
func TestParseEmptyRulesetPolicy_RoundTrip(t *testing.T) {
	for _, p := range []EmptyRulesetPolicy{EmptyRulesetConsistencyFirst, EmptyRulesetAvailabilityFirst} {
		got, ok := ParseEmptyRulesetPolicy(p.String())
		assert.True(t, ok, "%s 必须能被解析", p)
		assert.Equal(t, p, got)
	}
}
