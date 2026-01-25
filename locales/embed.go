// Package locales provides embedded translation files.
package locales

import "embed"

// FS contains embedded translation files.
//
//go:embed *.json
var FS embed.FS
