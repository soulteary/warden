package warden

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	httpkit "github.com/soulteary/http-kit/v2"
	otelprop "github.com/soulteary/http-kit/v2/otelprop"
)

// Client is the Warden API client.
//
//nolint:govet // fieldalignment: field order has been optimized, but not further adjusted to maintain API compatibility
type Client struct {
	httpClient               *httpkit.Client
	baseURL                  string
	apiKey                   string
	cache                    *Cache
	sourceMaxAge             time.Duration
	sourceClockSkew          time.Duration
	sourceClock              Clock
	logger                   Logger
	retry                    *RetryOptions
	signer                   *hmacSigner
	maxResponseBytes         int64
	cacheInvalidationChannel <-chan struct{}
	stopCacheListener        context.CancelFunc
	cacheListenerWg          sync.WaitGroup
}

// NewClient creates a new Warden API client with the provided options.
func NewClient(opts *Options) (*Client, error) {
	if opts == nil {
		opts = DefaultOptions()
	}

	// Validate options
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	// Create HTTP client using http-kit
	clientOpts := &httpkit.Options{
		BaseURL: opts.BaseURL,
		Timeout: opts.Timeout,
		// http-kit v2 removed Client.InjectTraceContext, which resolved the
		// global OpenTelemetry propagator on every call. otelprop.Global()
		// resolves it just as late, and Client.Do applies it to every attempt,
		// so outbound requests carry the same trace headers as before.
		Propagator: otelprop.Global(),
	}
	if opts.Transport != nil {
		clientOpts.Transport = opts.Transport
	}
	// Apply a custom TLS config (client certs / root CAs) when provided. We attach it
	// to a transport so it reaches the underlying http.Client.
	if opts.TLSConfig != nil {
		// Work on a copy: the caller's transport may be shared with other
		// clients, which must not inherit this client's TLS configuration.
		tr := &http.Transport{}
		if opts.Transport != nil {
			tr = opts.Transport.Clone()
		}
		tr.TLSClientConfig = opts.TLSConfig
		clientOpts.Transport = tr
	}

	httpClient, err := httpkit.NewClient(clientOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	// Redirects must not forward API keys, signatures or client certificates,
	// including redirects to another path on the same host.
	httpClient.GetHTTPClient().CheckRedirect = func(*http.Request, []*http.Request) error { return ErrRedirect }

	// Set default retry options if not provided
	retry := opts.Retry
	if retry == nil {
		retry = DefaultRetryOptions()
	}

	maxResp := opts.MaxResponseBytes
	if maxResp == 0 {
		maxResp = DefaultMaxResponseBytes
	}

	sourceClock := opts.Clock
	if sourceClock == nil {
		sourceClock = systemClock{}
	}
	client := &Client{
		sourceMaxAge: opts.SourceMaxAge, sourceClockSkew: opts.SourceClockSkew, sourceClock: sourceClock,
		httpClient:               httpClient,
		baseURL:                  opts.BaseURL,
		apiKey:                   opts.APIKey,
		cache:                    NewCache(opts.CacheTTL),
		logger:                   opts.Logger,
		retry:                    retry,
		signer:                   newHMACSigner(opts.HMACKeyID, opts.HMACSecret, opts.Clock),
		maxResponseBytes:         maxResp,
		cacheInvalidationChannel: opts.CacheInvalidationChannel,
	}

	if opts.SourceMaxAge > 0 {
		client.cache.now = sourceClock.Now
	}

	// Start cache invalidation listener if channel is provided
	if opts.CacheInvalidationChannel != nil {
		ctx, cancel := context.WithCancel(context.Background())
		client.stopCacheListener = cancel
		client.cacheListenerWg.Add(1)
		go client.listenForCacheInvalidation(ctx)
	}

	if base, err := url.Parse(opts.BaseURL); err == nil && opts.sendsAPIKeyInClear(base) {
		client.logger.Warn("Warden SDK: the API key is sent over plain HTTP to " + base.Host + "; use an https URL or HMAC signing")
	}

	client.logger.Debugf("Warden SDK client created: URL=%s, APIKey=%v, Retry=%v, Transport=%v",
		opts.BaseURL, opts.APIKey != "", retry.MaxRetries > 0, opts.Transport != nil)

	return client, nil
}

// listenForCacheInvalidation listens for cache invalidation signals on the configured channel.
func (c *Client) listenForCacheInvalidation(ctx context.Context) {
	defer c.cacheListenerWg.Done()
	for {
		select {
		case <-ctx.Done():
			c.logger.Debug("Cache invalidation listener stopped")
			return
		case <-c.cacheInvalidationChannel:
			c.logger.Debug("Cache invalidation signal received, clearing cache")
			c.cache.Clear()
		}
	}
}

// Close stops the cache invalidation listener and releases resources.
// This should be called when the client is no longer needed.
func (c *Client) Close() {
	if c.stopCacheListener != nil {
		c.stopCacheListener()
		c.cacheListenerWg.Wait()
	}
}

// GetUsers fetches the user list from Warden API.
// If pagination parameters are not provided, returns all users.
func (c *Client) GetUsers(ctx context.Context) ([]AllowListUser, error) {
	// Check cache first
	users, request := c.cache.beginRead()
	if users != nil {
		c.logger.Debug("Using cached user list from Warden")
		return users, nil
	}

	// Build request URL
	reqURL := fmt.Sprintf("%s/", c.baseURL)
	c.logger.Debugf("Fetching users from Warden API: %s", reqURL)

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, http.NoBody)
	if err != nil {
		return nil, NewError(ErrCodeRequestFailed, "failed to create request", err)
	}

	// Add authentication headers if configured
	c.addAuthHeaders(req)

	// Make request with retry
	resp, err := c.doRequestWithRetry(ctx, req)
	if err != nil {
		c.logger.Errorf("Failed to fetch users from Warden API: code=%s", logErrorCode(err))
		return nil, err
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close() //nolint:errcheck // Ignoring error in defer is safe
		}
	}()

	// Check status code
	deadline, responseErr := c.checkDataResponse(resp)
	if responseErr != nil {
		return nil, responseErr
	}

	// Parse response
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil, NewError(ErrCodeInvalidResponse, "failed to decode response", err)
	}

	// Recheck after decoding; receipt time never establishes new source evidence.
	if err := c.checkSourceDeadline(deadline); err != nil {
		return nil, err
	}
	if err := c.cache.commitRead(users, deadline, request); err != nil {
		return nil, err
	}

	c.logger.Debugf("Fetched %d users from Warden API", len(users))

	return users, nil
}

// GetUsersPaginated fetches a paginated user list from Warden API.
func (c *Client) GetUsersPaginated(ctx context.Context, page, pageSize int) (*PaginatedResponse, error) {
	if page < 1 {
		return nil, NewError(ErrCodeInvalidConfig, "page must be greater than 0", nil)
	}
	if pageSize < 1 {
		return nil, NewError(ErrCodeInvalidConfig, "pageSize must be greater than 0", nil)
	}

	// Build request URL with pagination parameters
	reqURL, err := url.Parse(fmt.Sprintf("%s/", c.baseURL))
	if err != nil {
		return nil, NewError(ErrCodeInvalidConfig, "invalid base URL", err)
	}

	q := reqURL.Query()
	q.Set("page", fmt.Sprintf("%d", page))
	q.Set("page_size", fmt.Sprintf("%d", pageSize))
	reqURL.RawQuery = q.Encode()

	c.logger.Debugf("Fetching paginated users from Warden API: %s (page=%d, pageSize=%d)", reqURL.String(), page, pageSize)

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), http.NoBody)
	if err != nil {
		return nil, NewError(ErrCodeRequestFailed, "failed to create request", err)
	}

	// Add authentication headers if configured
	c.addAuthHeaders(req)

	// Make request with retry
	resp, err := c.doRequestWithRetry(ctx, req)
	if err != nil {
		c.logger.Errorf("Failed to fetch paginated users from Warden API: code=%s", logErrorCode(err))
		return nil, err
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close() //nolint:errcheck // Ignoring error in defer is safe
		}
	}()

	// Check status code
	deadline, responseErr := c.checkDataResponse(resp)
	if responseErr != nil {
		return nil, responseErr
	}

	// Parse response
	var paginatedResp PaginatedResponse
	if err := json.NewDecoder(resp.Body).Decode(&paginatedResp); err != nil {
		return nil, NewError(ErrCodeInvalidResponse, "failed to decode paginated response", err)
	}

	if err := c.checkSourceDeadline(deadline); err != nil {
		return nil, err
	}

	c.logger.Debugf("Fetched paginated users: page=%d, pageSize=%d, total=%d", page, pageSize, paginatedResp.Pagination.Total)

	return &paginatedResp, nil
}

// CheckUserInList checks if a user (by phone or mail) is in the allow list and has active status.
// Returns false if the user is not found, has inactive/suspended status, or if there's an error.
// This method uses GetUserByIdentifier for better performance and includes status validation.
// If both phone and mail are provided, phone takes priority. If phone lookup fails with NotFound error,
// it will fall back to mail lookup.
func (c *Client) CheckUserInList(ctx context.Context, phone, mail string) bool {
	// Normalize input
	phone = strings.TrimSpace(phone)
	mail = strings.TrimSpace(strings.ToLower(mail))

	// If both phone and mail are provided, prioritize phone
	// GetUserByIdentifier requires exactly one identifier
	var user *AllowListUser
	var err error

	switch {
	case phone != "":
		// Try phone first if provided
		user, err = c.GetUserByIdentifier(ctx, phone, "", "")
		if err != nil {
			// Check if error is NotFound and mail is available for fallback
			if sdkErr, ok := err.(*Error); ok && sdkErr.Code == ErrCodeNotFound && mail != "" {
				c.logger.Debugf("User not found by phone, falling back to mail: phone=%s, mail=%s", sanitizePhone(phone), sanitizeEmail(mail))
				// Fall back to mail lookup
				user, err = c.GetUserByIdentifier(ctx, "", mail, "")
			} else {
				// Log error but don't expose details to caller (security: don't reveal if user exists)
				c.logger.Debugf("Failed to get user from Warden API: code=%s (phone=%s, mail=%s)", logErrorCode(err), sanitizePhone(phone), sanitizeEmail(mail))
				return false
			}
		} else if user != nil && !user.IsActive() {
			// User found by phone but not active - don't try mail (user exists but inactive)
			c.logger.Warnf("User status is not active: phone=%s, mail=%s, status=%s", sanitizePhone(phone), sanitizeEmail(mail), user.Status)
			return false
		}
	case mail != "":
		// Fall back to mail if phone is empty
		user, err = c.GetUserByIdentifier(ctx, "", mail, "")
	default:
		// Both are empty
		c.logger.Debug("CheckUserInList called with both phone and mail empty")
		return false
	}

	if err != nil {
		// Log error but don't expose details to caller (security: don't reveal if user exists)
		c.logger.Debugf("Failed to get user from Warden API: code=%s (phone=%s, mail=%s)", logErrorCode(err), sanitizePhone(phone), sanitizeEmail(mail))
		return false
	}

	// Check if user status is active
	if !user.IsActive() {
		c.logger.Warnf("User status is not active: phone=%s, mail=%s, status=%s", sanitizePhone(phone), sanitizeEmail(mail), user.Status)
		return false
	}

	c.logger.Debugf("User found and active: phone=%s, mail=%s, user_id=%s", sanitizePhone(phone), sanitizeEmail(mail), sanitizeString(user.UserID))
	return true
}

// CheckUserInListWithError is CheckUserInList that reports why the answer is
// not available. It returns (true, nil) for an active user, (false, nil) for a
// user that is not on the list or not active, and (false, err) when Warden
// could not be asked or refused to answer: a transport failure, a rejected
// credential (ErrCodeUnauthorized), a rate limit (ErrCodeRateLimited) or a
// server error. A caller can then tell "denied" from "unknown" and decide how
// to treat an outage, where CheckUserInList reports false for both.
//
// The lookup order is the one CheckUserInList uses: phone first; when the
// phone is not found and a mail is given, the mail. A user found by phone
// that is not active is denied without consulting the mail.
func (c *Client) CheckUserInListWithError(ctx context.Context, phone, mail string) (bool, error) {
	phone = strings.TrimSpace(phone)
	mail = strings.TrimSpace(strings.ToLower(mail))
	if phone == "" && mail == "" {
		return false, NewError(ErrCodeInvalidConfig, "at least one of phone or mail must be provided", nil)
	}

	if phone != "" {
		user, found, err := c.findUser(ctx, phone, "")
		if err != nil {
			return false, err
		}
		if found {
			return user.IsActive(), nil
		}
		if mail == "" {
			return false, nil
		}
	}
	user, found, err := c.findUser(ctx, "", mail)
	if err != nil || !found {
		return false, err
	}
	return user.IsActive(), nil
}

// findUser looks a user up and reports "not found" as a result, not an error.
// A user response without a status field is an error: the answer to "is this
// user active" is not in it.
func (c *Client) findUser(ctx context.Context, phone, mail string) (user *AllowListUser, found bool, err error) {
	user, hasStatus, err := c.getUser(ctx, phone, mail, "")
	if err == nil {
		if !hasStatus {
			return nil, false, NewError(ErrCodeInvalidResponse, missingStatusMessage, nil)
		}
		return user, true, nil
	}
	var sdkErr *Error
	if errors.As(err, &sdkErr) && sdkErr.Code == ErrCodeNotFound {
		return nil, false, nil
	}
	return nil, false, err
}

// LookupDestination holds the addresses a one-time code can be sent to.
type LookupDestination struct {
	Email string `json:"email,omitempty"`
	Phone string `json:"phone,omitempty"`
}

// LookupResult is the response of GET /v1/lookup.
type LookupResult struct {
	UserID         string            `json:"user_id"`
	Destination    LookupDestination `json:"destination"`
	Status         string            `json:"status"`
	ChannelHint    string            `json:"channel_hint,omitempty"`
	Name           string            `json:"name,omitempty"`
	DingtalkUserID string            `json:"dingtalk_userid,omitempty"`
}

// IsActive returns true if the user status is "active".
func (r *LookupResult) IsActive() bool {
	return r.Status == "active"
}

// Lookup resolves one identifier through GET /v1/lookup. The server treats an
// identifier containing "@" as a mail address; any other value is tried as a
// phone number and then as a user ID. An unknown identifier yields an error
// with ErrCodeNotFound. The result is not cached.
func (c *Client) Lookup(ctx context.Context, identifier string) (*LookupResult, error) {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return nil, NewError(ErrCodeInvalidConfig, "identifier must be provided", nil)
	}
	reqURL := strings.TrimSuffix(c.baseURL, "/") + "/v1/lookup?" + url.Values{"identifier": {identifier}}.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, http.NoBody)
	if err != nil {
		return nil, NewError(ErrCodeRequestFailed, "failed to create request", err)
	}
	c.addAuthHeaders(req)

	resp, err := c.doRequestWithRetry(ctx, req)
	if err != nil {
		c.logger.Errorf("Failed to look up user from Warden API: code=%s", logErrorCode(err))
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close() //nolint:errcheck // Ignoring error in defer is safe
	}()

	deadline, responseErr := c.checkDataResponse(resp)
	if responseErr != nil {
		return nil, responseErr
	}
	var result LookupResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, NewError(ErrCodeInvalidResponse, "failed to decode response", err)
	}
	if err := c.checkSourceDeadline(deadline); err != nil {
		return nil, err
	}
	return &result, nil
}

// missingStatusMessage describes a user response that carries no status field.
// A Warden server always returns the field unless its RESPONSE_FIELDS
// allowlist leaves it out.
const missingStatusMessage = "the user response carries no status field: the server's RESPONSE_FIELDS must include status for this check"

// GetUserByIdentifier fetches a single user by phone, mail, or user_id.
// Only one identifier should be provided.
//
// The server returns only the fields its RESPONSE_FIELDS allowlist names. A
// field that is left out is empty in the result; without status, IsActive
// reports false.
func (c *Client) GetUserByIdentifier(ctx context.Context, phone, mail, userID string) (*AllowListUser, error) {
	user, _, err := c.getUser(ctx, phone, mail, userID)
	return user, err
}

// getUser is GetUserByIdentifier that also reports whether the response
// carried a status field.
func (c *Client) getUser(ctx context.Context, phone, mail, userID string) (result *AllowListUser, hasStatus bool, err error) {
	identifierCount := 0
	for _, value := range []string{phone, mail, userID} {
		if value != "" {
			identifierCount++
		}
	}

	if identifierCount == 0 {
		return nil, false, NewError(ErrCodeInvalidConfig, "at least one identifier (phone, mail, or user_id) must be provided", nil)
	}
	if identifierCount > 1 {
		return nil, false, NewError(ErrCodeInvalidConfig, "only one identifier (phone, mail, or user_id) should be provided", nil)
	}

	reqURL := c.userRequestURL(phone, mail, userID)

	// Sanitize URL in logs to mask sensitive query parameters
	sanitizedURL := sanitizeURLString(reqURL)
	c.logger.Debugf("Fetching user from Warden API: %s", sanitizedURL)

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, http.NoBody)
	if err != nil {
		return nil, false, NewError(ErrCodeRequestFailed, "failed to create request", err)
	}

	// Add authentication headers if configured
	c.addAuthHeaders(req)

	// Make request with retry
	resp, err := c.doRequestWithRetry(ctx, req)
	if err != nil {
		c.logger.Errorf("Failed to fetch user from Warden API: code=%s", logErrorCode(err))
		return nil, false, err
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close() //nolint:errcheck // Ignoring error in defer is safe
		}
	}()

	// Check status code
	deadline, responseErr := c.checkDataResponse(resp)
	if responseErr != nil {
		return nil, false, responseErr
	}

	// Parse response. Status is decoded through a pointer so that a response
	// without the field can be told from one with an empty value.
	var decoded struct {
		Status *string `json:"status"`
		AllowListUser
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, false, NewError(ErrCodeInvalidResponse, "failed to decode response", err)
	}
	user := decoded.AllowListUser
	hasStatus = decoded.Status != nil
	if hasStatus {
		user.Status = *decoded.Status
	} else {
		c.logger.Warn("Warden user response carries no status field (the server's RESPONSE_FIELDS omits it); the user is treated as not active")
	}

	if err := c.checkSourceDeadline(deadline); err != nil {
		return nil, false, err
	}

	c.logger.Debugf("Fetched user from Warden API: user_id=%s, phone=%s, mail=%s", sanitizeString(user.UserID), sanitizePhone(user.Phone), sanitizeEmail(user.Mail))

	return &user, hasStatus, nil
}

// ClearCache clears the internal cache.
func (c *Client) ClearCache() {
	c.cache.Clear()
	c.logger.Debug("Cache cleared")
}

// InvalidateCache is an alias for ClearCache for consistency with event-driven invalidation.
func (c *Client) InvalidateCache() {
	c.ClearCache()
}

// isIdempotentMethod reports whether an HTTP method is safe to retry. Retrying
// non-idempotent methods (POST/PATCH) risks duplicate side effects, so we only
// retry idempotent verbs. All current SDK calls are GET.
func isIdempotentMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

// isRetryableError checks if an error should trigger a retry.
func (c *Client) isRetryableError(err error, statusCode int) bool {
	if errors.Is(err, ErrRedirect) {
		return false
	}

	if c.retry == nil || c.retry.MaxRetries == 0 {
		return false
	}

	// Network errors are always retryable
	if err != nil {
		return true
	}

	// Never retry on client errors (4xx) except for specific server errors
	// Only retry on server errors (5xx)
	if statusCode >= 400 && statusCode < 500 {
		return false
	}

	// Check if status code is in retryable list
	for _, code := range c.retry.RetryableStatusCodes {
		if statusCode == code {
			return true
		}
	}

	return false
}

// calculateRetryDelay calculates the delay before a retry using exponential
// backoff. retryIndex is zero-based, so the first retry waits RetryDelay.
func (c *Client) calculateRetryDelay(retryIndex int) time.Duration {
	if c.retry == nil {
		return 0
	}

	if retryIndex < 0 {
		retryIndex = 0
	}
	if c.retry.RetryDelay <= 0 {
		return 0
	}
	delayFloat := float64(c.retry.RetryDelay) * math.Pow(c.retry.BackoffMultiplier, float64(retryIndex))
	// Apply the configured cap before converting to time.Duration. float64(MaxInt64)
	// rounds to 2^63, whose conversion can wrap to a negative duration.
	if c.retry.MaxRetryDelay > 0 && delayFloat >= float64(c.retry.MaxRetryDelay) {
		return c.retry.MaxRetryDelay
	}
	maxSafeFloat := math.Nextafter(float64(math.MaxInt64), 0)
	if math.IsInf(delayFloat, 1) || delayFloat >= maxSafeFloat {
		return time.Duration(math.MaxInt64)
	}

	return time.Duration(delayFloat)
}

// doRequestWithRetry performs an HTTP request with retry logic.
func (c *Client) doRequestWithRetry(ctx context.Context, req *http.Request) (*http.Response, error) {
	var lastErr error
	var lastResp *http.Response

	// Only idempotent methods are retried to avoid duplicate side effects.
	retryAllowed := isIdempotentMethod(req.Method)

	maxAttempts := c.retry.MaxRetries + 1
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			if err := c.waitBeforeRetry(ctx, req, attempt, maxAttempts); err != nil {
				return nil, err
			}
		}

		// Make the request
		if err := c.signRequest(req); err != nil {
			return nil, err
		}
		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			lastResp = nil
			if !retryAllowed || !c.isRetryableError(err, 0) {
				return nil, NewError(ErrCodeRequestFailed, "failed to execute request", err)
			}
			if attempt < c.retry.MaxRetries {
				continue
			}
			return nil, NewError(ErrCodeRequestFailed, "failed to execute request after retries", err)
		}

		// Check if status code is retryable
		if c.shouldRetryResponse(retryAllowed, resp.StatusCode, attempt) {
			// Close response body before retry
			if err := resp.Body.Close(); err != nil {
				c.logger.Debug("Failed to close response body before retry")
			}
			lastResp = nil
			lastErr = NewError(ErrCodeServerError, fmt.Sprintf("server error: status %d", resp.StatusCode), nil)
			continue
		}

		// Success or non-retryable error - return response
		// The caller will check the status code
		if err := c.applyResponseLimit(resp); err != nil {
			return nil, NewError(ErrCodeInvalidResponse, "response body exceeds configured limit", err)
		}
		return resp, nil
	}

	// All retries exhausted
	if lastResp != nil {
		return lastResp, nil
	}
	if lastErr != nil {
		return nil, NewError(ErrCodeRequestFailed, "request failed after all retries", lastErr)
	}
	return nil, NewError(ErrCodeRequestFailed, "request failed after all retries", nil)
}

// addAuthHeaders adds authentication headers to the request.
func (c *Client) addAuthHeaders(req *http.Request) {
	if c.sourceMaxAge > 0 {
		req.Header.Set(HeaderSnapshotMetadata, "v1")
	}
	if c.signer == nil && c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
		// Also support Authorization header
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
		c.logger.Debug("Added API key headers to Warden request")
	}
}

// applyResponseLimit reads at most limit+1 bytes so an oversized response is
// rejected explicitly rather than exposed as an ambiguous truncated body.
func (c *Client) applyResponseLimit(resp *http.Response) error {
	if resp == nil || resp.Body == nil {
		return nil
	}
	if c.maxResponseBytes < 0 {
		return nil
	}
	limit := c.maxResponseBytes
	if limit == 0 {
		limit = DefaultMaxResponseBytes
	}
	readLimit := limit
	if limit < math.MaxInt64 {
		readLimit = limit + 1
	}
	orig := resp.Body
	body, err := io.ReadAll(io.LimitReader(orig, readLimit))
	if closeErr := orig.Close(); closeErr != nil {
		c.logger.Debug("Failed to close response body after bounded read")
	}
	if err != nil {
		return err
	}
	if int64(len(body)) > limit {
		return ErrResponseTooLarge
	}
	resp.Body = io.NopCloser(bytes.NewReader(body))
	return nil
}

// checkResponseStatus checks the HTTP response status and returns an error if not OK.
func (c *Client) checkResponseStatus(resp *http.Response) error {
	if resp.StatusCode == http.StatusOK {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Warnf("Warden API returned non-200 status: %d, failed to read body", resp.StatusCode)
		body = []byte("")
	} else {
		c.logger.Warnf("Warden API returned non-200 status: %d", resp.StatusCode)
	}

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return NewError(ErrCodeUnauthorized, c.unauthorizedMessage(), nil)
	case http.StatusTooManyRequests:
		rateLimited := NewError(ErrCodeRateLimited, "rate limited by Warden", nil)
		rateLimited.RetryAfter = parseRetryAfter(resp.Header.Get("Retry-After"), time.Now())
		return rateLimited
	case http.StatusNotFound:
		return NewError(ErrCodeNotFound, "not found", nil)
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable:
		return NewError(ErrCodeServerError, fmt.Sprintf("server error: status %d", resp.StatusCode), nil)
	default:
		return NewError(ErrCodeRequestFailed, fmt.Sprintf("warden API returned status %d: %s", resp.StatusCode, string(body)), nil)
	}
}

// unauthorizedMessage describes a 401 in terms of the credential this client
// sent, so a rejected HMAC signature is not reported as a wrong API key.
func (c *Client) unauthorizedMessage() string {
	switch {
	case c.signer != nil:
		return "unauthorized: the HMAC signature was rejected (check the key ID, the secret and the clock)"
	case c.apiKey != "":
		return "unauthorized: invalid API key"
	default:
		return "unauthorized: the request carried no API key or HMAC signature"
	}
}

// parseRetryAfter reads a Retry-After header: a number of seconds or an HTTP
// date. An absent, malformed or past value yields zero.
func parseRetryAfter(value string, now time.Time) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(value); err == nil {
		if seconds < 0 {
			return 0
		}
		return time.Duration(seconds) * time.Second
	}
	if at, err := http.ParseTime(value); err == nil && at.After(now) {
		return at.Sub(now)
	}
	return 0
}

func (c *Client) signRequest(req *http.Request) error {
	if c.signer != nil {
		if err := c.signer.sign(req); err != nil {
			return NewError(ErrCodeRequestFailed, "failed to sign request", err)
		}
	}
	return nil
}

func (c *Client) userRequestURL(phone, mail, userID string) string {
	// Build request URL
	reqURL := strings.TrimSuffix(c.baseURL, "/") + "/user"
	params := url.Values{}
	switch {
	case phone != "":
		params.Set("phone", phone)
	case mail != "":
		params.Set("mail", mail)
	case userID != "":
		params.Set("user_id", userID)
	}
	if len(params) > 0 {
		reqURL += "?" + params.Encode()
	}

	return reqURL
}

func (c *Client) waitBeforeRetry(ctx context.Context, req *http.Request, attempt, maxAttempts int) error {
	// Calculate delay before retry
	delay := c.calculateRetryDelay(attempt - 1)
	c.logger.Debugf("Retrying request (attempt %d/%d) after %v: method=%s",
		attempt+1, maxAttempts, delay, req.Method)

	// Wait before retry
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(delay):
	}
	return nil
}

func (c *Client) shouldRetryResponse(allowed bool, status, attempt int) bool {
	return allowed && c.isRetryableError(nil, status) && attempt < c.retry.MaxRetries
}
