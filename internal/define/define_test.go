package define

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseTrustedProxyIPs(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"empty", "", []string{}},
		{"single", "192.168.1.1", []string{"192.168.1.1"}},
		{"comma_separated", "192.168.1.1, 10.0.0.1 , 172.16.0.1", []string{"192.168.1.1", "10.0.0.1", "172.16.0.1"}},
		{"with_spaces", "  a  ,  b  ,  c  ", []string{"a", "b", "c"}},
		{"drops_empty", "a,,b,,c", []string{"a", "b", "c"}},
		{"all_empty", ", , , ", []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseTrustedProxyIPs(tt.in)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestNormalizeEndpointLabel is the cardinality guard for the Prometheus "endpoint" label.
// The metrics middleware runs before authentication, so without normalization an
// unauthenticated caller can mint one permanent time series per request path.
func TestNormalizeEndpointLabel(t *testing.T) {
	for _, path := range KnownRoutePaths {
		assert.Equal(t, path, NormalizeEndpointLabel(path), "已注册路由应保留自身标签")
	}

	unknown := []string{
		"/foo",
		"/user/",
		"/v1/",
		"/definitely-not-a-route",
		"/metrics/../user",
		"/health/../../etc/passwd",
		"/user?phone=13800138000",
		"/USER",
		"//",
		"/" + strings.Repeat("a", 4096),
	}
	for _, path := range unknown {
		assert.Equal(t, LABEL_OTHER, NormalizeEndpointLabel(path),
			"未注册路径 %q 必须归入 %q", path, LABEL_OTHER)
	}

	// An empty path is what net/http reports for a bare origin-form request.
	assert.Equal(t, PATH_ROOT, NormalizeEndpointLabel(""))

	// The label's cardinality must be bounded by the allowlist plus the "other" bucket,
	// no matter what is requested.
	distinct := map[string]struct{}{}
	for _, path := range append(append([]string{}, KnownRoutePaths...), unknown...) {
		distinct[NormalizeEndpointLabel(path)] = struct{}{}
	}
	assert.LessOrEqual(t, len(distinct), len(KnownRoutePaths)+1,
		"endpoint 标签取值必须限制在已注册路由 + %q 之内", LABEL_OTHER)
}

// TestKnownRoutePathsAreUnique guards the allowlist itself: a duplicated entry would make
// the cardinality bound in TestNormalizeEndpointLabel silently weaker than it reads.
func TestKnownRoutePathsAreUnique(t *testing.T) {
	seen := map[string]struct{}{}
	for _, path := range KnownRoutePaths {
		assert.True(t, strings.HasPrefix(path, "/"), "路由路径必须以 / 开头: %q", path)
		_, dup := seen[path]
		assert.False(t, dup, "KnownRoutePaths 存在重复项: %q", path)
		seen[path] = struct{}{}
	}
	assert.NotEmpty(t, seen)
}

// TestNormalizeMethodLabel is the second half of the cardinality guard. net/http accepts any
// valid token as a request method and hands it to the handler verbatim, so an unnormalized
// method label is just as attacker-controlled as the path.
func TestNormalizeMethodLabel(t *testing.T) {
	for _, method := range KnownHTTPMethods {
		assert.Equal(t, method, NormalizeMethodLabel(method), "标准方法应保留自身标签")
	}

	unknown := []string{
		"EVILMETHOD",
		"PROPFIND",
		"get",     // HTTP methods are case-sensitive; "get" is not GET
		"Get",     // ditto
		"",        // empty token
		"GET\r\n", // token with control characters
		strings.Repeat("X", 8192),
	}
	for _, method := range unknown {
		assert.Equal(t, LABEL_OTHER, NormalizeMethodLabel(method),
			"未知方法 %q 必须归入 %q", method, LABEL_OTHER)
	}

	distinct := map[string]struct{}{}
	for _, method := range append(append([]string{}, KnownHTTPMethods...), unknown...) {
		distinct[NormalizeMethodLabel(method)] = struct{}{}
	}
	assert.LessOrEqual(t, len(distinct), len(KnownHTTPMethods)+1,
		"method 标签取值必须限制在标准方法 + %q 之内", LABEL_OTHER)
}
