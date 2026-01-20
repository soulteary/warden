package warden

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Client is the Warden API client.
//
//nolint:govet // fieldalignment: field order has been optimized, but not further adjusted to maintain API compatibility
type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	cache      *Cache
	logger     Logger
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

	client := &Client{
		httpClient: &http.Client{
			Timeout: opts.Timeout,
		},
		baseURL: opts.BaseURL,
		apiKey:  opts.APIKey,
		cache:   NewCache(opts.CacheTTL),
		logger:  opts.Logger,
	}

	client.logger.Debugf("Warden SDK client created: URL=%s, APIKey=%v", opts.BaseURL, opts.APIKey != "")

	return client, nil
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

	// Add API key header if configured
	c.addAuthHeaders(req)

	// Make request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Errorf("Failed to fetch users from Warden API: %v", err)
		return nil, NewError(ErrCodeRequestFailed, "failed to fetch users from Warden", err)
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

	// Add API key header if configured
	c.addAuthHeaders(req)

	// Make request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Errorf("Failed to fetch paginated users from Warden API: %v", err)
		return nil, NewError(ErrCodeRequestFailed, "failed to fetch paginated users from Warden", err)
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
func (c *Client) CheckUserInList(ctx context.Context, phone, mail string) bool {
	// Normalize input
	phone = strings.TrimSpace(phone)
	mail = strings.TrimSpace(strings.ToLower(mail))

	// Use GetUserByIdentifier for direct query (better performance than fetching all users)
	user, err := c.GetUserByIdentifier(ctx, phone, mail, "")
	if err != nil {
		// Log error but don't expose details to caller (security: don't reveal if user exists)
		c.logger.Debugf("Failed to get user from Warden API: %v (phone=%s, mail=%s)", err, phone, mail)
		return false
	}

	// Check if user status is active
	if !user.IsActive() {
		c.logger.Warnf("User status is not active: phone=%s, mail=%s, status=%s", phone, mail, user.Status)
		return false
	}

	c.logger.Debugf("User found and active: phone=%s, mail=%s, user_id=%s", phone, mail, user.UserID)
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
	reqURL := fmt.Sprintf("%s/user", c.baseURL)
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

	c.logger.Debugf("Fetching user from Warden API: %s", reqURL)

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, http.NoBody)
	if err != nil {
		return nil, NewError(ErrCodeRequestFailed, "failed to create request", err)
	}

	// Add API key header if configured
	c.addAuthHeaders(req)

	// Make request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Errorf("Failed to fetch user from Warden API: %v", err)
		return nil, NewError(ErrCodeRequestFailed, "failed to fetch user from Warden", err)
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

	c.logger.Debugf("Fetched user from Warden API: user_id=%s, phone=%s, mail=%s", user.UserID, user.Phone, user.Mail)

	return &user, nil
}

// ClearCache clears the internal cache.
func (c *Client) ClearCache() {
	c.cache.Clear()
	c.logger.Debug("Cache cleared")
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
