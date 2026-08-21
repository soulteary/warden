// Package config: environment.go defines the deployment Environment type.
//
// Historically a single overloaded "MODE" string conflated two orthogonal concerns:
// the data merge strategy (ONLY_LOCAL / LOCAL_FIRST / REMOTE_FIRST / ...) and the
// deployment environment used for security policy (production hardening). This file
// introduces a dedicated Environment type so security decisions key off ENVIRONMENT
// only, never off the merge mode.
package config

import "strings"

// Environment is the deployment environment used exclusively for security policy
// (e.g. forbidding InsecureSkipVerify, requiring auth). It is NOT a data merge mode.
type Environment string

const (
	// EnvDevelopment is the default, permissive environment.
	EnvDevelopment Environment = "development"
	// EnvTest is for automated/integration testing.
	EnvTest Environment = "test"
	// EnvProduction enables strict security validation.
	EnvProduction Environment = "production"
)

// DefaultEnvironment is used when nothing is configured. Development is the safe
// default because it does not silently relax production checks.
const DefaultEnvironment = EnvDevelopment

// ParseEnvironment normalizes an environment string. It accepts the historical
// "prod" alias and normalizes it to production. The returned bool is false when the
// input is non-empty but unrecognized (callers should warn and fall back to default).
func ParseEnvironment(s string) (Environment, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "":
		return DefaultEnvironment, true
	case "dev", "develop", "development":
		return EnvDevelopment, true
	case "test", "testing":
		return EnvTest, true
	case "prod", "production":
		return EnvProduction, true
	default:
		return DefaultEnvironment, false
	}
}

// Validate reports whether the Environment is a recognized value.
func (e Environment) Validate() bool {
	switch e {
	case EnvDevelopment, EnvTest, EnvProduction:
		return true
	default:
		return false
	}
}

// IsProduction reports whether strict production security policy applies.
func (e Environment) IsProduction() bool { return e == EnvProduction }

// String returns the canonical string form.
func (e Environment) String() string { return string(e) }
