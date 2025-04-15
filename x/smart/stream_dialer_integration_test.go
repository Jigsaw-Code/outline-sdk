//go:build integration

package smart

import (
	"bytes"
	"context"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/x/configurl"
	"github.com/stretchr/testify/require"
)

func TestNewDialer_BrokenConfig(t *testing.T) {
	configBytes := []byte(`
dns:
    # We get censored DNS responses when we send queries to an IP in China.
    - udp: { address: china.cn }
    # We get censored DNS responses when we send queries to a resolver in Iran.
    - udp: { address: ns1.tic.ir }
    - tcp: { address: ns1.tic.ir }
    # We get censored DNS responses when we send queries to an IP in Turkmenistan.
    - udp: { address: tmcell.tm }
    # We get censored DNS responses when we send queries to a resolver in Russia.
    - udp: { address: dns1.transtelecom.net. }
    # Testing captive portal.
    - tls:
        name: captive-portal.badssl.com
        address: captive-portal.badssl.com:443
    # Testing forged TLS certificate.
    - https: { name: mitm-software.badssl.com }
tls:
    - ""
    - split:1
    - split:2
    - split:5
    - tlsfrag:1
fallback:
    # Nonexistent Outline Server
    - ss://Y2hhY2hhMjAtaWV0Zi1wb2x5MTMwNTprSzdEdHQ0MkJLOE9hRjBKYjdpWGFK@1.2.3.4:9999/?outline=1
    # Nonexistant Psiphon Config JSON
    - psiphon: {
        "PropagationChannelId":"ID1",
        "SponsorId":"ID2",
        "DisableLocalSocksProxy" : true,
        "DisableLocalHTTPProxy" : true,
        "EstablishTunnelTimeoutSeconds": 1,
        }
    # Nonexistant local socks5 proxy
    - socks5://192.168.1.10:1080
`)

	testDomains := []string{"www.nonexistant-canary-domain.sdfghshdfvbsdr.com"}
	transportType := ""

	logBuffer := new(bytes.Buffer)
	logger := log.New(logBuffer, "", log.LstdFlags)

	providers := configurl.NewDefaultProviders()
	packetDialer, err := providers.NewPacketDialer(context.Background(), transportType)
	if err != nil {
		t.Fatalf("Could not create packet dialer: %v", err)
	}
	streamDialer, err := providers.NewStreamDialer(context.Background(), transportType)
	if err != nil {
		t.Fatalf("Could not create stream dialer: %v", err)
	}

	finder := StrategyFinder{
		LogWriter:    logger.Writer(),
		TestTimeout:  5 * time.Second,
		StreamDialer: streamDialer,
		PacketDialer: packetDialer,
	}

	_, err := finder.NewDialer(context.Background(), testDomains, configBytes)

	require.Error(t, err)
	require.Contains(t, err.Error(), "could not find a working fallback: all tests failed")

	// Check the content of the log writer. Different systems have different network error messages. So we only check the broad strokes
	expectedLogs := []string{
		"rcode is not success: RCodeNameError ❌",
		"request for A query failed: dial DNS resolver failed: x509:",
		`request for A query failed: receive DNS message failed: failed to get HTTP response: Post "https://mitm-software.badssl.com:443/dns-query": tls: failed to verify certificate: x509: “mitm-software.badssl.com” certificate is not trusted ❌`,
		"🏃 running test: 'ss://Y2hhY2hhMjAtaWV0Zi1wb2x5MTMwNTprSzdEdHQ0MkJLOE9hRjBKYjdpWGFK@1.2.3.4:9999/?outline=1'",
		// TODO add psiphon attempt logging when it exists
		"🏃 running test: 'socks5://192.168.1.10:1080' (domain: www.nonexistant-canary-domain.sdfghshdfvbsdr.com.)",
	}
	logContent := logBuffer.String()
	for _, expectedLog := range expectedLogs {
		require.True(t, strings.Contains(logContent, expectedLog), "Expected log '%s' not found in: %s", expectedLog, logContent)
	}
}
