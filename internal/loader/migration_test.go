package loader

import (
	"context"
	"crypto/x509"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/soulteary/warden/internal/cmd"
)

// The tests in this file pin the parser-kit v1 behaviours that the v3 migration
// could have dropped silently: v3 has no equivalent of AllowEmptyFile, decodes
// zero bytes as an empty list instead of failing, validates a remote URL when
// the source is built rather than when it is read, and no longer injects trace
// headers on its own.

const oneUserJSON = `[{"phone":"13800138000","mail":"a@example.com"}]`

// TestOnlyLocalMissingFileIsEmpty covers the AllowEmptyFile replacement: in
// ONLY_LOCAL mode a rules file that does not exist loads as an empty list
// rather than an error, which is what File().AllowMissing() now provides.
func TestOnlyLocalMissingFileIsEmpty(t *testing.T) {
	r, err := NewRulesLoader(nil, ModeOnlyLocal)
	require.NoError(t, err)

	missing := filepath.Join(t.TempDir(), "absent.json")

	t.Run("FromFile", func(t *testing.T) {
		users, err := r.FromFile(context.Background(), missing)
		require.NoError(t, err)
		assert.Empty(t, users)
	})

	t.Run("Load", func(t *testing.T) {
		res := r.LoadWithResult(context.Background(), missing, "", "", "")
		require.NoError(t, res.Err)
		assert.Empty(t, res.Users)
		assert.Equal(t, SourceLocal, res.Source)
	})

	t.Run("other_modes_still_fail", func(t *testing.T) {
		strict, err := NewRulesLoader(nil, ModeLocalFirst)
		require.NoError(t, err)
		_, err = strict.FromFile(context.Background(), missing)
		require.Error(t, err)
	})
}

// TestZeroByteSourceFails covers the wrapper that restores the v1 reading of an
// empty payload. v1 decoded every source with json.Unmarshal, so zero bytes
// failed that source; v3 decodes them as an empty list, which AllowEmptyData
// would accept and hand to EMPTY_RULESET_POLICY as a legitimate empty rule set.
func TestZeroByteSourceFails(t *testing.T) {
	t.Run("zero_byte_file", func(t *testing.T) {
		empty := filepath.Join(t.TempDir(), "empty.json")
		require.NoError(t, os.WriteFile(empty, []byte{}, 0o600))

		r, err := NewRulesLoader(nil, ModeOnlyLocal)
		require.NoError(t, err)

		_, err = r.FromFile(context.Background(), empty)
		require.Error(t, err, "an empty file must fail the source, not decode as []")

		res := r.LoadWithResult(context.Background(), empty, "", "", "")
		require.Error(t, res.Err)
	})

	t.Run("zero_byte_remote", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK) // no body at all
		}))
		defer srv.Close()

		r, err := NewRulesLoader(&cmd.Config{HTTPTimeout: 3}, ModeOnlyRemote)
		require.NoError(t, err)

		res := r.LoadWithResult(context.Background(), "", "", srv.URL, "")
		require.Error(t, res.Err, "an empty response body must fail the source")
	})

	t.Run("zero_byte_source_falls_through_to_the_next", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "a.json"), []byte{}, 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "b.json"), []byte(oneUserJSON), 0o600))

		r, err := NewRulesLoader(nil, ModeOnlyLocal)
		require.NoError(t, err)

		res := r.LoadWithResult(context.Background(), "", dir, "", "")
		require.NoError(t, res.Err)
		require.Len(t, res.Users, 1, "the empty source fails and the next one answers")
		assert.Equal(t, "13800138000", res.Users[0].Phone)
	})
}

// TestEmptyArrayIsAValidEmptyList is the other half of the wrapper: an explicit
// "[]" is content, not an absent source, and must still load as an empty list.
func TestEmptyArrayIsAValidEmptyList(t *testing.T) {
	t.Run("file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "empty-array.json")
		require.NoError(t, os.WriteFile(path, []byte(`[]`), 0o600))

		r, err := NewRulesLoader(nil, ModeOnlyLocal)
		require.NoError(t, err)

		users, err := r.FromFile(context.Background(), path)
		require.NoError(t, err)
		assert.Empty(t, users)

		res := r.LoadWithResult(context.Background(), path, "", "", "")
		require.NoError(t, res.Err)
		assert.Empty(t, res.Users)
	})

	t.Run("remote", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			if _, err := w.Write([]byte(`[]`)); err != nil {
				t.Errorf("write response: %v", err)
			}
		}))
		defer srv.Close()

		r, err := NewRulesLoader(&cmd.Config{HTTPTimeout: 3}, ModeOnlyRemote)
		require.NoError(t, err)

		res := r.LoadWithResult(context.Background(), "", "", srv.URL, "")
		require.NoError(t, res.Err)
		assert.Empty(t, res.Users)
	})
}

// TestRemoteRequestCarriesTraceparent covers the propagator: parser-kit v1
// injected the global OpenTelemetry propagator on every remote fetch, and v3
// only does so when the source is built with one.
func TestRemoteRequestCarriesTraceparent(t *testing.T) {
	var gotTraceparent atomic.Pointer[string]
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		header := req.Header.Get("traceparent")
		gotTraceparent.Store(&header)
		if _, err := w.Write([]byte(oneUserJSON)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer srv.Close()

	// Install a global propagator, as tracing-kit does at startup.
	prev := otel.GetTextMapPropagator()
	defer otel.SetTextMapPropagator(prev)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	traceID, err := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	require.NoError(t, err)
	spanID, err := trace.SpanIDFromHex("00f067aa0ba902b7")
	require.NoError(t, err)
	ctx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	}))

	r, err := NewRulesLoader(&cmd.Config{HTTPTimeout: 3}, ModeOnlyRemote)
	require.NoError(t, err)

	res := r.LoadWithResult(ctx, "", "", srv.URL, "")
	require.NoError(t, res.Err)
	require.Len(t, res.Users, 1)

	traceparent := gotTraceparent.Load()
	require.NotNil(t, traceparent, "the remote source was never fetched")
	require.NotEmpty(t, *traceparent, "remote fetch must carry the global trace context")
	assert.Equal(t, "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01", *traceparent)
}

// TestRemoteRequestWithoutTracingSendsNoTraceparent is the companion: with no
// global propagator configured, otelprop.Global() resolves OpenTelemetry's
// no-op default and adds nothing, exactly as an untraced v1 process did.
func TestRemoteRequestWithoutTracingSendsNoTraceparent(t *testing.T) {
	var gotTraceparent atomic.Pointer[string]
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		header := req.Header.Get("traceparent")
		gotTraceparent.Store(&header)
		if _, err := w.Write([]byte(oneUserJSON)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer srv.Close()

	prev := otel.GetTextMapPropagator()
	defer otel.SetTextMapPropagator(prev)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator())

	r, err := NewRulesLoader(&cmd.Config{HTTPTimeout: 3}, ModeOnlyRemote)
	require.NoError(t, err)

	res := r.LoadWithResult(context.Background(), "", "", srv.URL, "")
	require.NoError(t, res.Err)

	traceparent := gotTraceparent.Load()
	require.NotNil(t, traceparent, "the remote source was never fetched")
	assert.Empty(t, *traceparent)
}

// TestInvalidRemoteURLFailsOnlyThatSource covers the deferred construction
// error. remotesource.New validates the URL up front, so a bad one must not
// take BuildSources -- and with it every other source -- down with it.
func TestInvalidRemoteURLFailsOnlyThatSource(t *testing.T) {
	local := filepath.Join(t.TempDir(), "rules.json")
	require.NoError(t, os.WriteFile(local, []byte(oneUserJSON), 0o600))

	for _, badURL := range []string{"not-a-valid-url", "ftp://example.com/rules.json", "http://"} {
		t.Run(badURL, func(t *testing.T) {
			t.Run("sources_are_still_built", func(t *testing.T) {
				sources := BuildSources(local, "", badURL, "", ModeLocalFirst, BuildRemoteOptions(nil))
				require.Len(t, sources, 2, "the bad remote is still a source, it just fails when read")
			})

			t.Run("merge_mode_still_loads_local", func(t *testing.T) {
				r, err := NewRulesLoader(&cmd.Config{HTTPTimeout: 3}, ModeLocalFirst)
				require.NoError(t, err)

				res := r.LoadWithResult(context.Background(), local, "", badURL, "")
				require.NoError(t, res.Err)
				require.Len(t, res.Users, 1)
				assert.Equal(t, "13800138000", res.Users[0].Phone)
			})

			t.Run("only_remote_reports_the_error", func(t *testing.T) {
				r, err := NewRulesLoader(&cmd.Config{HTTPTimeout: 3}, ModeOnlyRemote)
				require.NoError(t, err)

				res := r.LoadWithResult(context.Background(), "", "", badURL, "")
				require.Error(t, res.Err)
			})
		})
	}
}

// TestInsecureSkipVerify covers the TLS switch that moved from LoadOptions onto
// the remote source: a self-signed server must be refused when it is off and
// accepted when it is on.
func TestInsecureSkipVerify(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if _, err := w.Write([]byte(oneUserJSON)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer srv.Close()

	t.Run("off_rejects_self_signed", func(t *testing.T) {
		r, err := NewRulesLoader(&cmd.Config{HTTPTimeout: 3, HTTPInsecureTLS: false}, ModeOnlyRemote)
		require.NoError(t, err)

		res := r.LoadWithResult(context.Background(), "", "", srv.URL, "")
		require.Error(t, res.Err)
		var unknownAuthority x509.UnknownAuthorityError
		var certInvalid x509.CertificateInvalidError
		var hostnameErr x509.HostnameError
		assert.True(t,
			errors.As(res.Err, &unknownAuthority) || errors.As(res.Err, &certInvalid) || errors.As(res.Err, &hostnameErr),
			"expected a certificate verification failure, got: %v", res.Err)
	})

	t.Run("on_accepts_self_signed", func(t *testing.T) {
		r, err := NewRulesLoader(&cmd.Config{HTTPTimeout: 3, HTTPInsecureTLS: true}, ModeOnlyRemote)
		require.NoError(t, err)

		res := r.LoadWithResult(context.Background(), "", "", srv.URL, "")
		require.NoError(t, res.Err)
		require.Len(t, res.Users, 1)
	})

	// Guard against the switch being read from the wrong place: a loader built
	// without the flag must not inherit it from another loader.
	t.Run("flag_is_per_loader", func(t *testing.T) {
		insecure := BuildRemoteOptions(&cmd.Config{HTTPInsecureTLS: true})
		secure := BuildRemoteOptions(&cmd.Config{HTTPInsecureTLS: false})
		assert.True(t, insecure.InsecureSkipVerify)
		assert.False(t, secure.InsecureSkipVerify)
	})
}
