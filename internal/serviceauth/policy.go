// Package serviceauth defines optional service credential permissions.
package serviceauth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

// Policy restricts an authenticated HMAC key to named operations.
type Policy struct {
	Actions []string `json:"actions"`
}

var allowedActions = map[string]bool{
	"directory:lookup": true,
	"directory:list":   true,
	"admin:loglevel":   true,
	"metrics:read":     true,
}

// Parse accepts a complete policy map. An empty setting retains legacy access.
func Parse(raw string, keys map[string]string) (map[string]Policy, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	var policies map[string]Policy
	if err := decoder.Decode(&policies); err != nil {
		return nil, errors.New("WARDEN_SERVICE_POLICIES must be a JSON object of key IDs and actions")
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return nil, errors.New("WARDEN_SERVICE_POLICIES must contain one JSON value")
	}
	if len(policies) == 0 {
		return nil, errors.New("WARDEN_SERVICE_POLICIES must contain at least one policy")
	}
	seenSecrets := make(map[string]bool, len(keys))
	for _, secret := range keys {
		if seenSecrets[secret] {
			return nil, errors.New("WARDEN_SERVICE_POLICIES requires distinct HMAC secrets for every key ID")
		}
		seenSecrets[secret] = true
	}
	for key, policy := range policies {
		if key == "" || key != strings.TrimSpace(key) || strings.TrimSpace(keys[key]) == "" {
			return nil, errors.New("WARDEN_SERVICE_POLICIES must reference configured HMAC key IDs")
		}
		if len(policy.Actions) == 0 {
			return nil, fmt.Errorf("WARDEN_SERVICE_POLICIES[%q] requires actions", key)
		}
		seen := make(map[string]bool, len(policy.Actions))
		for _, action := range policy.Actions {
			if !allowedActions[action] || seen[action] {
				return nil, fmt.Errorf("WARDEN_SERVICE_POLICIES[%q] has an unknown or duplicate action", key)
			}
			seen[action] = true
		}
	}
	return policies, nil
}

// Allows reports whether a key has a specific operation. Unknown keys deny.
func Allows(policies map[string]Policy, key, action string) bool {
	if action == "" {
		return false
	}
	for _, allowed := range policies[key].Actions {
		if allowed == action {
			return true
		}
	}
	return false
}
