// Copyright 2026 The Outline Authors
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
	"errors"
	"fmt"
	"io"
	"maps"
	"sync"
	"time"

	"golang.getoutline.org/sdk/x/configurl"
)

// StrategyFinderOptions configures a [StrategyFinder]. Setters collect options;
// [StrategyFinderOptions.New] validates them. Finish all configuration, including
// registry and fallback registrations, before calling New. Do not copy options
// after their first New call. Concurrent New calls are supported after setup.
type StrategyFinderOptions struct {
	// testTimeout is the positive timeout for each connectivity test.
	testTimeout time.Duration
	// logWriter receives strategy selection logs; nil disables logging.
	logWriter io.Writer
	// cache stores successful strategies for later searches; nil disables caching.
	cache StrategyResultCache
	// configURLRegistry is shared by the finders created from these options.
	// If nil, each New call creates and prepares a fresh default registry.
	configURLRegistry configurl.MutableRegistry
	// registryMu serializes registry preparation during concurrent New calls.
	// It does not protect setters, which must finish before New is called.
	registryMu sync.Mutex
	// registryPrepared records whether these options have installed the context-aware
	// default stream parser in configURLRegistry. It prevents repeated New calls from
	// wrapping the previous wrapper again. New protects it with registryMu;
	// SetConfigURLRegistry resets it during setup.
	registryPrepared bool
	// fallbackParsers maps YAML fallback type names to parsers. New copies this map
	// into each finder so later changes to the options do not affect existing finders.
	fallbackParsers map[string]FallbackParser
}

// NewStrategyFinderOptions creates options with a five-second test timeout.
func NewStrategyFinderOptions() *StrategyFinderOptions {
	return &StrategyFinderOptions{testTimeout: 5 * time.Second}
}

// SetConfigURLRegistry shares registry with the finders created by New. A nil
// registry selects a fresh default registry for each New call. Complete
// registrations before New. On its first successful call, New wraps the default
// stream parser to read a TLS-specific base from the construction context.
// The original parser continues to serve DNS searches and fallbacks.
func (o *StrategyFinderOptions) SetConfigURLRegistry(registry configurl.MutableRegistry) *StrategyFinderOptions {
	o.configURLRegistry = registry
	o.registryPrepared = false
	return o
}

// SetTestTimeout sets the positive timeout for each connectivity test.
func (o *StrategyFinderOptions) SetTestTimeout(timeout time.Duration) *StrategyFinderOptions {
	o.testTimeout = timeout
	return o
}

// SetLogWriter sets the optional destination for strategy selection logs.
func (o *StrategyFinderOptions) SetLogWriter(writer io.Writer) *StrategyFinderOptions {
	o.logWriter = writer
	return o
}

// SetCache sets the optional cache of successful strategies.
func (o *StrategyFinderOptions) SetCache(cache StrategyResultCache) *StrategyFinderOptions {
	o.cache = cache
	return o
}

// RegisterFallbackParser registers or replaces a parser for a YAML fallback type.
// A parser can capture the registry passed to SetConfigURLRegistry to construct
// nested config URLs using the same component registrations and base instances.
func (o *StrategyFinderOptions) RegisterFallbackParser(name string, parser FallbackParser) *StrategyFinderOptions {
	if o.fallbackParsers == nil {
		o.fallbackParsers = make(map[string]FallbackParser)
	}
	o.fallbackParsers[name] = parser
	return o
}

// New validates the options and creates a finder without performing network I/O.
// It snapshots the options and fallback registrations, sharing the registry, cache,
// and log writer by reference. The deprecated dialer fields are left nil.
func (o *StrategyFinderOptions) New() (*StrategyFinder, error) {
	if o.testTimeout <= 0 {
		return nil, errors.New("test timeout must be positive")
	}
	for name, parser := range o.fallbackParsers {
		if parser == nil {
			return nil, fmt.Errorf("fallback parser %q must not be nil", name)
		}
	}
	registry := o.configURLRegistry
	// Prepare the stream base to accept a search-specific DNS dialer from the
	// construction context. This lets TLS strategies share that dialer and its
	// cache while DNS discovery and fallbacks keep using the registered base.
	if registry == nil {
		registry = configurl.NewDefaultRegistry()
		registerContextStreamBase(registry)
	} else {
		o.registryMu.Lock()
		if !o.registryPrepared {
			registerContextStreamBase(registry)
			o.registryPrepared = true
		}
		o.registryMu.Unlock()
	}
	return &StrategyFinder{
		TestTimeout:       o.testTimeout,
		LogWriter:         o.logWriter,
		Cache:             o.cache,
		configURLRegistry: registry,
		fallbackParsers:   maps.Clone(o.fallbackParsers),
	}, nil
}
