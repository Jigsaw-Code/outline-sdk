package smart

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseConfig_InvalidConfig(t *testing.T) {
	config := `
dns:
  - randomkey: {}
`
	configBytes := []byte(config)
	finder := &StrategyFinder{}
	_, err := finder.parseConfig(configBytes)
	require.Error(t, err)
}

func TestParseConfig_ValidConfig(t *testing.T) {
	config := `
dns:
  - system: {}
  - udp: { address: ns1.tic.ir }
  - tcp: { address: ns1.tic.ir }
  - udp: { address: tmcell.tm }
  - udp: { address: dns1.transtelecom.net. }
  - tls:
      name: captive-portal.badssl.com
      address: captive-portal.badssl.com:443
  - https: { name: mitm-software.badssl.com }

tls:
  - ""
  - split:1
  - split:2
  - split:5
  - tlsfrag:1

fallback:
  - ss://Y2hhY2hhMjAtaWV0Zi1wb2x5MTMwNTprSzdEdHQ0MkJLOE9hRjBKYjdpWGFK@1.2.3.4:9999/?outline=1
  - psiphon: {
      "PropagationChannelId":"FFFFFFFFFFFFFFFF",
      "SponsorId":"FFFFFFFFFFFFFFFF",
    }
  - socks5://192.168.1.10:1080
`
	configBytes := []byte(config)
	finder := &StrategyFinder{}
	parsedConfig, err := finder.parseConfig(configBytes)
	require.NoError(t, err)

	expectedConfig := configConfig{
		DNS: []dnsEntryConfig{
			{System: &struct{}{}},
			{UDP: &udpEntryConfig{Address: "ns1.tic.ir"}},
			{TCP: &tcpEntryConfig{Address: "ns1.tic.ir"}},
			{UDP: &udpEntryConfig{Address: "tmcell.tm"}},
			{UDP: &udpEntryConfig{Address: "dns1.transtelecom.net."}},
			{TLS: &tlsEntryConfig{Name: "captive-portal.badssl.com", Address: "captive-portal.badssl.com:443"}},
			{HTTPS: &httpsEntryConfig{Name: "mitm-software.badssl.com"}},
		},
		TLS: []string{"", "split:1", "split:2", "split:5", "tlsfrag:1"},
		Fallback: []fallbackEntryConfig{
			"ss://Y2hhY2hhMjAtaWV0Zi1wb2x5MTMwNTprSzdEdHQ0MkJLOE9hRjBKYjdpWGFK@1.2.3.4:9999/?outline=1",
			fallbackEntryStructConfig{
				Psiphon: map[string]any {
					"PropagationChannelId": "FFFFFFFFFFFFFFFF",
					"SponsorId":            "FFFFFFFFFFFFFFFF",
				},
			},
			"socks5://192.168.1.10:1080",
		},
	}

	require.Equal(t, expectedConfig, parsedConfig)
}

func TestParseConfig_YamlPsiphonConfig(t *testing.T) {
	config := `
fallback:
  - psiphon: 
      PropagationChannelId: FFFFFFFFFFFFFFFF
      SponsorId: FFFFFFFFFFFFFFFF
`
	configBytes := []byte(config)
	finder := &StrategyFinder{}
	parsedConfig, err := finder.parseConfig(configBytes)
	require.NoError(t, err)

	expectedConfig := configConfig{
		Fallback: []fallbackEntryConfig{
			fallbackEntryStructConfig{
				Psiphon: map[string]any {
					"PropagationChannelId": "FFFFFFFFFFFFFFFF",
					"SponsorId":            "FFFFFFFFFFFFFFFF",
				},
			},
		},
	}

	require.Equal(t, expectedConfig, parsedConfig)
}

func Test_getPsiphonConfigSignature_ValidFields(t *testing.T) {
	finder := &StrategyFinder{}
	config := []byte(`{
		"PropagationChannelId": "FFFFFFFFFFFFFFFF",
		"SponsorId": "FFFFFFFFFFFFFFFF",
		"ClientPlatform": "outline",
		"ClientVersion": "1"
	}`)
	expected := "{PropagationChannelId: FFFFFFFFFFFFFFFF, SponsorId: FFFFFFFFFFFFFFFF}"
	actual := finder.getPsiphonConfigSignature(config)
	require.Equal(t, expected, actual)
}

func Test_getPsiphonConfigSignature_InvalidFields(t *testing.T) {
	// If we don't understand the psiphon config we received for any reason
	// then just output it as an opaque string

	finder := &StrategyFinder{}
	config := []byte(`{"ClientPlatform": "outline", "ClientVersion": "1"}`)
	expected := `{"ClientPlatform": "outline", "ClientVersion": "1"}`
	actual := finder.getPsiphonConfigSignature(config)
	require.Equal(t, expected, actual)
}

func Test_getPsiphonConfigSignature_InvalidJson(t *testing.T) {
	finder := &StrategyFinder{}
	config := []byte(`invalid json`)
	expected := `invalid json`
	actual := finder.getPsiphonConfigSignature(config)
	require.Equal(t, expected, actual)
}