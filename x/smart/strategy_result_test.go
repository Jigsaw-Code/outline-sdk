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

func TestWinningStrategy_MarshalDNS(t *testing.T) {
	cases := []struct {
		name string
		conf *dnsEntryConfig
		yaml string
	}{
		{
			name: "System",
			conf: &dnsEntryConfig{System: &struct{}{}},
			yaml: "{proxyless: {dns: {system: {}}}}",
		},
		{
			name: "UDP",
			conf: &dnsEntryConfig{UDP: &udpEntryConfig{Address: "8.8.8.8:53"}},
			yaml: `{proxyless: {dns: {udp: {address: "8.8.8.8:53"}}}}`,
		},
		{
			name: "TCP",
			conf: &dnsEntryConfig{TCP: &tcpEntryConfig{Address: "8.8.4.4:53"}},
			yaml: `{proxyless: {dns: {tcp: {address: "8.8.4.4:53"}}}}`,
		},
		{
			name: "TLS",
			conf: &dnsEntryConfig{TLS: &tlsEntryConfig{Name: "dot.example.com", Address: "5.6.7.8:853"}},
			yaml: `{proxyless: {dns: {tls: {name: dot.example.com, address: "5.6.7.8:853"}}}}`,
		},
		{
			name: "HTTPS",
			conf: &dnsEntryConfig{HTTPS: &httpsEntryConfig{Name: "doh.example.com", Address: "1.2.3.4:443"}},
			yaml: `{proxyless: {dns: {https: {name: doh.example.com, address: "1.2.3.4:443"}}}}`,
		},
		{
			name: "Nil",
			conf: nil,
			yaml: `{proxyless: {}}`,
		},
	}

	for _, tc := range cases {
		t.Run("Marshal_"+tc.name, func(t *testing.T) {
			cache := newMockCache()
			result := &winningStrategy{
				Proxyless: &proxylessEntryConfig{
					DNS: tc.conf,
				},
			}
			ok := marshalWinningStrategyToCache(cache, result)
			require.True(t, ok)
			cache.expectExistIgnoreSpace(t, winningStrategyCacheKey, tc.yaml)
		})
		t.Run("Unmarshal_"+tc.name, func(t *testing.T) {
			cache := newMockCache()
			cache.entries[winningStrategyCacheKey] = []byte(tc.yaml)
			actual, ok := unmarshalWinningStrategyFromCache(cache)
			require.True(t, ok)
			require.NotNil(t, actual.Proxyless)
			require.Equal(t, tc.conf, actual.Proxyless.DNS)
			require.Empty(t, actual.Proxyless.TLS)
			require.Nil(t, actual.Fallback)
		})
	}
}

func TestWinningStrategy_MarshalTLS(t *testing.T) {
	cases := []struct {
		name    string
		dnsConf *dnsEntryConfig
		tlsConf string
		yaml    string
	}{
		{
			name:    "Split&NilDNS",
			dnsConf: nil,
			tlsConf: "split:5",
			yaml:    `{proxyless: {tls: "split:5"}}`,
		},
		{
			name:    "TLSFrag&SysDNS",
			dnsConf: &dnsEntryConfig{System: &struct{}{}},
			tlsConf: "tlsfrag:8",
			yaml:    `{proxyless: {dns: {system: {}}, tls: "tlsfrag:8"}}`,
		},
		{
			name:    "Pipe&DoH",
			dnsConf: &dnsEntryConfig{HTTPS: &httpsEntryConfig{Name: "mitm-software.badssl.com"}},
			tlsConf: "split:314|tlsfrag:159",
			yaml:    `{proxyless: {dns: {https: {name: mitm-software.badssl.com}}, tls: "split:314|tlsfrag:159"}}`,
		},
	}

	for _, tc := range cases {
		t.Run("Marshal_"+tc.name, func(t *testing.T) {
			cache := newMockCache()
			result := &winningStrategy{
				Proxyless: &proxylessEntryConfig{
					DNS: tc.dnsConf,
					TLS: tc.tlsConf,
				},
			}
			ok := marshalWinningStrategyToCache(cache, result)
			require.True(t, ok)
			cache.expectExistIgnoreSpace(t, winningStrategyCacheKey, tc.yaml)
		})
		t.Run("Unmarshal_"+tc.name, func(t *testing.T) {
			cache := newMockCache()
			cache.entries[winningStrategyCacheKey] = []byte(tc.yaml)
			actual, ok := unmarshalWinningStrategyFromCache(cache)
			require.True(t, ok)
			require.NotNil(t, actual.Proxyless)
			require.Equal(t, tc.dnsConf, actual.Proxyless.DNS)
			require.Equal(t, tc.tlsConf, actual.Proxyless.TLS)
			require.Nil(t, actual.Fallback)
		})
	}
}

func TestWinningStrategy_MarshalFallback(t *testing.T) {
	cases := []struct {
		name string
		conf fallbackEntryConfig
		yaml string
	}{
		{
			name: "Shadowsocks",
			conf: "ss://Y2hhY2hh@1.2.3.4:9999/?outline=1",
			yaml: `{fallback: "ss://Y2hhY2hh@1.2.3.4:9999/?outline=1"}`,
		},
		{
			name: "StructPsiphon",
			conf: fallbackEntryStructConfig{
				Psiphon: map[string]any{
					"PropagationChannelId": "42",
					"SponsorId":            "TBD",
				},
			},
			yaml: `{fallback: {psiphon: {PropagationChannelId: "42", SponsorId: TBD}}}`,
		},
	}

	for _, tc := range cases {
		t.Run("Marshal"+tc.name, func(t *testing.T) {
			cache := newMockCache()
			result := &winningStrategy{Fallback: tc.conf}
			ok := marshalWinningStrategyToCache(cache, result)
			require.True(t, ok)
			cache.expectExistIgnoreSpace(t, winningStrategyCacheKey, tc.yaml)
		})
		t.Run("Unmarshal"+tc.name, func(t *testing.T) {
			cache := newMockCache()
			cache.entries[winningStrategyCacheKey] = []byte(tc.yaml)
			actual, ok := unmarshalWinningStrategyFromCache(cache)
			require.True(t, ok)
			require.NotNil(t, actual.Fallback)
			require.Equal(t, tc.conf, actual.Fallback)
			require.Nil(t, actual.Proxyless)
		})
	}
}

// --- Helpers ---

type mockCache struct {
	entries map[string][]byte
}

func newMockCache() *mockCache {
	return &mockCache{entries: make(map[string][]byte)}
}

func (c *mockCache) Get(key string) ([]byte, bool) {
	v, ok := c.entries[key]
	return v, ok
}

func (c *mockCache) Put(key string, value []byte) {
	c.entries[key] = value
}

func (c *mockCache) expectExistIgnoreSpace(t *testing.T, key string, val string) {
	v, ok := c.entries[key]
	require.True(t, ok)
	require.Equal(t, val, strings.TrimSpace(string(v)))
}
