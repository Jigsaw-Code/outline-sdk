// Copyright 2025 The Outline Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package smart

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWinningStrategy_MarshalProxyless(t *testing.T) {
	cases := []struct {
		name string
		dns  *dnsEntryConfig
		tls  string
		yaml string
	}{{
		name: "DNS_System",
		dns:  &dnsEntryConfig{System: &struct{}{}},
		yaml: "{dns: [{system: {}}]}",
	}, {
		name: "DNS_UDP",
		dns:  &dnsEntryConfig{UDP: &udpEntryConfig{Address: "4.3.2.1:53"}},
		yaml: `{dns: [{udp: {address: "4.3.2.1:53"}}]}`,
	}, {
		name: "DNS_TCP",
		dns:  &dnsEntryConfig{TCP: &tcpEntryConfig{Address: "1.1.4.4:53"}},
		yaml: `{dns: [{tcp: {address: "1.1.4.4:53"}}]}`,
	}, {
		name: "DNS_TLS",
		dns:  &dnsEntryConfig{TLS: &tlsEntryConfig{Name: "dot.example.com", Address: "5.6.7.8:853"}},
		yaml: `{dns: [{tls: {name: dot.example.com, address: "5.6.7.8:853"}}]}`,
	}, {
		name: "DNS_HTTPS",
		dns:  &dnsEntryConfig{HTTPS: &httpsEntryConfig{Name: "doh.example.com", Address: "8.7.6.5:443"}},
		yaml: `{dns: [{https: {name: doh.example.com, address: "8.7.6.5:443"}}]}`,
	}, {
		name: "Empty",
		yaml: `{}`,
	}, {
		name: "TLS_Split",
		tls:  "split:5",
		yaml: `{tls: ["split:5"]}`,
	}, {
		name: "TLS_Frag",
		tls:  "tlsfrag:8",
		yaml: `{tls: ["tlsfrag:8"]}`,
	}, {
		name: "DNS_System_TLS_Frag",
		dns:  &dnsEntryConfig{System: &struct{}{}},
		tls:  "tlsfrag:8",
		yaml: `{dns: [{system: {}}], tls: ["tlsfrag:8"]}`,
	}, {
		name: "DNS_HTTPS_TLS_Pipe",
		dns:  &dnsEntryConfig{HTTPS: &httpsEntryConfig{Name: "doh.example.com", Address: "9.9.9.9:443"}},
		tls:  "split:314|tlsfrag:159",
		yaml: `{dns: [{https: {name: doh.example.com, address: "9.9.9.9:443"}}], tls: ["split:314|tlsfrag:159"]}`,
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := newProxylessWinningConfig(tc.dns, tc.tls)
			actual, err := w.toYAML()
			require.NoError(t, err)
			require.Equal(t, tc.yaml, strings.TrimSpace(string(actual)))
		})
	}
}

func TestWinningStrategy_MarshalFallback(t *testing.T) {
	cases := []struct {
		name string
		conf fallbackEntryConfig
		yaml string
	}{{
		name: "Shadowsocks",
		conf: "ss://Y2hhY2hh@1.2.3.4:9999/?outline=1",
		yaml: `{fallback: ["ss://Y2hhY2hh@1.2.3.4:9999/?outline=1"]}`,
	}, {
		name: "StructPsiphon",
		conf: fallbackEntryStructConfig{
			Psiphon: map[string]any{
				"PropagationChannelId": "42",
				"SponsorId":            "TBD",
			},
		},
		yaml: `{fallback: [{psiphon: {PropagationChannelId: "42", SponsorId: TBD}}]}`,
	}, {
		name: "Empty",
		yaml: `{}`,
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := newFallbackWinningConfig(tc.conf)
			actual, err := w.toYAML()
			require.NoError(t, err)
			require.Equal(t, tc.yaml, strings.TrimSpace(string(actual)))
		})
	}
}
