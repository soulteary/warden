package remote

import (
	"context"
	"crypto/tls"
	"github.com/soulteary/warden/internal/define"
	"net/http"
	"time"
)

func testHTTPClient(timeout time.Duration, insecure bool) *http.Client {
	// #nosec G402 -- local fixtures exercise the explicit development TLS option
	return &http.Client{Timeout: timeout, Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure}}, CheckRedirect: rejectRedirect}
}
func fetchTestWithOptions(ctx context.Context, opts *FetchOptions) ([]byte, error) {
	opts.HTTPClient = testHTTPClient(opts.Timeout, opts.InsecureTLS)
	defer opts.HTTPClient.CloseIdleConnections()
	return FetchDecryptedWithOptions(ctx, opts)
}
func fetchTestDecrypted(ctx context.Context, rawURL, auth string, enabled bool, path, pem string, timeout time.Duration, insecure bool) ([]byte, error) {
	return fetchTestWithOptions(ctx, &FetchOptions{URL: rawURL, AuthHeader: auth, DecryptEnabled: enabled, RSAKeyPath: path, RSAKeyPEM: pem, Timeout: timeout, InsecureTLS: insecure})
}
func fetchTestUsers(ctx context.Context, rawURL, auth string, enabled bool, path, pem string, timeout time.Duration, insecure bool) ([]define.AllowListUser, error) {
	body, err := fetchTestDecrypted(ctx, rawURL, auth, enabled, path, pem, timeout, insecure)
	if err != nil {
		return nil, err
	}
	return parseUsers(body)
}
