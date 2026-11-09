package main

import (
	"encoding/json"
	"errors"
	"io"
	"strings"

	version "github.com/soulteary/version-kit/v4"
)

// handleVersionCommand runs before the application startup path. Package init
// functions have already run; this does not claim to bypass Go initialization.
func handleVersionCommand(args []string, output io.Writer) (bool, error) {
	for _, arg := range args {
		if arg == "--version" || strings.HasPrefix(arg, "--version=") {
			if len(args) != 1 || arg != "--version" {
				return true, errors.New("--version must be used alone")
			}
			return true, json.NewEncoder(output).Encode(version.Default())
		}
	}
	return false, nil
}
