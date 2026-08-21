package remote

import (
	"sync"

	"github.com/soulteary/warden/internal/logger"
)

// onceReporter emits a signal at most once until Reset is called. Used to de-duplicate
// deprecation warnings so a busy background refresh loop does not flood logs.
type onceReporter struct {
	mu   sync.Mutex
	done bool
}

func newOnceReporter() *onceReporter { return &onceReporter{} }

// fire returns true the first time it is called (and after Reset), false otherwise.
func (o *onceReporter) fire() bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.done {
		return false
	}
	o.done = true
	return true
}

// Reset re-arms the reporter. Intended for tests.
func (o *onceReporter) Reset() {
	o.mu.Lock()
	o.done = false
	o.mu.Unlock()
}

// LegacyEncryptionObserver is invoked once (deduped) the first time the
// deprecated legacy encryption format is used. It lets callers (main) record a
// deprecation metric without this package importing the metrics package.
var LegacyEncryptionObserver func()

// reportLegacyDeprecation logs a single deprecation warning for the legacy v1 format.
// The message contains no secrets or payloads.
func reportLegacyDeprecation() {
	if legacyDeprecationOnce.fire() {
		logger.GetLoggerKit().Warn().
			Str("format", string(FormatLegacy)).
			Str("recommended", string(FormatV2)).
			Msg("remote: legacy unauthenticated encryption format in use; migrate to envelope v2")
		if LegacyEncryptionObserver != nil {
			LegacyEncryptionObserver()
		}
	}
}

// ResetLegacyDeprecationForTest re-arms the legacy deprecation reporter (test helper).
func ResetLegacyDeprecationForTest() {
	legacyDeprecationOnce.Reset()
}
