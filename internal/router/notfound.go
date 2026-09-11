// Package router provides HTTP routing functionality.
// notfound.go: catch-all handler for paths Warden does not register.
package router

import (
	"net/http"

	"github.com/soulteary/warden/internal/i18n"
)

// NotFound returns the handler mounted on the catch-all "/" pattern.
//
// Warden registers every endpoint it serves explicitly; the "/" pattern exists only to
// absorb everything else. Historically "/" was bound to the full user-list handler, so
// any unmatched path (/foo, /user/, /v1/, ...) returned the complete allow list. The
// root document is now served by the exact-match "/{$}" pattern and unmatched paths get
// a localized JSON 404 instead.
func NotFound() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		WriteJSONError(w, http.StatusNotFound, i18n.T(r, "error.not_found"))
	}
}
