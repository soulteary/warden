package warden

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	httpkit "github.com/soulteary/http-kit"
)

// Client is the Warden API client.
//
//nolint:govet // fieldalignment: field order has been optimized, but not further adjusted to maintain API compatibility
type Client struct {
	httpClient               *httpkit.Client
	baseURL                  string
	apiKey                   string
	cache                    *Cache
	logger                   Logger
	retry                    *RetryOptions
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
	}
	if opts.Transport != nil {
		clientOpts.Transport = opts.Transport
	}

	httpClient, err := httpkit.NewClient(clientOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	// Set default retry options if not provided
	retry := opts.Retry
	if retry == nil {
		retry = DefaultRetryOptions()
	}

	client := &Client{
		httpClient:               httpClient,
		baseURL:                  opts.BaseURL,
		apiKey:                   opts.APIKey,
		cache:                    NewCache(opts.CacheTTL),
		logger:                   opts.Logger,
		retry:                    retry,
		cacheInvalidationChannel: opts.CacheInvalidationChannel,
	}

	// Start cache invalidation listener if channel is provided
	if opts.CacheInvalidationChannel != nil {
		ctx, cancel := context.WithCancel(context.Background())
		client.stopCacheListener = cancel
		client.cacheListenerWg.Add(1)
		go client.listenForCacheInvalidation(ctx)
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
	if users := c.cache.Get(); users != nil {
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

	// Inject trace context into headers
	c.httpClient.InjectTraceContext(ctx, req)

	// Add API key header if configured
	c.addAuthHeaders(req)

	// Make request with retry
	resp, err := c.doRequestWithRetry(ctx, req)
	if err != nil {
		c.logger.Errorf("Failed to fetch users from Warden API: %v", err)
		return nil, err
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close() //nolint:errcheck // Ignoring error in defer is safe
		}
	}()

	// Check status code
	if err := c.checkResponseStatus(resp); err != nil {
		return nil, err
	}

	// Parse response
	var users []AllowListUser
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil, NewError(ErrCodeInvalidResponse, "failed to decode response", err)
	}

	// Update cache
	c.cache.Set(users)

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

	// Inject trace context into headers
	c.httpClient.InjectTraceContext(ctx, req)

	// Add API key header if configured
	c.addAuthHeaders(req)

	// Make request with retry
	resp, err := c.doRequestWithRetry(ctx, req)
	if err != nil {
		c.logger.Errorf("Failed to fetch paginated users from Warden API: %v", err)
		return nil, err
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close() //nolint:errcheck // Ignoring error in defer is safe
		}
	}()

	// Check status code
	if err := c.checkResponseStatus(resp); err != nil {
		return nil, err
	}

	// Parse response
	var paginatedResp PaginatedResponse
	if err := json.NewDecoder(resp.Body).Decode(&paginatedResp); err != nil {
		return nil, NewError(ErrCodeInvalidResponse, "failed to decode paginated response", err)
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
				c.logger.Debugf("Failed to get user from Warden API: %v (phone=%s, mail=%s)", err, sanitizePhone(phone), sanitizeEmail(mail))
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
		c.logger.Debugf("Failed to get user from Warden API: %v (phone=%s, mail=%s)", err, sanitizePhone(phone), sanitizeEmail(mail))
		return false
	}

	// Check if user status is active
	if !user.IsActive() {
		c.logger.Warnf("User status is not active: phone=%s, mail=%s, status=%s", sanitizePhone(phone), sanitizeEmail(mail), user.Status)
		return false
	}

	c.logger.Debugf("User found and active: phone=%s, mail=%s, user_id=%s", sanitizePhone(phone), sanitizeEmail(mail), user.UserID)
	return true
}

// GetUserByIdentifier fetches a single user by phone, mail, or user_id.
// Only one identifier should be provided.
func (c *Client) GetUserByIdentifier(ctx context.Context, phone, mail, userID string) (*AllowListUser, error) {
	// Validate that exactly one identifier is provided
	identifierCount := 0
	if phone != "" {
		identifierCount++
	}
	if mail != "" {
		identifierCount++
	}
	if userID != "" {
		identifierCount++
	}

	if identifierCount == 0 {
		return nil, NewError(ErrCodeInvalidConfig, "at least one identifier (phone, mail, or user_id) must be provided", nil)
	}
	if identifierCount > 1 {
		return nil, NewError(ErrCodeInvalidConfig, "only one identifier (phone, mail, or user_id) should be provided", nil)
	}

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

	// Sanitize URL in logs to mask sensitive query parameters
	sanitizedURL := sanitizeURLString(reqURL)
	c.logger.Debugf("Fetching user from Warden API: %s", sanitizedURL)

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, http.NoBody)
	if err != nil {
		return nil, NewError(ErrCodeRequestFailed, "failed to create request", err)
	}

	// Inject trace context into headers
	c.httpClient.InjectTraceContext(ctx, req)

	// Add API key header if configured
	c.addAuthHeaders(req)

	// Make request with retry
	resp, err := c.doRequestWithRetry(ctx, req)
	if err != nil {
		c.logger.Errorf("Failed to fetch user from Warden API: %v", err)
		return nil, err
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close() //nolint:errcheck // Ignoring error in defer is safe
		}
	}()

	// Check status code
	if err := c.checkResponseStatus(resp); err != nil {
		return nil, err
	}

	// Parse response
	var user AllowListUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, NewError(ErrCodeInvalidResponse, "failed to decode response", err)
	}

	c.logger.Debugf("Fetched user from Warden API: user_id=%s, phone=%s, mail=%s", user.UserID, sanitizePhone(user.Phone), sanitizeEmail(user.Mail))

	return &user, nil
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

// isRetryableError checks if an error should trigger a retry.
func (c *Client) isRetryableError(err error, statusCode int) bool {
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

// calculateRetryDelay calculates the delay for the next retry attempt using exponential backoff.
func (c *Client) calculateRetryDelay(attempt int) time.Duration {
	if c.retry == nil {
		return 0
	}

	delay := time.Duration(float64(c.retry.RetryDelay) * float64(attempt) * c.retry.BackoffMultiplier)
	if delay > c.retry.MaxRetryDelay {
		delay = c.retry.MaxRetryDelay
	}

	return delay
}

// doRequestWithRetry performs an HTTP request with retry logic.
func (c *Client) doRequestWithRetry(ctx context.Context, req *http.Request) (*http.Response, error) {
	var lastErr error
	var lastResp *http.Response

	maxAttempts := c.retry.MaxRetries + 1
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			// Calculate delay before retry
			delay := c.calculateRetryDelay(attempt - 1)
			c.logger.Debugf("Retrying request (attempt %d/%d) after %v: %s %s",
				attempt+1, maxAttempts, delay, req.Method, req.URL.String())

			// Wait before retry
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		// Make the request
		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			lastResp = nil
			if !c.isRetryableError(err, 0) {
				return nil, NewError(ErrCodeRequestFailed, "failed to execute request", err)
			}
			if attempt < c.retry.MaxRetries {
				continue
			}
			return nil, NewError(ErrCodeRequestFailed, "failed to execute request after retries", err)
		}

		// Check if status code is retryable
		if c.isRetryableError(nil, resp.StatusCode) && attempt < c.retry.MaxRetries {
			// Close response body before retry
			if err := resp.Body.Close(); err != nil {
				c.logger.Debugf("Failed to close response body before retry: %v", err)
			}
			lastResp = nil
			lastErr = NewError(ErrCodeServerError, fmt.Sprintf("server error: status %d", resp.StatusCode), nil)
			continue
		}

		// Success or non-retryable error - return response
		// The caller will check the status code
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
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
		// Also support Authorization header
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
		c.logger.Debug("Added API key headers to Warden request")
	}
}

// checkResponseStatus checks the HTTP response status and returns an error if not OK.
func (c *Client) checkResponseStatus(resp *http.Response) error {
	if resp.StatusCode == http.StatusOK {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Warnf("Warden API returned non-200 status: %d, failed to read body: %v", resp.StatusCode, err)
		body = []byte("")
	} else {
		c.logger.Warnf("Warden API returned non-200 status: %d, body: %s", resp.StatusCode, string(body))
	}

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return NewError(ErrCodeUnauthorized, "unauthorized: invalid API key", nil)
	case http.StatusNotFound:
		return NewError(ErrCodeNotFound, "not found", nil)
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable:
		return NewError(ErrCodeServerError, fmt.Sprintf("server error: status %d", resp.StatusCode), nil)
	default:
		return NewError(ErrCodeRequestFailed, fmt.Sprintf("warden API returned status %d: %s", resp.StatusCode, string(body)), nil)
	}
}
