// Copyright 2023 The Outline Authors
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

package configurl

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

type ExtensibleProvider[ObjectType comparable] struct {
	BaseInstance ObjectType
	builders     map[string]BuildFunc[ObjectType]
}

type BuildFunc[ObjectType any] func(ctx context.Context, config *Config) (ObjectType, error)

func (p *ExtensibleProvider[ObjectType]) buildersMap() map[string]BuildFunc[ObjectType] {
	if p.builders == nil {
		p.builders = make(map[string]BuildFunc[ObjectType])
	}
	return p.builders
}

// RegisterType will register a factory for the given subtype.
func (p *ExtensibleProvider[ObjectType]) RegisterType(subtype string, newDialer BuildFunc[ObjectType]) error {
	builders := p.buildersMap()
	if _, found := builders[subtype]; found {
		return fmt.Errorf("type %v registered twice", subtype)
	}
	builders[subtype] = newDialer
	return nil
}

// NewInstance creates a new instance according to the config.
func (p *ExtensibleProvider[ObjectType]) NewInstance(ctx context.Context, config *Config) (ObjectType, error) {
	var zero ObjectType
	if config == nil {
		if p.BaseInstance == zero {
			return zero, errors.New("base instance is not configured")
		}
		return p.BaseInstance, nil
	}

	newDialer, ok := p.buildersMap()[config.URL.Scheme]
	if !ok {
		return zero, fmt.Errorf("config type '%v' is not registered", config.URL.Scheme)
	}
	return newDialer(ctx, config)
}

var (
	_ Provider[any]     = (*ExtensibleProvider[any])(nil)
	_ TypeRegistry[any] = (*ExtensibleProvider[any])(nil)
)

// Provider creates an instance from a config.
type Provider[ObjectType any] interface {
	NewInstance(ctx context.Context, config *Config) (ObjectType, error)
}

// TypeRegistry registers config types.
type TypeRegistry[ObjectType any] interface {
	RegisterType(subtype string, newInstance BuildFunc[ObjectType]) error
}

// Transport config.
type Config struct {
	URL        url.URL
	BaseConfig *Config
}

func ParseConfig(configText string) (*Config, error) {
	parts := strings.Split(strings.TrimSpace(configText), "|")
	if len(parts) == 1 && parts[0] == "" {
		return nil, nil
	}

	var config *Config = nil
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, errors.New("empty config part")
		}
		// Make it "<scheme>:" if it's only "<scheme>" to parse as a URL.
		if !strings.Contains(part, ":") {
			part += ":"
		}
		url, err := url.Parse(part)
		if err != nil {
			return nil, fmt.Errorf("part is not a valid URL: %w", err)
		}
		config = &Config{URL: *url, BaseConfig: config}
	}
	return config, nil
}
