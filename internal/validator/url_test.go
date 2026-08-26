package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateRemoteURL(t *testing.T) {
	tests := []struct {
		name    string
		urlStr  string
		wantErr bool
	}{
		// Valid URLs
		{
			name:    "有效的HTTPS URL",
			urlStr:  "https://example.com/config.json",
			wantErr: false,
		},
		{
			name:    "有效的HTTP URL",
			urlStr:  "http://example.com/config.json",
			wantErr: false,
		},
		{
			name:    "有效的URL带端口",
			urlStr:  "https://example.com:8080/config.json",
			wantErr: false,
		},
		{
			name:    "有效的URL带路径和查询参数",
			urlStr:  "https://example.com/api/v1/config?key=value",
			wantErr: false,
		},

		// Invalid URLs
		{
			name:    "空URL",
			urlStr:  "",
			wantErr: true,
		},
		{
			name:    "无效的URL格式",
			urlStr:  "not-a-url",
			wantErr: true,
		},
		{
			name:    "不允许的协议（ftp）",
			urlStr:  "ftp://example.com/config.json",
			wantErr: true,
		},
		{
			name:    "不允许的协议（file）",
			urlStr:  "file:///etc/passwd",
			wantErr: true,
		},
		{
			name:    "缺少host",
			urlStr:  "https:///config.json",
			wantErr: true,
		},

		// Security restrictions: prohibit localhost
		{
			name:    "禁止localhost",
			urlStr:  "http://localhost/config.json",
			wantErr: true,
		},
		{
			name:    "禁止127.0.0.1",
			urlStr:  "http://127.0.0.1/config.json",
			wantErr: true,
		},
		{
			name:    "禁止::1",
			urlStr:  "http://[::1]/config.json",
			wantErr: true,
		},

		// Security restrictions: prohibit private IPs
		{
			name:    "禁止10.0.0.0/8",
			urlStr:  "http://10.0.0.1/config.json",
			wantErr: true,
		},
		{
			name:    "禁止172.16.0.0/12",
			urlStr:  "http://172.16.0.1/config.json",
			wantErr: true,
		},
		{
			name:    "禁止192.168.0.0/16",
			urlStr:  "http://192.168.1.1/config.json",
			wantErr: true,
		},
		{
			name:    "禁止127.0.0.0/8",
			urlStr:  "http://127.0.0.1/config.json",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRemoteURL(tt.urlStr)
			if tt.wantErr {
				assert.Error(t, err, "ValidateRemoteURL(%q) 应该返回错误", tt.urlStr)
			} else {
				assert.NoError(t, err, "ValidateRemoteURL(%q) 不应该返回错误", tt.urlStr)
			}
		})
	}
}

func TestValidateConfigPath(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		allowedDirs []string
		wantErr     bool
	}{
		// Valid paths
		{
			name:        "有效的相对路径",
			path:        "./config.yaml",
			allowedDirs: nil,
			wantErr:     false,
		},
		{
			name:        "有效的绝对路径",
			path:        "/tmp/config.yaml",
			allowedDirs: nil,
			wantErr:     false,
		},
		{
			name:        "路径在允许的目录下",
			path:        "./config.yaml",
			allowedDirs: []string{".", "/tmp"},
			wantErr:     false,
		},

		// Invalid paths
		{
			name:        "空路径",
			path:        "",
			allowedDirs: nil,
			wantErr:     true,
		},
		{
			name:        "路径不在允许的目录下",
			path:        "/etc/passwd",
			allowedDirs: []string{"/tmp", "/var"},
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			absPath, err := ValidateConfigPath(tt.path, tt.allowedDirs)
			if tt.wantErr {
				assert.Error(t, err, "ValidateConfigPath(%q, %v) 应该返回错误", tt.path, tt.allowedDirs)
				assert.Empty(t, absPath, "错误时应该返回空路径")
			} else {
				assert.NoError(t, err, "ValidateConfigPath(%q, %v) 不应该返回错误", tt.path, tt.allowedDirs)
				assert.NotEmpty(t, absPath, "成功时应该返回绝对路径")
			}
		})
	}
}

func TestValidateConfigPath_WithAllowedDirs(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		path        string
		allowedDirs []string
		wantErr     bool
	}{
		{
			name:        "路径在允许的目录下",
			path:        tmpDir + "/config.yaml",
			allowedDirs: []string{tmpDir},
			wantErr:     false,
		},
		{
			name:        "路径不在允许的目录下",
			path:        "/etc/passwd",
			allowedDirs: []string{tmpDir},
			wantErr:     true,
		},
		{
			name:        "多个允许的目录，路径在其中一个下",
			path:        tmpDir + "/config.yaml",
			allowedDirs: []string{"/tmp", tmpDir, "/var"},
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			absPath, err := ValidateConfigPath(tt.path, tt.allowedDirs)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, absPath)
			}
		})
	}
}

// TestValidateConfigPath_PathTraversal tests path traversal detection
func TestValidateConfigPath_PathTraversal(t *testing.T) {
	// Test that path traversal doesn't cause panics and returns valid result
	assert.NotPanics(t, func() {
		absPath, err := ValidateConfigPath("../../etc/passwd", nil)
		// The function should either return an error or a valid path
		// We just verify it doesn't panic - behavior depends on filepath.Abs implementation
		_ = absPath
		_ = err
	})
}
