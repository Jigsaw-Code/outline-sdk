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
	"bytes"

	"github.com/goccy/go-yaml"
)

// StrategyResultCacheKey is the key associated with a strategy result in [StrategyResultCache].
const StrategyResultCacheKey = "smart-strategy-result"

// StrategyResultCache is a cache of strategy results that can be used by [StrategyFinder]
// to resume a strategy efficiently.
// Implementations are expected to be called concurrently from different goroutines.
type StrategyResultCache interface {
	// Get retrieves a strategy result value associated with the given key.
	// It returns the value and true if found.
	Get(key string) (value string, ok bool)

	// Put adds the strategy result value to the cache with the given key.
	// If called with an empty value string, it should remove the cache entry.
	Put(key string, value string)
}

// winningStrategy contains the detailed configurations of a winning strategy.
type winningStrategy struct {
	DNS      *dnsEntryConfig     `yaml:"dns,omitempty"`
	TLS      string              `yaml:"tls,omitempty"`
	Fallback fallbackEntryConfig `yaml:"fallback,omitempty"`
}

// strategyResult holds the details of a successful strategy found by [StrategyFinder].
// It is designed to be serializable to YAML for caching.
type strategyResult struct {
	// Winner contains the details of the winning strategy.
	Winner *winningStrategy `yaml:"winner"`
}

// marshalStrategyResultToCache stores the given strategyResult in the cache.
// If called with nil result, it removes the entry from the cache.
func marshalStrategyResultToCache(cache StrategyResultCache, result *strategyResult) bool {
	if cache == nil {
		return false
	}
	if result == nil {
		cache.Put(StrategyResultCacheKey, "")
		return true
	}
	data, err := yaml.Marshal(result)
	if err != nil {
		return false
	}
	cache.Put(StrategyResultCacheKey, string(data))
	return true
}

// unmarshalStrategyResultFromCache retrieves a cached strategy result from the cache.
// It returns nil and false if no valid strategy results are found in the cache.
func unmarshalStrategyResultFromCache(cache StrategyResultCache) (*strategyResult, bool) {
	if cache == nil {
		return nil, false
	}
	data, ok := cache.Get(StrategyResultCacheKey)
	if !ok || data == "" {
		return nil, false
	}
	result := &strategyResult{}
	decoder := yaml.NewDecoder(bytes.NewReader([]byte(data)), yaml.DisallowUnknownField())
	if err := decoder.Decode(result); err != nil {
		return nil, false
	}
	return result, true
}
