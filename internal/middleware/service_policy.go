package middleware

import (
	"net/http"

	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/serviceauth"
)

func serviceAction(r *http.Request) string {
	switch r.URL.Path {
	case define.PATH_ROOT, define.PATH_DATA_JSON, define.PATH_V1_USERS:
		if r.Method == http.MethodGet {
			return "directory:list"
		}
	case define.PATH_USER, define.PATH_V1_USER, define.PATH_V1_LOOKUP:
		if r.Method == http.MethodGet {
			return "directory:lookup"
		}
	case define.PATH_LOG_LEVEL:
		if r.Method == http.MethodGet || r.Method == http.MethodPost || r.Method == http.MethodPut {
			return "admin:loglevel"
		}
	case define.PATH_METRICS:
		if r.Method == http.MethodGet {
			return "metrics:read"
		}
	}
	return ""
}

func servePolicyAuthenticated(cfg *HMACConfig, next http.Handler, w http.ResponseWriter, r *http.Request) {
	sig, ts, key := r.Header.Get(headerSignature), r.Header.Get(headerTimestamp), r.Header.Get(headerKeyID)
	version := r.Header.Get(headerVersion)
	// Policies need a nonce-bound signature: v2, or v3 which adds the audience.
	if sig == "" || ts == "" || key == "" || (version != signatureV2 && version != signatureV3) || !verifyHMAC(*cfg, r, sig, ts, key) {
		writeUnauthorized(w, r)
		return
	}
	if !serviceauth.Allows(cfg.ServicePolicies, key, serviceAction(r)) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	next.ServeHTTP(w, WithPrincipal(r, principalHMACPrefix+key))
}
