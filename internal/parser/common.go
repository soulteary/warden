// Package parser provides data parsing functionality.
// Supports parsing user data from local files and remote APIs, and provides multiple data merging strategies.
package parser

import (
	// Standard library
	"context"

	// Internal packages
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/logger"
)

var log = logger.GetLogger()

// mergeUsers converts map to slice, according to specified order
// dict: user data map, key is phone
// order: order list, stores phone
func mergeUsers(dict map[string]define.AllowListUser, order []string) []define.AllowListUser {
	result := make([]define.AllowListUser, 0, len(order))
	for _, phone := range order {
		if user, exists := dict[phone]; exists {
			result = append(result, user)
		}
	}
	return result
}

// addRulesToDict adds rules to dictionary, maintains order list
// dict: user data map, key is phone
// order: order list, stores phone
// rules: rules list to add
// logMessage: whether to log message (used to distinguish remote and local rules)
// Returns number of new rules added
func addRulesToDict(dict map[string]define.AllowListUser, order *[]string, rules []define.AllowListUser, logMessage bool) {
	for _, rule := range rules {
		if _, exists := dict[rule.Phone]; !exists {
			*order = append(*order, rule.Phone)
			if logMessage {
				log.Debug().Msgf("Loading remote rule %s => %s", rule.Mail, rule.Phone)
			}
		}
		dict[rule.Phone] = rule
	}
}

// GetRules gets rules according to mode (supports context)
//
// This function is the unified entry point for rule retrieval, selects different data retrieval and merging strategies based on different modes (appMode).
// Supported modes include:
//   - DEFAULT/REMOTE_FIRST: remote first, local supplement
//   - ONLY_REMOTE: only use remote rules
//   - ONLY_LOCAL: only use local rules
//   - LOCAL_FIRST: local first, remote supplement
//   - REMOTE_FIRST_ALLOW_REMOTE_FAILED: remote first, allow continuation when remote fails
//   - LOCAL_FIRST_ALLOW_REMOTE_FAILED: local first, allow continuation when remote fails
//
// Parameters:
//   - ctx: context for request cancellation and timeout control
//   - rulesFile: local rules file path
//   - configURL: remote configuration URL
//   - authorizationHeader: Authorization header for remote request
//   - appMode: application mode, determines data retrieval strategy
//
// Returns:
//   - []define.AllowListUser: merged user list, arranged in addition order
//
// Side effects:
//   - Records debug and warning logs
//   - May perform network requests (depending on mode)
//   - May read local files (depending on mode)
func GetRules(ctx context.Context, rulesFile, configURL, authorizationHeader, appMode string) (result []define.AllowListUser) {
	switch appMode {
	case "DEFAULT", "REMOTE_FIRST":
		return remoteRulesFirstAppendNotExistsFromLocalRules(ctx, rulesFile, configURL, authorizationHeader, false)
	case "ONLY_REMOTE":
		return onlyRemoteRules(ctx, configURL, authorizationHeader)
	case "ONLY_LOCAL":
		return onlyLocalRules(rulesFile)
	case "LOCAL_FIRST":
		return localRulesFirstAppendNotExistsFromRemoteRules(ctx, rulesFile, configURL, authorizationHeader, false)
	case "REMOTE_FIRST_ALLOW_REMOTE_FAILED":
		return remoteRulesFirstAppendNotExistsFromLocalRules(ctx, rulesFile, configURL, authorizationHeader, true)
	case "LOCAL_FIRST_ALLOW_REMOTE_FAILED":
		return localRulesFirstAppendNotExistsFromRemoteRules(ctx, rulesFile, configURL, authorizationHeader, true)
	default:
		return remoteRulesFirstAppendNotExistsFromLocalRules(ctx, rulesFile, configURL, authorizationHeader, false)
	}
}

// remoteRulesFirstAppendNotExistsFromLocalRules remote rules first, supplement items not in remote rules from local rules
//
// This function implements remote-first data merging strategy:
// - First tries to get rules from remote API
// - If remote retrieval fails and allowSkipRemoteFailed is false, returns empty result
// - If remote retrieval fails and allowSkipRemoteFailed is true, continues using local rules
// - Supplements items from local rules that don't exist in remote rules to result
//
// Parameters:
//   - ctx: context for request cancellation and timeout control
//   - rulesFile: local rules file path
//   - configURL: remote configuration URL
//   - authorizationHeader: Authorization header for remote request
//   - allowSkipRemoteFailed: whether to allow continuation when remote fails
//
// Returns:
//   - []define.AllowListUser: merged user list, arranged in addition order
func remoteRulesFirstAppendNotExistsFromLocalRules(ctx context.Context, rulesFile, configURL, authorizationHeader string, allowSkipRemoteFailed bool) (result []define.AllowListUser) {
	var dict = make(map[string]define.AllowListUser)
	var order = make([]string, 0) // Maintain order list

	// Prefer remote rules for initialization
	// If configURL is empty, skip remote request, directly use local rules
	if configURL != "" {
		remoteRules, err := FromRemoteConfig(ctx, configURL, authorizationHeader)
		if err != nil {
			log.Warn().
				Err(err).
				Msg(define.WARN_GET_REMOTE_FAILED_FALLBACK_LOCAL)
			if !allowSkipRemoteFailed {
				return result
			}
		} else if len(remoteRules) > 0 {
			addRulesToDict(dict, &order, remoteRules, true)
		}
	}

	// Supplement local rules that don't exist in remote rules
	localRules := FromFile(rulesFile)
	addRulesToDict(dict, &order, localRules, false)

	result = mergeUsers(dict, order)
	log.Debug().Msgf("Rules update completed 📦")
	return result
}

// localRulesFirstAppendNotExistsFromRemoteRules local rules first, supplement items not in local rules from remote rules
//
// This function implements local-first data merging strategy:
// - First loads rules from local file
// - Then tries to get rules from remote API
// - If remote retrieval fails and allowSkipRemoteFailed is false, returns result containing only local rules
// - If remote retrieval fails and allowSkipRemoteFailed is true, continues using local rules
// - Supplements items from remote rules that don't exist in local rules to result
//
// Parameters:
//   - ctx: context for request cancellation and timeout control
//   - rulesFile: local rules file path
//   - configURL: remote configuration URL
//   - authorizationHeader: Authorization header for remote request
//   - allowSkipRemoteFailed: whether to allow continuation when remote fails
//
// Returns:
//   - []define.AllowListUser: merged user list, arranged in addition order
func localRulesFirstAppendNotExistsFromRemoteRules(ctx context.Context, rulesFile, configURL, authorizationHeader string, allowSkipRemoteFailed bool) (result []define.AllowListUser) {
	var dict = make(map[string]define.AllowListUser)
	var order = make([]string, 0) // Maintain order list

	// Prefer loading local data
	localRules := FromFile(rulesFile)
	addRulesToDict(dict, &order, localRules, false)

	// Supplement remote rules that don't exist in local rules
	// If configURL is empty, skip remote request
	if configURL != "" {
		remoteRules, err := FromRemoteConfig(ctx, configURL, authorizationHeader)
		if err != nil {
			log.Warn().
				Err(err).
				Msg(define.WARN_GET_REMOTE_FAILED_FALLBACK_LOCAL)
			if !allowSkipRemoteFailed {
				return result
			}
		} else if len(remoteRules) > 0 {
			addRulesToDict(dict, &order, remoteRules, true)
		}
	}

	result = mergeUsers(dict, order)
	log.Debug().Msgf("Rules update completed 📦")
	return result
}

// onlyRemoteRules only uses remote rules
//
// This function only retrieves rules from remote API, does not use local files.
// If remote retrieval fails, returns empty result.
//
// Parameters:
//   - ctx: context for request cancellation and timeout control
//   - configURL: remote configuration URL
//   - authorizationHeader: Authorization header for remote request
//
// Returns:
//   - []define.AllowListUser: user list retrieved from remote, returns empty list if retrieval fails
func onlyRemoteRules(ctx context.Context, configURL, authorizationHeader string) (result []define.AllowListUser) {
	var dict = make(map[string]define.AllowListUser)
	var order = make([]string, 0) // Maintain order list

	// Use remote rules for initialization
	// If configURL is empty, directly return empty result
	if configURL != "" {
		remoteRules, err := FromRemoteConfig(ctx, configURL, authorizationHeader)
		if err != nil {
			log.Warn().
				Err(err).
				Msg(define.WARN_GET_REMOTE_FAILED_FALLBACK_LOCAL)
		} else if len(remoteRules) > 0 {
			addRulesToDict(dict, &order, remoteRules, true)
		}
	}

	result = mergeUsers(dict, order)
	log.Debug().Msgf("Rules update completed 📦")
	return result
}

// onlyLocalRules only uses local rules
//
// This function only loads rules from local file, does not access remote API.
//
// Parameters:
//   - rulesFile: local rules file path
//
// Returns:
//   - []define.AllowListUser: user list loaded from local file
func onlyLocalRules(rulesFile string) (result []define.AllowListUser) {
	var dict = make(map[string]define.AllowListUser)
	var order = make([]string, 0) // Maintain order list

	localRules := FromFile(rulesFile)
	addRulesToDict(dict, &order, localRules, false)

	result = mergeUsers(dict, order)
	log.Debug().Msgf("Rules update completed 📦")
	return result
}
