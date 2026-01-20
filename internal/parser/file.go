// Package parser provides data parsing functionality.
// Supports parsing user data from local files and remote APIs, and provides multiple data merging strategies.
package parser

import (
	// Standard library
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	// Internal packages
	"github.com/soulteary/warden/internal/define"
)

// FromFile reads user rules list from local file.
//
// This function reads user data from specified JSON file with the following features:
// - Automatically handles file not found (returns empty list)
// - Automatically handles file read errors (logs warning)
// - Automatically handles JSON parsing errors (logs warning)
//
// Parameters:
//   - rulesFile: path to rules file, should be a valid JSON file path
//
// Returns:
//   - []define.AllowListUser: parsed user list, returns empty list if file does not exist or parsing fails
//
// Side effects:
//   - Records warning and error logs
//   - If file exists but read fails, records error log
//   - If JSON parsing fails, records warning log
func FromFile(rulesFile string) (rules []define.AllowListUser) {
	if _, err := os.Stat(rulesFile); errors.Is(err, os.ErrNotExist) {
		log.Warn().
			Str("err", define.WARN_RULE_NOT_FOUND).
			Msgf(define.WARN_RULE_NOT_FOUND)
		return rules
	}

	// #nosec G304 -- rulesFile comes from configuration file, already validated
	file, err := os.Open(rulesFile)
	if err != nil {
		log.Error().
			Err(fmt.Errorf("%s: %w", define.ERROR_CAN_NOT_OPEN_RULE, err)).
			Str("file", rulesFile).
			Msg(define.ERROR_CAN_NOT_OPEN_RULE)
		return rules
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			log.Error().
				Err(closeErr).
				Str("file", rulesFile).
				Msg("Failed to close file")
		}
	}()

	// Limit file read size to prevent memory exhaustion attacks
	raw, err := io.ReadAll(io.LimitReader(file, define.MAX_JSON_SIZE))
	if err != nil {
		log.Warn().
			Err(fmt.Errorf("%s: %w", define.WARN_READ_RULE_ERR, err)).
			Str("file", rulesFile).
			Msg(define.WARN_READ_RULE_ERR)
		return rules
	}

	err = json.Unmarshal(raw, &rules)
	if err != nil {
		log.Warn().
			Err(fmt.Errorf("%s: %w", define.WARN_PARSE_RULE_ERR, err)).
			Str("file", rulesFile).
			Msg(define.WARN_PARSE_RULE_ERR)
		return rules
	}

	// Normalize all user data (set default values, generate user_id)
	for i := range rules {
		rules[i].Normalize()
	}

	return rules
}
