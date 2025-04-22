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

import "github.com/goccy/go-yaml"

// StrategyResultCache is a cache of strategy results that can be used by [StrategyFinder]
// to resume a strategy efficiently.
// Implementations are expected to be called concurrently from different goroutines.
type StrategyResultCache interface {
	// Get retrieves a strategy result value associated with the given key.
	// It returns the value text encoded in UTF-8 and true if found.
	Get(key string) (value []byte, ok bool)

	// Put adds the strategy result value encoded in UTF-8 to the cache with the given key.
	// If called with nil value, it should remove the cache entry.
	Put(key string, value []byte)
}

// winningStrategyCacheKey is the key for storing the winning strategy in the
// [StrategyResultCache] each time [StrategyFinder].NewDialer is invoked.
const winningStrategyCacheKey = "winning_strategy"

type proxylessEntryConfig struct {
	DNS *dnsEntryConfig `yaml:"dns,omitempty"`
	TLS string          `yaml:"tls,omitempty"`
}

type winningStrategy struct {
	Proxyless *proxylessEntryConfig `yaml:"proxyless,omitempty"`
	Fallback  fallbackEntryConfig   `yaml:"fallback,omitempty"`
}

func marshalWinningStrategyToCache(cache StrategyResultCache, winner *winningStrategy) bool {
	if cache == nil {
		return false
	}
	if winner == nil {
		cache.Put(winningStrategyCacheKey, nil)
		return true
	}

	data, err := yaml.MarshalWithOptions(winner, yaml.Flow(true))
	if err != nil {
		return false
	}
	cache.Put(winningStrategyCacheKey, data)
	return true
}

func unmarshalWinningStrategyFromCache(cache StrategyResultCache) (*winningStrategy, bool) {
	if cache == nil {
		return nil, false
	}
	data, ok := cache.Get(winningStrategyCacheKey)
	if !ok || len(data) == 0 {
		return nil, false
	}

	result := &winningStrategy{}
	if yaml.UnmarshalWithOptions([]byte(data), result, yaml.DisallowUnknownField()) != nil {
		return nil, false
	}

	// Convert to strongly typed fallback config
	if result.Fallback != nil {
		if v, ok := result.Fallback.(map[string]any); ok {
			var fallbackEntry fallbackEntryStructConfig
			if mapToAny(v, &fallbackEntry) != nil {
				return nil, false
			}
			result.Fallback = fallbackEntry
		}
	}

	return result, true
}
