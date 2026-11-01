package warden

import (
	"errors"
	"net/http"
	"regexp"
	"time"
)

// Negotiated response metadata is separate from the compatibility JSON schema.
const (
	HeaderSnapshotMetadata   = "Warden-Snapshot-Metadata"
	HeaderSnapshotVersion    = "Warden-Snapshot-Version"
	HeaderSourceSucceededAt  = "Warden-Source-Succeeded-At"
	HeaderSnapshotValidUntil = "Warden-Snapshot-Valid-Until"
)

// Source freshness errors never include raw response headers or user records.
var (
	ErrSourceUnknown = errors.New("warden: source freshness evidence unavailable or invalid")
	ErrSourceStale   = errors.New("warden: source freshness deadline reached")
	ErrSourceFuture  = errors.New("warden: source timestamp exceeds clock allowance")
)

var sourceVersionPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

func singleSourceHeader(headers http.Header, name string) (string, bool) {
	values := headers.Values(name)
	if len(values) != 1 || len(values[0]) > 128 {
		return "", false
	}
	return values[0], true
}

func sourceFreshnessError(reason error) error {
	return NewError(ErrCodeSourceFreshness, "source freshness requirement not met", reason)
}

func (c *Client) sourceDeadline(headers http.Header) (time.Time, error) {
	metadata, ok := singleSourceHeader(headers, HeaderSnapshotMetadata)
	if !ok || metadata != "v1" {
		return time.Time{}, sourceFreshnessError(ErrSourceUnknown)
	}
	version, ok := singleSourceHeader(headers, HeaderSnapshotVersion)
	if !ok || !sourceVersionPattern.MatchString(version) {
		return time.Time{}, sourceFreshnessError(ErrSourceUnknown)
	}
	sourceRaw, sourceOK := singleSourceHeader(headers, HeaderSourceSucceededAt)
	untilRaw, untilOK := singleSourceHeader(headers, HeaderSnapshotValidUntil)
	source, sourceErr := time.Parse(time.RFC3339Nano, sourceRaw)
	until, untilErr := time.Parse(time.RFC3339Nano, untilRaw)
	if !sourceOK || !untilOK || sourceErr != nil || untilErr != nil || source.IsZero() || !until.After(source) {
		return time.Time{}, sourceFreshnessError(ErrSourceUnknown)
	}
	now := c.sourceClock.Now()
	if source.After(now.Add(c.sourceClockSkew)) {
		return time.Time{}, sourceFreshnessError(ErrSourceFuture)
	}
	deadline := source.Add(c.sourceMaxAge)
	if until.Before(deadline) {
		deadline = until
	}
	deadline = deadline.Add(-c.sourceClockSkew)
	if !now.Before(deadline) {
		return time.Time{}, sourceFreshnessError(ErrSourceStale)
	}
	return deadline, nil
}

func (c *Client) checkSourceDeadline(deadline time.Time) error {
	if c.sourceMaxAge > 0 && !c.sourceClock.Now().Before(deadline) {
		return sourceFreshnessError(ErrSourceStale)
	}
	return nil
}

func (c *Client) checkDataResponse(resp *http.Response) (time.Time, error) {
	statusErr := c.checkResponseStatus(resp)
	if c.sourceMaxAge <= 0 {
		return time.Time{}, statusErr
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		return time.Time{}, statusErr
	}
	deadline, err := c.sourceDeadline(resp.Header)
	if err != nil {
		return time.Time{}, err
	}
	return deadline, statusErr
}
