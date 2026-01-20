package parser

import (
	"testing"

	"github.com/soulteary/warden/internal/define"
	"github.com/stretchr/testify/assert"
)

func TestMergeUsersOrder(t *testing.T) {
	dict := map[string]define.AllowListUser{
		"13800138000": {Phone: "13800138000", Mail: "user1@example.com"},
		"13900139000": {Phone: "13900139000", Mail: "user2@example.com"},
	}
	order := []string{"13900139000", "13800138000", "not-exists"}

	result := mergeUsers(dict, order)
	if assert.Len(t, result, 2) {
		assert.Equal(t, "13900139000", result[0].Phone)
		assert.Equal(t, "13800138000", result[1].Phone)
	}
}

func TestAddRulesToDict(t *testing.T) {
	dict := map[string]define.AllowListUser{
		"13800138000": {Phone: "13800138000", Mail: "old@example.com"},
	}
	order := []string{"13800138000"}

	rules := []define.AllowListUser{
		{Phone: "13800138000", Mail: "new@example.com"},
		{Phone: "13900139000", Mail: "user2@example.com"},
	}

	addRulesToDict(dict, &order, rules, false)

	assert.Len(t, order, 2)
	assert.Equal(t, "13800138000", order[0])
	assert.Equal(t, "13900139000", order[1])
	assert.Equal(t, "new@example.com", dict["13800138000"].Mail)
	assert.Equal(t, "user2@example.com", dict["13900139000"].Mail)
}
