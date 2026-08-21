package cache

import (
	"bytes"
	"strings"
	"sync"
	"testing"

	loggerkit "github.com/soulteary/logger-kit"
	"github.com/soulteary/warden/internal/define"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureLog swaps the package logger for one writing to buf for the duration of fn.
// It restores the original logger afterwards. Serialized via a mutex because the
// package logger is a shared global.
var logSwapMu sync.Mutex

func captureLog(t *testing.T, fn func()) string {
	t.Helper()
	logSwapMu.Lock()
	defer logSwapMu.Unlock()

	var buf bytes.Buffer
	orig := log
	log = loggerkit.New(loggerkit.Config{
		Level:       loggerkit.DebugLevel,
		Format:      loggerkit.FormatJSON,
		ServiceName: "warden-test",
		Output:      &buf,
	})
	defer func() { log = orig }()
	fn()
	return buf.String()
}

// TestValidateUser_DoesNotLogRawPhoneOrMail is the log-capture privacy test required by
// the plan: an invalid user must be logged with masked identifiers only, never the raw
// phone/mail (which the underlying validator error would otherwise embed).
func TestValidateUser_DoesNotLogRawPhoneOrMail(t *testing.T) {
	const rawPhone = "13800138000"
	const rawMail = "alice@example.com"

	out := captureLog(t, func() {
		// Invalid phone (letters) with a valid-looking mail forces the phone branch,
		// whose validator error embeds the raw phone value if logged directly.
		err := validateUser(define.AllowListUser{
			Phone: "not-a-phone-xyz",
			Mail:  rawMail,
		})
		require.Error(t, err)

		// Invalid email path.
		err = validateUser(define.AllowListUser{
			Phone: rawPhone,
			Mail:  "not-an-email",
		})
		_ = err
	})

	require.NotEmpty(t, out, "expected log output to inspect")
	assert.NotContains(t, out, rawPhone, "raw phone must never be logged")
	assert.NotContains(t, out, rawMail, "raw mail must never be logged")
	assert.NotContains(t, out, "not-a-phone-xyz", "raw invalid phone value must not be logged")
	assert.NotContains(t, out, "not-an-email", "raw invalid mail value must not be logged")
	// A stable, non-sensitive reason code should be present instead.
	assert.True(t,
		strings.Contains(out, "invalid_phone") || strings.Contains(out, "invalid_email"),
		"expected a stable non-PII reason code in the log output")
}
