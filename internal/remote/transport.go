package remote

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/soulteary/warden/internal/define"
)

var errRedirect = errors.New("remote redirects are disabled")

type addressResolver interface {
	LookupNetIP(context.Context, string, string) ([]netip.Addr, error)
}
type contextDialer interface {
	DialContext(context.Context, string, string) (net.Conn, error)
}

// NewHTTPClient restricts rule-source connections to validated addresses. It
// ignores environment proxies, which could resolve the target outside this policy.
func NewHTTPClient(timeout time.Duration, insecureTLS bool) (*http.Client, error) {
	return NewHTTPClientWithOptions(ClientOptions{Timeout: timeout, InsecureTLS: insecureTLS})
}

// ClientOptions configures the restricted rule-source client.
type ClientOptions struct {
	// Timeout bounds one request, including reading the body. Zero means no bound.
	Timeout time.Duration
	// InsecureTLS skips certificate verification (development only).
	InsecureTLS bool
	// MaxIdleConns is the size of the idle-connection pool. A value below one
	// selects define.DEFAULT_MAX_IDLE_CONNS.
	MaxIdleConns int
}

// maxIdleConnsPerHost is the idle-connection limit per host. The client talks
// to one rule source, so a small number is enough.
const maxIdleConnsPerHost = 2

// NewHTTPClientWithOptions is NewHTTPClient with every setting of the client.
// The client keeps idle connections, so build it once and reuse it.
func NewHTTPClientWithOptions(opts ClientOptions) (*http.Client, error) {
	timeout, insecureTLS := opts.Timeout, opts.InsecureTLS
	maxIdleConns := opts.MaxIdleConns
	if maxIdleConns < 1 {
		maxIdleConns = define.DEFAULT_MAX_IDLE_CONNS
	}
	origins, err := privateOrigins(os.Getenv("REMOTE_PRIVATE_ORIGINS"))
	if err != nil {
		return nil, err
	}
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	transport := &http.Transport{
		DialContext:           restrictedDial(net.DefaultResolver, dialer),
		TLSClientConfig:       &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: insecureTLS}, // #nosec G402 -- explicit development configuration, production validation rejects it
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		IdleConnTimeout:       30 * time.Second,
		MaxIdleConns:          maxIdleConns,
		MaxIdleConnsPerHost:   min(maxIdleConnsPerHost, maxIdleConns),
	}
	return &http.Client{Timeout: timeout, Transport: &originTransport{inner: transport, private: origins}, CheckRedirect: rejectRedirect}, nil
}

func rejectRedirect(_ *http.Request, _ []*http.Request) error { return errRedirect }

type originTransport struct {
	inner   *http.Transport
	private map[string]bool
}

func (t *originTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	origin, err := remoteOrigin(req.URL)
	if err != nil {
		return nil, err
	}
	// The scheme is part of the private-origin exception, not merely host:port.
	ctx := context.WithValue(req.Context(), privateOriginKey{}, t.private[origin])
	resp, err := t.inner.RoundTrip(req.Clone(ctx))
	if err != nil {
		stopOnAddressPolicy(ctx, err)
	}
	return resp, err
}
func (t *originTransport) CloseIdleConnections() { t.inner.CloseIdleConnections() }

type privateOriginKey struct{}

func remoteOrigin(u *url.URL) (string, error) {
	if u == nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || strings.Contains(u.Hostname(), "%") {
		return "", errors.New("remote URL must use HTTP(S) without credentials or scoped addresses")
	}
	port := u.Port()
	if port == "" {
		if u.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return "", errors.New("invalid remote URL port")
	}
	return u.Scheme + "://" + net.JoinHostPort(strings.ToLower(strings.TrimSuffix(u.Hostname(), ".")), port), nil
}
func privateOrigins(raw string) (map[string]bool, error) {
	result := map[string]bool{}
	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		u, err := url.Parse(entry)
		if err != nil {
			return nil, errors.New("invalid REMOTE_PRIVATE_ORIGINS")
		}
		origin, err := remoteOrigin(u)
		if err != nil || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.ForceQuery || u.RawPath != "" || u.Fragment != "" || strings.Contains(u.Host, "*") {
			return nil, errors.New("REMOTE_PRIVATE_ORIGINS entries must be exact HTTP(S) origins")
		}
		result[origin] = true
	}
	return result, nil
}

// ValidatePrivateOrigins validates the explicit exception list before startup.
func ValidatePrivateOrigins() error {
	_, err := privateOrigins(os.Getenv("REMOTE_PRIVATE_ORIGINS"))
	return err
}

// PrivateOriginAllowed reports an exact, explicitly configured private source.
// The exception never allows loopback, link-local or metadata destinations.
func PrivateOriginAllowed(raw string) bool {
	origins, err := privateOrigins(os.Getenv("REMOTE_PRIVATE_ORIGINS"))
	if err != nil {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	origin, err := remoteOrigin(u)
	return err == nil && origins[origin]
}

func restrictedDial(resolver addressResolver, dialer contextDialer) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		ips, err := resolver.LookupNetIP(ctx, "ip", host)
		if err != nil {
			return nil, err
		}
		if len(ips) == 0 {
			return nil, errors.New("remote host resolved to no addresses")
		}
		allowPrivate, ok := ctx.Value(privateOriginKey{}).(bool)
		if !ok {
			allowPrivate = false
		}
		for _, ip := range ips {
			if !allowedAddress(ip, allowPrivate) {
				return nil, ErrAddressPolicy
			}
		}
		var last error
		for _, ip := range ips {
			conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip.Unmap().String(), port))
			if err == nil {
				return conn, nil
			}
			last = err
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
		}
		return nil, fmt.Errorf("remote connection failed: %w", last)
	}
}

// ErrAddressPolicy reports that the connection policy refused an address the
// remote host resolved to. Another attempt meets the same policy, so the error
// is not worth retrying until the address or REMOTE_PRIVATE_ORIGINS changes.
var ErrAddressPolicy = errors.New("remote host resolves to a prohibited address")

type addressPolicyStopKey struct{}

// WithAddressPolicyStop returns a context that the restricted client cancels,
// with ErrAddressPolicy as its cause, when the connection policy refuses the
// request. A caller that retries failed requests uses it to end the retries at
// once, and reads the refusal back with AddressPolicyStopped. The returned
// function releases the context.
func WithAddressPolicyStop(ctx context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancelCause(ctx)
	return context.WithValue(ctx, addressPolicyStopKey{}, cancel), func() { cancel(nil) }
}

// AddressPolicyStopped reports whether the connection policy stopped a context
// made by WithAddressPolicyStop.
func AddressPolicyStopped(ctx context.Context) bool {
	return errors.Is(context.Cause(ctx), ErrAddressPolicy)
}

func stopOnAddressPolicy(ctx context.Context, err error) {
	if !errors.Is(err, ErrAddressPolicy) {
		return
	}
	if stop, ok := ctx.Value(addressPolicyStopKey{}).(context.CancelCauseFunc); ok {
		stop(ErrAddressPolicy)
	}
}

var prohibitedPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"), netip.MustParsePrefix("100.64.0.0/10"), netip.MustParsePrefix("192.0.0.0/24"), netip.MustParsePrefix("192.0.2.0/24"), netip.MustParsePrefix("192.88.99.0/24"), netip.MustParsePrefix("198.18.0.0/15"), netip.MustParsePrefix("198.51.100.0/24"), netip.MustParsePrefix("203.0.113.0/24"), netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("::/96"), netip.MustParsePrefix("fec0::/10"), netip.MustParsePrefix("64:ff9b::/96"), netip.MustParsePrefix("64:ff9b:1::/48"), netip.MustParsePrefix("100::/64"), netip.MustParsePrefix("2001::/23"), netip.MustParsePrefix("2001:db8::/32"), netip.MustParsePrefix("2002::/16"),
}

func allowedAddress(ip netip.Addr, allowPrivate bool) bool {
	if !ip.IsValid() || ip.Zone() != "" {
		return false
	}
	ip = ip.Unmap()
	if !ip.IsGlobalUnicast() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
		return false
	}
	// Azure's platform virtual IP is global-unicast but is not a public rule source.
	if ip == netip.MustParseAddr("168.63.129.16") || ip == netip.MustParseAddr("fd00:ec2::254") {
		return false
	}
	if ip.IsPrivate() {
		return allowPrivate
	}
	for _, prefix := range prohibitedPrefixes {
		if prefix.Contains(ip) {
			return false
		}
	}
	return true
}

// AddressAllowed applies the same policy to startup validation and connections.
func AddressAllowed(ip net.IP, allowPrivate bool) bool {
	addr, ok := netip.AddrFromSlice(ip)
	return ok && allowedAddress(addr, allowPrivate)
}

// ValidateSourceURL checks source syntax and the selected transport policy.
// Encryption envelopes do not authenticate their publisher; production delivery
// requires HTTPS in addition to decryption.
func ValidateSourceURL(raw string, requireHTTPS bool) error {
	u, err := url.Parse(raw)
	if err != nil {
		return errors.New("invalid remote source URL")
	}
	if _, err = remoteOrigin(u); err != nil {
		return err
	}
	if requireHTTPS && u.Scheme != "https" {
		return errors.New("production remote configuration requires HTTPS")
	}
	return nil
}
