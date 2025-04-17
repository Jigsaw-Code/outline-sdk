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

func TestStrategyResult_Serialization_WinnerDNS(t *testing.T) {
	cases := []struct {
		name string
		conf *dnsEntryConfig
		yaml string
	}{
		{
			name: "System",
			conf: &dnsEntryConfig{System: &struct{}{}},
			yaml: `winner:
  dns:
    system: {}`,
		},
		{
			name: "UDP",
			conf: &dnsEntryConfig{UDP: &udpEntryConfig{Address: "8.8.8.8:53"}},
			yaml: `winner:
  dns:
    udp:
      address: 8.8.8.8:53`,
		},
		{
			name: "TCP",
			conf: &dnsEntryConfig{TCP: &tcpEntryConfig{Address: "8.8.4.4:53"}},
			yaml: `winner:
  dns:
    tcp:
      address: 8.8.4.4:53`,
		},
		{
			name: "TLS",
			conf: &dnsEntryConfig{TLS: &tlsEntryConfig{Name: "dot.example.com", Address: "5.6.7.8:853"}},
			yaml: `winner:
  dns:
    tls:
      name: dot.example.com
      address: 5.6.7.8:853`,
		},
		{
			name: "HTTPS",
			conf: &dnsEntryConfig{HTTPS: &httpsEntryConfig{Name: "doh.example.com", Address: "1.2.3.4:443"}},
			yaml: `winner:
  dns:
    https:
      name: doh.example.com
      address: 1.2.3.4:443`,
		},
		{
			name: "Nil",
			conf: nil,
			yaml: `winner: {}`,
		},
	}

	for _, tc := range cases {
		t.Run("Marshal"+tc.name, func(t *testing.T) {
			cache := newMockCache()
			result := &strategyResult{
				Winner: &winningStrategy{
					DNS: tc.conf,
				},
			}
			ok := marshalStrategyResultToCache(cache, result)
			require.True(t, ok)
			require.Equal(t, tc.yaml, strings.TrimSpace(cache.entries[StrategyResultCacheKey]))
		})
		t.Run("Unmarshal"+tc.name, func(t *testing.T) {
			cache := newMockCache()
			cache.entries[StrategyResultCacheKey] = tc.yaml
			actual, ok := unmarshalStrategyResultFromCache(cache)
			require.True(t, ok)
			require.NotNil(t, actual.Winner)
			require.Equal(t, tc.conf, actual.Winner.DNS)
			require.Empty(t, actual.Winner.TLS)
			require.Nil(t, actual.Winner.Fallback)
		})
	}
}

// --- Helpers ---

type mockCache struct {
	entries map[string]string
}

func newMockCache() *mockCache {
	return &mockCache{entries: make(map[string]string)}
}

func (c *mockCache) Get(key string) (string, bool) {
	v, ok := c.entries[key]
	return v, ok
}

func (c *mockCache) Put(key string, value string) {
	c.entries[key] = value
}
