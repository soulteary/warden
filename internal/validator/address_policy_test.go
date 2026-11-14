package validator

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

// Startup validation applies the connection policy of the rule-source client,
// which refuses more than loopback and private addresses.
func TestValidateRemoteURLAppliesTheConnectionPolicy(t *testing.T) {
	public := net.IPAddr{IP: net.ParseIP("93.184.216.34")}
	resolveTo := func(ips ...string) staticIPResolver {
		resolver := staticIPResolver{}
		for _, ip := range ips {
			resolver.addrs = append(resolver.addrs, net.IPAddr{IP: net.ParseIP(ip)})
		}
		return resolver
	}

	// Literal addresses in ranges the policy refuses: benchmarking, shared
	// address space and the NAT64 form of the metadata address.
	for _, literal := range []string{"http://198.18.0.1/data.json", "http://100.64.0.1/data.json", "http://[64:ff9b::a9fe:a9fe]/data.json"} {
		require.Error(t, validateRemoteURL(literal, staticIPResolver{}), literal)
	}

	// Every address a name resolves to is checked, not only the first.
	for _, refused := range []string{"127.0.0.1", "169.254.169.254", "10.0.0.5", "198.18.0.1"} {
		resolver := staticIPResolver{addrs: []net.IPAddr{public, {IP: net.ParseIP(refused)}}}
		require.ErrorContains(t, validateRemoteURL("https://rules.example/data.json", resolver), "resolves to a prohibited address", refused)
	}

	// A private address is accepted only for an origin listed exactly.
	t.Setenv("REMOTE_PRIVATE_ORIGINS", "https://rules.internal:8443")
	require.NoError(t, validateRemoteURL("https://rules.internal:8443/data.json", resolveTo("10.0.0.5")))
	require.Error(t, validateRemoteURL("https://rules.internal/data.json", resolveTo("10.0.0.5")), "another port is another origin")
	require.Error(t, validateRemoteURL("https://other.internal:8443/data.json", resolveTo("10.0.0.5")))
	// The exception never covers loopback or metadata addresses.
	require.Error(t, validateRemoteURL("https://rules.internal:8443/data.json", resolveTo("127.0.0.1")))
	require.Error(t, validateRemoteURL("https://rules.internal:8443/data.json", resolveTo("169.254.169.254")))

	// Listing a loopback or metadata origin does not make it acceptable,
	// neither as a literal address nor through a name.
	t.Setenv("REMOTE_PRIVATE_ORIGINS", "http://127.0.0.1:8080, http://169.254.169.254, http://10.0.0.5")
	require.Error(t, validateRemoteURL("http://127.0.0.1:8080/data.json", staticIPResolver{}))
	require.ErrorContains(t, validateRemoteURL("http://169.254.169.254/latest/meta-data", staticIPResolver{}), "remote URL has a prohibited address")
	require.NoError(t, validateRemoteURL("http://10.0.0.5/data.json", staticIPResolver{}))

	// A list that is not valid fails validation of any URL.
	t.Setenv("REMOTE_PRIVATE_ORIGINS", "https://rules.internal:8443/path")
	require.ErrorContains(t, validateRemoteURL("https://rules.example/data.json", resolveTo("93.184.216.34")), "REMOTE_PRIVATE_ORIGINS")
}
