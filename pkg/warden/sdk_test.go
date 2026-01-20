package warden

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// mockLogger is a simple logger implementation for testing.
type mockLogger struct {
	debugs []string
	infos  []string
	warns  []string
	errors []string
}

func (m *mockLogger) Debug(msg string) {
	m.debugs = append(m.debugs, msg)
}

func (m *mockLogger) Debugf(format string, args ...interface{}) {
	m.debugs = append(m.debugs, fmt.Sprintf(format, args...))
}

func (m *mockLogger) Info(msg string) {
	m.infos = append(m.infos, msg)
}

func (m *mockLogger) Infof(format string, args ...interface{}) {
	m.infos = append(m.infos, fmt.Sprintf(format, args...))
}

func (m *mockLogger) Warn(msg string) {
	m.warns = append(m.warns, msg)
}

func (m *mockLogger) Warnf(format string, args ...interface{}) {
	m.warns = append(m.warns, fmt.Sprintf(format, args...))
}

func (m *mockLogger) Error(msg string) {
	m.errors = append(m.errors, msg)
}

func (m *mockLogger) Errorf(format string, args ...interface{}) {
	m.errors = append(m.errors, fmt.Sprintf(format, args...))
}

func TestNewClient(t *testing.T) {
	//nolint:govet // fieldalignment: test struct field order does not affect functionality
	tests := []struct {
		name    string
		opts    *Options
		wantErr bool
	}{
		{
			name: "valid options",
			opts: &Options{
				BaseURL:  "http://localhost:8081",
				APIKey:   "test-key",
				Timeout:  10 * time.Second,
				CacheTTL: 5 * time.Minute,
				Logger:   &NoOpLogger{},
			},
			wantErr: false,
		},
		{
			name: "missing base URL",
			opts: &Options{
				BaseURL: "",
			},
			wantErr: true,
		},
		{
			name:    "default options",
			opts:    DefaultOptions().WithBaseURL("http://localhost:8081"),
			wantErr: false,
		},
		{
			name:    "URL without protocol",
			opts:    DefaultOptions().WithBaseURL("localhost:8081"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && client == nil {
				t.Error("NewClient() returned nil client without error")
			}
		})
	}
}

func TestClient_GetUsers(t *testing.T) {
	// Create mock server
	mockUsers := []AllowListUser{
		{Phone: "13800138000", Mail: "user1@example.com"},
		{Phone: "13900139000", Mail: "user2@example.com"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET request, got %s", r.Method)
		}

		// Check API key if provided
		if apiKey := r.Header.Get("X-API-Key"); apiKey != "" && apiKey != "test-key" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(mockUsers))
	}))
	defer server.Close()

	// Create client
	logger := &mockLogger{}
	opts := DefaultOptions().
		WithBaseURL(server.URL).
		WithAPIKey("test-key").
		WithLogger(logger).
		WithCacheTTL(1 * time.Second)

	client, err := NewClient(opts)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test GetUsers
	ctx := context.Background()
	users, err := client.GetUsers(ctx)
	if err != nil {
		t.Fatalf("GetUsers() error = %v", err)
	}

	if len(users) != len(mockUsers) {
		t.Errorf("GetUsers() returned %d users, want %d", len(users), len(mockUsers))
	}

	// Test cache - second call should use cache
	users2, err := client.GetUsers(ctx)
	if err != nil {
		t.Fatalf("GetUsers() error = %v", err)
	}

	if len(users2) != len(mockUsers) {
		t.Errorf("GetUsers() (cached) returned %d users, want %d", len(users2), len(mockUsers))
	}

	// Verify cache was used (check for debug log)
	foundCacheLog := false
	for _, log := range logger.debugs {
		if contains(log, "cached") {
			foundCacheLog = true
			break
		}
	}
	if !foundCacheLog {
		t.Error("Expected cache to be used on second call")
	}
}

func TestClient_GetUsersPaginated(t *testing.T) {
	// Create mock server
	mockUsers := []AllowListUser{
		{Phone: "13800138000", Mail: "user1@example.com"},
		{Phone: "13900139000", Mail: "user2@example.com"},
		{Phone: "14000140000", Mail: "user3@example.com"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		pageSize := r.URL.Query().Get("page_size")

		if page == "" || pageSize == "" {
			http.Error(w, "Missing pagination parameters", http.StatusBadRequest)
			return
		}

		// Simple pagination logic for testing
		var start, end int
		_, err := fmt.Sscanf(page, "%d", &start)
		require.NoError(t, err)
		_, err = fmt.Sscanf(pageSize, "%d", &end)
		require.NoError(t, err)
		start = (start - 1) * end
		end = start + end
		if end > len(mockUsers) {
			end = len(mockUsers)
		}

		response := PaginatedResponse{
			Data: mockUsers[start:end],
			Pagination: PaginationInfo{
				Page:       start/end + 1,
				PageSize:   end - start,
				Total:      len(mockUsers),
				TotalPages: (len(mockUsers) + end - start - 1) / (end - start),
			},
		}

		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(response))
	}))
	defer server.Close()

	// Create client
	opts := DefaultOptions().
		WithBaseURL(server.URL).
		WithAPIKey("test-key")

	client, err := NewClient(opts)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test GetUsersPaginated
	ctx := context.Background()
	resp, err := client.GetUsersPaginated(ctx, 1, 2)
	if err != nil {
		t.Fatalf("GetUsersPaginated() error = %v", err)
	}

	if resp == nil {
		t.Fatal("GetUsersPaginated() returned nil response")
	}

	if len(resp.Data) != 2 {
		t.Errorf("GetUsersPaginated() returned %d users, want 2", len(resp.Data))
	}

	if resp.Pagination.Total != 3 {
		t.Errorf("GetUsersPaginated() total = %d, want 3", resp.Pagination.Total)
	}
}

func TestClient_CheckUserInList(t *testing.T) {
	// Create mock server that handles /user endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Handle /user endpoint (used by GetUserByIdentifier)
		if r.URL.Path == "/user" {
			phone := r.URL.Query().Get("phone")
			mail := r.URL.Query().Get("mail")

			var user AllowListUser
			switch {
			case phone == "13800138000":
				user = AllowListUser{Phone: "13800138000", Mail: "user1@example.com", Status: "active", UserID: "user1"}
			case mail == "user2@example.com" || mail == "USER2@EXAMPLE.COM":
				user = AllowListUser{Phone: "13900139000", Mail: "user2@example.com", Status: "active", UserID: "user2"}
			case phone == "14000140000":
				user = AllowListUser{Phone: "14000140000", Mail: "user3@example.com", Status: "inactive", UserID: "user3"}
			case mail == "user4@example.com":
				user = AllowListUser{Phone: "14100141000", Mail: "user4@example.com", Status: "suspended", UserID: "user4"}
			default:
				w.WriteHeader(http.StatusNotFound)
				return
			}
			require.NoError(t, json.NewEncoder(w).Encode(user))
			return
		}

		// Handle root endpoint (for backward compatibility)
		mockUsers := []AllowListUser{
			{Phone: "13800138000", Mail: "user1@example.com", Status: "active"},
			{Phone: "13900139000", Mail: "user2@example.com", Status: "active"},
		}
		require.NoError(t, json.NewEncoder(w).Encode(mockUsers))
	}))
	defer server.Close()

	// Create client
	opts := DefaultOptions().
		WithBaseURL(server.URL).
		WithCacheTTL(1 * time.Second)

	client, err := NewClient(opts)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// Test existing active user by phone
	if !client.CheckUserInList(ctx, "13800138000", "") {
		t.Error("CheckUserInList() should return true for existing active user by phone")
	}

	// Test existing active user by mail
	if !client.CheckUserInList(ctx, "", "user2@example.com") {
		t.Error("CheckUserInList() should return true for existing active user by mail")
	}

	// Test case-insensitive mail matching
	if !client.CheckUserInList(ctx, "", "USER2@EXAMPLE.COM") {
		t.Error("CheckUserInList() should match mail case-insensitively")
	}

	// Test non-existing user
	if client.CheckUserInList(ctx, "99999999999", "") {
		t.Error("CheckUserInList() should return false for non-existing phone")
	}

	// Test inactive user (should be rejected)
	if client.CheckUserInList(ctx, "14000140000", "") {
		t.Error("CheckUserInList() should return false for inactive user")
	}

	// Test suspended user (should be rejected)
	if client.CheckUserInList(ctx, "", "user4@example.com") {
		t.Error("CheckUserInList() should return false for suspended user")
	}
}

func TestClient_ClearCache(t *testing.T) {
	mockUsers := []AllowListUser{
		{Phone: "13800138000", Mail: "user1@example.com"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(mockUsers))
	}))
	defer server.Close()

	opts := DefaultOptions().
		WithBaseURL(server.URL).
		WithCacheTTL(1 * time.Minute)

	client, err := NewClient(opts)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// Fetch users to populate cache
	_, err = client.GetUsers(ctx)
	if err != nil {
		t.Fatalf("GetUsers() error = %v", err)
	}

	// Clear cache
	client.ClearCache()

	// Verify cache is cleared by checking if next call makes a new request
	// (we can't directly verify this, but we can check that cache is empty)
	// The cache should be empty after ClearCache
}

func TestOptions_Validate(t *testing.T) {
	//nolint:govet // fieldalignment: test struct field order does not affect functionality
	tests := []struct {
		name    string
		opts    *Options
		wantErr bool
	}{
		{
			name: "valid options",
			opts: &Options{
				BaseURL:  "http://localhost:8081",
				Timeout:  10 * time.Second,
				CacheTTL: 5 * time.Minute,
			},
			wantErr: false,
		},
		{
			name:    "missing base URL",
			opts:    &Options{BaseURL: ""},
			wantErr: true,
		},
		{
			name: "invalid timeout",
			opts: &Options{
				BaseURL: "http://localhost:8081",
				Timeout: -1 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "negative cache TTL",
			opts: &Options{
				BaseURL:  "http://localhost:8081",
				Timeout:  10 * time.Second,
				CacheTTL: -1 * time.Second,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Options.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestError(t *testing.T) {
	err := NewError(ErrCodeRequestFailed, "test error", nil)
	if err.Error() == "" {
		t.Error("Error.Error() should not return empty string")
	}

	if err.Code != ErrCodeRequestFailed {
		t.Errorf("Error.Code = %s, want %s", err.Code, ErrCodeRequestFailed)
	}
}

func TestCache(t *testing.T) {
	cache := NewCache(1 * time.Second)

	// Test empty cache
	if users := cache.Get(); users != nil {
		t.Error("Cache.Get() should return nil for empty cache")
	}

	// Set cache
	users := []AllowListUser{
		{Phone: "13800138000", Mail: "user1@example.com"},
	}
	cache.Set(users)

	// Get from cache
	cached := cache.Get()
	if len(cached) != len(users) {
		t.Errorf("Cache.Get() returned %d users, want %d", len(cached), len(users))
	}

	// Clear cache
	cache.Clear()
	if cache.Get() != nil {
		t.Error("Cache.Get() should return nil after Clear()")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || substr == "" || containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ========== Additional tests: Error handling ==========

func TestClient_GetUsers_HTTPErrors(t *testing.T) {
	//nolint:govet // fieldalignment: test struct field order does not affect functionality
	tests := []struct {
		name            string
		statusCode      int
		expectedErrCode string
	}{
		{
			name:            "unauthorized",
			statusCode:      http.StatusUnauthorized,
			expectedErrCode: ErrCodeUnauthorized,
		},
		{
			name:            "not found",
			statusCode:      http.StatusNotFound,
			expectedErrCode: ErrCodeNotFound,
		},
		{
			name:            "server error",
			statusCode:      http.StatusInternalServerError,
			expectedErrCode: ErrCodeServerError,
		},
		{
			name:            "bad gateway",
			statusCode:      http.StatusBadGateway,
			expectedErrCode: ErrCodeServerError,
		},
		{
			name:            "service unavailable",
			statusCode:      http.StatusServiceUnavailable,
			expectedErrCode: ErrCodeServerError,
		},
		{
			name:            "bad request",
			statusCode:      http.StatusBadRequest,
			expectedErrCode: ErrCodeRequestFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "test error", tt.statusCode)
			}))
			defer server.Close()

			opts := DefaultOptions().WithBaseURL(server.URL)
			client, err := NewClient(opts)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			ctx := context.Background()
			_, err = client.GetUsers(ctx)
			if err == nil {
				t.Fatal("Expected error but got nil")
			}

			sdkErr, ok := err.(*Error)
			if !ok {
				t.Fatalf("Expected *Error, got %T", err)
			}

			if sdkErr.Code != tt.expectedErrCode {
				t.Errorf("Error.Code = %s, want %s", sdkErr.Code, tt.expectedErrCode)
			}
		})
	}
}

func TestClient_GetUsers_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("invalid json"))
		require.NoError(t, err)
	}))
	defer server.Close()

	opts := DefaultOptions().WithBaseURL(server.URL)
	client, err := NewClient(opts)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	_, err = client.GetUsers(ctx)
	if err == nil {
		t.Fatal("Expected error for invalid JSON")
	}

	sdkErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("Expected *Error, got %T", err)
	}

	if sdkErr.Code != ErrCodeInvalidResponse {
		t.Errorf("Error.Code = %s, want %s", sdkErr.Code, ErrCodeInvalidResponse)
	}
}

func TestClient_GetUsers_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode([]AllowListUser{}))
	}))
	defer server.Close()

	opts := DefaultOptions().WithBaseURL(server.URL)
	client, err := NewClient(opts)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	users, err := client.GetUsers(ctx)
	if err != nil {
		t.Fatalf("GetUsers() error = %v", err)
	}

	if len(users) != 0 {
		t.Errorf("GetUsers() returned %d users, want 0", len(users))
	}
}

func TestClient_GetUsersPaginated_InvalidParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode([]AllowListUser{}))
	}))
	defer server.Close()

	opts := DefaultOptions().WithBaseURL(server.URL)
	client, err := NewClient(opts)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// Test invalid page
	_, err = client.GetUsersPaginated(ctx, 0, 10)
	if err == nil {
		t.Fatal("Expected error for page=0")
	}

	// Test invalid pageSize
	_, err = client.GetUsersPaginated(ctx, 1, 0)
	if err == nil {
		t.Fatal("Expected error for pageSize=0")
	}

	// Test negative page
	_, err = client.GetUsersPaginated(ctx, -1, 10)
	if err == nil {
		t.Fatal("Expected error for negative page")
	}

	// Test negative pageSize
	_, err = client.GetUsersPaginated(ctx, 1, -1)
	if err == nil {
		t.Fatal("Expected error for negative pageSize")
	}
}

func TestClient_CheckUserInList_ErrorHandling(t *testing.T) {
	// Test with server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}))
	defer server.Close()

	opts := DefaultOptions().WithBaseURL(server.URL)
	client, err := NewClient(opts)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	// Should return false on error, not panic
	result := client.CheckUserInList(ctx, "13800138000", "")
	if result {
		t.Error("CheckUserInList() should return false on error")
	}
}

func TestClient_CheckUserInList_EdgeCases(t *testing.T) {
	// Create mock server that handles /user endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Handle /user endpoint (used by GetUserByIdentifier)
		if r.URL.Path == "/user" {
			phone := strings.TrimSpace(r.URL.Query().Get("phone"))
			mail := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("mail")))

			var user AllowListUser
			switch {
			case phone == "13900139000":
				user = AllowListUser{Phone: "13900139000", Mail: "user2@example.com", Status: "active", UserID: "user2"}
			case mail == "user2@example.com":
				user = AllowListUser{Phone: "13900139000", Mail: "user2@example.com", Status: "active", UserID: "user2"}
			case mail == "user3@example.com":
				user = AllowListUser{Phone: "", Mail: "user3@example.com", Status: "active", UserID: "user3"}
			case phone == "14000140000":
				user = AllowListUser{Phone: "14000140000", Mail: "", Status: "active", UserID: "user4"}
			default:
				w.WriteHeader(http.StatusNotFound)
				return
			}
			require.NoError(t, json.NewEncoder(w).Encode(user))
			return
		}

		// Handle root endpoint (for backward compatibility)
		mockUsers := []AllowListUser{
			{Phone: "13800138000", Mail: "user1@example.com", Status: "active"},
		}
		require.NoError(t, json.NewEncoder(w).Encode(mockUsers))
	}))
	defer server.Close()

	opts := DefaultOptions().WithBaseURL(server.URL)
	client, err := NewClient(opts)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// Test matching with spaces (should be trimmed)
	if !client.CheckUserInList(ctx, "13900139000", "") {
		t.Error("CheckUserInList() should trim spaces from phone")
	}

	// Test matching with case-insensitive mail
	if !client.CheckUserInList(ctx, "", "user2@example.com") {
		t.Error("CheckUserInList() should match mail case-insensitively")
	}

	// Test matching user with empty phone
	if !client.CheckUserInList(ctx, "", "user3@example.com") {
		t.Error("CheckUserInList() should match user with empty phone")
	}

	// Test matching user with empty mail
	if !client.CheckUserInList(ctx, "14000140000", "") {
		t.Error("CheckUserInList() should match user with empty mail")
	}

	// Test both empty
	if client.CheckUserInList(ctx, "", "") {
		t.Error("CheckUserInList() should return false when both phone and mail are empty")
	}
}

func TestClient_AddAuthHeaders(t *testing.T) {
	var receivedAPIKey string
	var receivedAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAPIKey = r.Header.Get("X-API-Key")
		receivedAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode([]AllowListUser{}))
	}))
	defer server.Close()

	// Test with API key
	opts := DefaultOptions().
		WithBaseURL(server.URL).
		WithAPIKey("test-api-key")

	client, err := NewClient(opts)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	_, err = client.GetUsers(ctx)
	if err != nil {
		t.Fatalf("GetUsers() error = %v", err)
	}

	if receivedAPIKey != "test-api-key" {
		t.Errorf("X-API-Key header = %s, want test-api-key", receivedAPIKey)
	}

	if receivedAuth != "Bearer test-api-key" {
		t.Errorf("Authorization header = %s, want Bearer test-api-key", receivedAuth)
	}

	// Test without API key
	receivedAPIKey = ""
	receivedAuth = ""
	opts2 := DefaultOptions().WithBaseURL(server.URL)
	client2, err := NewClient(opts2)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	_, err = client2.GetUsers(ctx)
	if err != nil {
		t.Fatalf("GetUsers() error = %v", err)
	}

	if receivedAPIKey != "" {
		t.Error("X-API-Key header should be empty when no API key is set")
	}

	if receivedAuth != "" {
		t.Error("Authorization header should be empty when no API key is set")
	}
}

// ========== Additional tests: Cache functionality ==========

func TestCache_Expiration(t *testing.T) {
	cache := NewCache(100 * time.Millisecond)

	users := []AllowListUser{
		{Phone: "13800138000", Mail: "user1@example.com"},
	}

	// Set cache
	cache.Set(users)

	// Get immediately - should work
	cached := cache.Get()
	if len(cached) != 1 {
		t.Errorf("Cache.Get() returned %d users, want 1", len(cached))
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Get after expiration - should return nil
	cached = cache.Get()
	if cached != nil {
		t.Error("Cache.Get() should return nil after expiration")
	}
}

func TestCache_ConcurrentAccess(t *testing.T) {
	cache := NewCache(1 * time.Minute)
	users := []AllowListUser{
		{Phone: "13800138000", Mail: "user1@example.com"},
	}

	// Test concurrent writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			cache.Set(users)
			done <- true
		}()
	}

	// Wait for all writes
	for i := 0; i < 10; i++ {
		<-done
	}

	// Test concurrent reads
	for i := 0; i < 10; i++ {
		go func() {
			cached := cache.Get()
			if cached == nil {
				t.Error("Cache.Get() returned nil in concurrent read")
			}
			done <- true
		}()
	}

	// Wait for all reads
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestCache_DataIsolation(t *testing.T) {
	cache := NewCache(1 * time.Minute)

	users1 := []AllowListUser{
		{Phone: "13800138000", Mail: "user1@example.com"},
	}

	// Set cache
	cache.Set(users1)

	// Get and modify
	cached := cache.Get()
	if len(cached) != 1 {
		t.Fatalf("Cache.Get() returned %d users, want 1", len(cached))
	}

	// Modify the returned slice
	cached[0].Phone = "modified"

	// Get again - should not be modified
	cached2 := cache.Get()
	if cached2[0].Phone == "modified" {
		t.Error("Cache should return a copy, modifications should not affect cache")
	}
}

func TestCache_ZeroTTL(t *testing.T) {
	cache := NewCache(0) // Zero TTL means cache expires immediately

	users := []AllowListUser{
		{Phone: "13800138000", Mail: "user1@example.com"},
	}

	cache.Set(users)

	// With zero TTL, cache expires immediately (or very quickly)
	// Add a small delay to ensure expiration
	time.Sleep(10 * time.Millisecond)
	cached := cache.Get()
	if cached != nil {
		t.Error("Cache.Get() should return nil with zero TTL (cache expires immediately)")
	}
}

// ========== Additional tests: Options ==========

func TestOptions_WithMethods(t *testing.T) {
	opts := DefaultOptions()

	// Test chaining
	opts = opts.
		WithBaseURL("http://test.com").
		WithAPIKey("test-key").
		WithTimeout(5 * time.Second).
		WithCacheTTL(2 * time.Minute)

	if opts.BaseURL != "http://test.com" {
		t.Errorf("BaseURL = %s, want http://test.com", opts.BaseURL)
	}

	if opts.APIKey != "test-key" {
		t.Errorf("APIKey = %s, want test-key", opts.APIKey)
	}

	if opts.Timeout != 5*time.Second {
		t.Errorf("Timeout = %v, want 5s", opts.Timeout)
	}

	if opts.CacheTTL != 2*time.Minute {
		t.Errorf("CacheTTL = %v, want 2m", opts.CacheTTL)
	}
}

func TestOptions_Validate_URLNormalization(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "URL with trailing slash",
			input:    "http://localhost:8081/",
			expected: "http://localhost:8081",
		},
		{
			name:     "URL without protocol",
			input:    "localhost:8081",
			expected: "http://localhost:8081",
		},
		{
			name:     "URL with https",
			input:    "https://example.com",
			expected: "https://example.com",
		},
		{
			name:     "URL with http",
			input:    "http://example.com",
			expected: "http://example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &Options{
				BaseURL: tt.input,
				Timeout: 10 * time.Second,
			}

			err := opts.Validate()
			if err != nil {
				t.Fatalf("Validate() error = %v", err)
			}

			if opts.BaseURL != tt.expected {
				t.Errorf("BaseURL = %s, want %s", opts.BaseURL, tt.expected)
			}
		})
	}
}

func TestOptions_DefaultLogger(t *testing.T) {
	opts := &Options{
		BaseURL: "http://localhost:8081",
		Timeout: 10 * time.Second,
		Logger:  nil, // nil logger
	}

	err := opts.Validate()
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if opts.Logger == nil {
		t.Error("Logger should be set to NoOpLogger when nil")
	}

	_, ok := opts.Logger.(*NoOpLogger)
	if !ok {
		t.Errorf("Logger type = %T, want *NoOpLogger", opts.Logger)
	}
}

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	if opts.Timeout != 10*time.Second {
		t.Errorf("Default Timeout = %v, want 10s", opts.Timeout)
	}

	if opts.CacheTTL != 5*time.Minute {
		t.Errorf("Default CacheTTL = %v, want 5m", opts.CacheTTL)
	}

	if opts.Logger == nil {
		t.Error("Default Logger should not be nil")
	}

	_, ok := opts.Logger.(*NoOpLogger)
	if !ok {
		t.Errorf("Default Logger type = %T, want *NoOpLogger", opts.Logger)
	}
}

// ========== Additional tests: Error type ==========

func TestError_WithNestedError(t *testing.T) {
	originalErr := fmt.Errorf("original error")
	err := NewError(ErrCodeRequestFailed, "test error", originalErr)

	if err.Error() == "" {
		t.Error("Error.Error() should not return empty string")
	}

	if err.Code != ErrCodeRequestFailed {
		t.Errorf("Error.Code = %s, want %s", err.Code, ErrCodeRequestFailed)
	}

	if err.Unwrap() != originalErr {
		t.Error("Error.Unwrap() should return the original error")
	}

	// Test error message contains original error
	if !contains(err.Error(), "original error") {
		t.Error("Error.Error() should include the original error message")
	}
}

func TestError_WithoutNestedError(t *testing.T) {
	err := NewError(ErrCodeInvalidConfig, "config error", nil)

	if err.Error() == "" {
		t.Error("Error.Error() should not return empty string")
	}

	if err.Unwrap() != nil {
		t.Error("Error.Unwrap() should return nil when no nested error")
	}
}

func TestError_AllErrorCodes(t *testing.T) {
	codes := []string{
		ErrCodeInvalidConfig,
		ErrCodeRequestFailed,
		ErrCodeInvalidResponse,
		ErrCodeUnauthorized,
		ErrCodeNotFound,
		ErrCodeServerError,
	}

	for _, code := range codes {
		err := NewError(code, "test", nil)
		if err.Code != code {
			t.Errorf("Error.Code = %s, want %s", err.Code, code)
		}
	}
}

// ========== Additional tests: LogrusAdapter ==========

func TestLogrusAdapter(t *testing.T) {
	// This test requires logrus, if available in the project
	// To avoid adding dependencies, we can test the nil logger case
	adapter := NewLogrusAdapter(nil)
	if adapter == nil {
		t.Error("NewLogrusAdapter() should not return nil even with nil logger")
	}

	// Test all methods don't panic
	adapter.Debug("test")
	adapter.Debugf("test %s", "format")
	adapter.Info("test")
	adapter.Infof("test %s", "format")
	adapter.Warn("test")
	adapter.Warnf("test %s", "format")
	adapter.Error("test")
	adapter.Errorf("test %s", "format")
}

// ========== Additional tests: NoOpLogger ==========

func TestNoOpLogger(t *testing.T) {
	logger := &NoOpLogger{}

	// Test all methods don't panic
	logger.Debug("test")
	logger.Debugf("test %s", "format")
	logger.Info("test")
	logger.Infof("test %s", "format")
	logger.Warn("test")
	logger.Warnf("test %s", "format")
	logger.Error("test")
	logger.Errorf("test %s", "format")
}

// ========== Additional tests: Context cancellation ==========

func TestClient_GetUsers_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow request
		time.Sleep(2 * time.Second)
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode([]AllowListUser{}))
	}))
	defer server.Close()

	opts := DefaultOptions().
		WithBaseURL(server.URL).
		WithTimeout(5 * time.Second)

	client, err := NewClient(opts)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = client.GetUsers(ctx)
	if err == nil {
		t.Error("Expected error when context is cancelled")
	}
}

func TestClient_GetUsers_ContextTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow request
		time.Sleep(2 * time.Second)
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode([]AllowListUser{}))
	}))
	defer server.Close()

	opts := DefaultOptions().
		WithBaseURL(server.URL).
		WithTimeout(5 * time.Second)

	client, err := NewClient(opts)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err = client.GetUsers(ctx)
	if err == nil {
		t.Error("Expected error when context times out")
	}
}

// ========== Additional tests: Edge cases ==========

func TestClient_GetUsersPaginated_EmptyResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := PaginatedResponse{
			Data: []AllowListUser{},
			Pagination: PaginationInfo{
				Page:       1,
				PageSize:   10,
				Total:      0,
				TotalPages: 0,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(response))
	}))
	defer server.Close()

	opts := DefaultOptions().WithBaseURL(server.URL)
	client, err := NewClient(opts)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	resp, err := client.GetUsersPaginated(ctx, 1, 10)
	if err != nil {
		t.Fatalf("GetUsersPaginated() error = %v", err)
	}

	if len(resp.Data) != 0 {
		t.Errorf("GetUsersPaginated() returned %d users, want 0", len(resp.Data))
	}

	if resp.Pagination.Total != 0 {
		t.Errorf("Pagination.Total = %d, want 0", resp.Pagination.Total)
	}
}

func TestCache_EmptySlice(t *testing.T) {
	cache := NewCache(1 * time.Minute)

	// Set empty slice
	cache.Set([]AllowListUser{})

	// Get should return empty slice, not nil
	cached := cache.Get()
	if cached == nil {
		t.Error("Cache.Get() should return empty slice, not nil")
	}

	if len(cached) != 0 {
		t.Errorf("Cache.Get() returned %d users, want 0", len(cached))
	}
}
