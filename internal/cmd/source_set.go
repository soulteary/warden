package cmd

import (
	"fmt"
	"regexp"
)

var sourceSetPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9:_-]{0,63}$`)

func validateSourceSetID(value string) error {
	if value != "" && !sourceSetPattern.MatchString(value) {
		return fmt.Errorf("SOURCE_SET_ID must be empty or 1-64 letters, digits, colons, underscores or hyphens, beginning with a letter or digit")
	}
	return nil
}
