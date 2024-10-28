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

	"github.com/Jigsaw-Code/outline-sdk/transport"
)

// ConfigToDialer enables the creation of stream and packet dialers based on a config. The config is
// extensible by registering wrappers for config subtypes.
type ConfigToDialer struct {
	BaseStreamDialer   transport.StreamDialer
	BasePacketDialer   transport.PacketDialer
	BasePacketListener transport.PacketListener

	sdBuilders map[string]NewStreamDialerFunc
	pdBuilders map[string]NewPacketDialerFunc
	plBuilders map[string]NewPacketListenerFunc
}

var (
	_ ConfigToStreamDialer   = (*ConfigToDialer)(nil)
	_ StreamDialerRegistry   = (*ConfigToDialer)(nil)
	_ ConfigToPacketDialer   = (*ConfigToDialer)(nil)
	_ PacketDialerRegistry   = (*ConfigToDialer)(nil)
	_ ConfigToPacketListener = (*ConfigToDialer)(nil)
	_ PacketListenerRegistry = (*ConfigToDialer)(nil)
)

// ConfigToStreamDialer creates a [transport.StreamDialer] from a config.
type ConfigToStreamDialer interface {
	NewStreamDialerFromConfig(ctx context.Context, config *Config) (transport.StreamDialer, error)
}

// StreamDialerRegistry registers [transport.StreamDialer] types.
type StreamDialerRegistry interface {
	RegisterStreamDialerType(subtype string, newInstance NewStreamDialerFunc) error
}

// ConfigToPacketDialer creates a [transport.PacketDialer] from a config.
type ConfigToPacketDialer interface {
	NewPacketDialerFromConfig(ctx context.Context, config *Config) (transport.PacketDialer, error)
}

// PacketDialerRegistry registers [transport.PacketDialer] types.
type PacketDialerRegistry interface {
	RegisterPacketDialerType(subtype string, newInstance NewPacketDialerFunc) error
}

// ConfigToPacketListener creates a [transport.PacketListener] from a config.
type ConfigToPacketListener interface {
	NewPacketListenerFromConfig(ctx context.Context, config *Config) (transport.PacketListener, error)
}

// PacketListenerRegistry registers [transport.PacketListener] types.
type PacketListenerRegistry interface {
	RegisterPacketListenerType(subtype string, newInstance NewPacketListenerFunc) error
}

// NewStreamDialerFunc creates a [transport.StreamDialer] based on the config.
type NewStreamDialerFunc func(ctx context.Context, config *Config) (transport.StreamDialer, error)

// NewPacketDialerFunc creates a [transport.PacketDialer] based on the config.
type NewPacketDialerFunc func(ctx context.Context, config *Config) (transport.PacketDialer, error)

// NewPacketListenerFunc creates a [net.PacketConn] based on the config.
type NewPacketListenerFunc func(ctx context.Context, config *Config) (transport.PacketListener, error)

// Transport config.
type Config struct {
	URL        url.URL
	BaseConfig *Config
}

func ParseConfig(configText string) (*Config, error) {
	config := &Config{}
	parts := strings.Split(strings.TrimSpace(configText), "|")
	if len(parts) == 1 && parts[0] == "" {
		return nil, nil
	}

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

// NewDefaultConfigToDialer creates a [ConfigToDialer] with a set of default wrappers already registered.
func NewDefaultConfigToDialer() *ConfigToDialer {
	p := new(ConfigToDialer)

	p.BaseStreamDialer = &transport.TCPDialer{}
	p.BasePacketDialer = &transport.UDPDialer{}
	p.BasePacketListener = &transport.UDPListener{}

	// Please keep the list in alphabetical order.
	registerDO53StreamDialer(p, "do53", p.NewStreamDialerFromConfig, p.NewPacketDialerFromConfig)
	registerDOHStreamDialer(p, "doh", p.NewStreamDialerFromConfig)

	registerOverrideStreamDialer(p, "override", p.NewStreamDialerFromConfig)
	registerOverridePacketDialer(p, "override", p.NewPacketDialerFromConfig)

	registerSOCKS5StreamDialer(p, "socks5", p.NewStreamDialerFromConfig)
	registerSOCKS5PacketDialer(p, "socks5", p.NewStreamDialerFromConfig, p.NewPacketDialerFromConfig)
	registerSOCKS5PacketListener(p, "socks5", p.NewStreamDialerFromConfig, p.NewPacketDialerFromConfig)

	registerSplitStreamDialer(p, "split", p.NewStreamDialerFromConfig)

	registerShadowsocksStreamDialer(p, "ss", p.NewStreamDialerFromConfig)
	registerShadowsocksPacketDialer(p, "ss", p.NewPacketDialerFromConfig)
	registerShadowsocksPacketListener(p, "ss", p.NewPacketDialerFromConfig)

	registerTLSStreamDialer(p, "tls", p.NewStreamDialerFromConfig)

	registerTLSFragStreamDialer(p, "tlsfrag", p.NewStreamDialerFromConfig)

	registerWebsocketStreamDialer(p, "ws", p.NewStreamDialerFromConfig)
	registerWebsocketPacketDialer(p, "ws", p.NewStreamDialerFromConfig)

	return p
}

// RegisterStreamDialerType will register a factory for stream dialers under the given subtype.
func (p *ConfigToDialer) RegisterStreamDialerType(subtype string, newDialer NewStreamDialerFunc) error {
	if p.sdBuilders == nil {
		p.sdBuilders = make(map[string]NewStreamDialerFunc)
	}

	if _, found := p.sdBuilders[subtype]; found {
		return fmt.Errorf("config parser %v for StreamDialer added twice", subtype)
	}
	p.sdBuilders[subtype] = newDialer
	return nil
}

// RegisterPacketDialerType will register a factory for packet dialers under the given subtype.
func (p *ConfigToDialer) RegisterPacketDialerType(subtype string, newDialer NewPacketDialerFunc) error {
	if p.pdBuilders == nil {
		p.pdBuilders = make(map[string]NewPacketDialerFunc)
	}

	if _, found := p.pdBuilders[subtype]; found {
		return fmt.Errorf("config parser %v for StreamDialer added twice", subtype)
	}
	p.pdBuilders[subtype] = newDialer
	return nil
}

// RegisterPacketListenerType will register a factory for packet listeners under the given subtype.
func (p *ConfigToDialer) RegisterPacketListenerType(subtype string, newPacketListener NewPacketListenerFunc) error {
	if p.plBuilders == nil {
		p.plBuilders = make(map[string]NewPacketListenerFunc)
	}

	if _, found := p.plBuilders[subtype]; found {
		return fmt.Errorf("config parser %v for PacketConn added twice", subtype)
	}
	p.plBuilders[subtype] = newPacketListener
	return nil
}

// NewStreamDialer creates a [transport.StreamDialer] according to the config text.
func (p *ConfigToDialer) NewStreamDialer(configText string) (transport.StreamDialer, error) {
	config, err := ParseConfig(configText)
	if err != nil {
		return nil, err
	}
	return p.NewStreamDialerFromConfig(context.Background(), config)
}

// NewStreamDialerFromConfig creates a [transport.StreamDialer] according to the config.
func (p *ConfigToDialer) NewStreamDialerFromConfig(ctx context.Context, config *Config) (transport.StreamDialer, error) {
	if config == nil {
		if p.BaseStreamDialer == nil {
			return nil, errors.New("base stream dialer not configured")
		}
		return p.BaseStreamDialer, nil
	}

	newDialer, ok := p.sdBuilders[config.URL.Scheme]
	if !ok {
		return nil, fmt.Errorf("config scheme '%v' is not supported for Stream Dialers", config.URL.Scheme)
	}
	return newDialer(ctx, config)
}

// NewPacketDialer creates a [transport.PacketDialer] according to the config text.
func (p *ConfigToDialer) NewPacketDialer(configText string) (transport.PacketDialer, error) {
	config, err := ParseConfig(configText)
	if err != nil {
		return nil, err
	}
	return p.NewPacketDialerFromConfig(context.Background(), config)
}

func (p *ConfigToDialer) NewPacketDialerFromConfig(ctx context.Context, config *Config) (transport.PacketDialer, error) {
	if config == nil {
		if p.BasePacketDialer == nil {
			return nil, errors.New("base packet dialer not configured")
		}
		return p.BasePacketDialer, nil
	}

	newDialer, ok := p.pdBuilders[config.URL.Scheme]
	if !ok {
		return nil, fmt.Errorf("config scheme '%v' is not supported for Packet Dialers", config.URL.Scheme)
	}
	return newDialer(ctx, config)
}

// NewPacketListner creates a [transport.PacketListener] according to the config text.
func (p *ConfigToDialer) NewPacketListener(configText string) (transport.PacketListener, error) {
	config, err := ParseConfig(configText)
	if err != nil {
		return nil, err
	}
	return p.NewPacketListenerFromConfig(context.Background(), config)
}

// NewPacketListenerFromconfig creates a [transport.PacketListener] according to config.
func (p *ConfigToDialer) NewPacketListenerFromConfig(ctx context.Context, config *Config) (transport.PacketListener, error) {
	if config == nil {
		if p.BasePacketListener == nil {
			return nil, errors.New("base packet listener not configured")
		}
		return p.BasePacketListener, nil
	}
	newPacketListener, ok := p.plBuilders[config.URL.Scheme]
	if !ok {
		return nil, fmt.Errorf("config scheme '%v' is not supported for Stream Dialers", config.URL.Scheme)
	}
	return newPacketListener(ctx, config)
}

func SanitizeConfig(configStr string) (string, error) {
	config, err := ParseConfig(configStr)
	if err != nil {
		return "", err
	}

	// Do nothing if the config is empty
	if config == nil {
		return "", nil
	}

	// Iterate through each part
	textParts := make([]string, 0, 1)
	for config != nil {
		scheme := strings.ToLower(config.URL.Scheme)
		var part string
		switch scheme {
		case "ss":
			part, err = sanitizeShadowsocksURL(config.URL)
			if err != nil {
				return "", err
			}
		case "socks5":
			part, err = sanitizeSocks5URL(&config.URL)
			if err != nil {
				return "", err
			}
		case "override", "split", "tls", "tlsfrag":
			// No sanitization needed
			part = config.URL.String()
		default:
			part = scheme + "://UNKNOWN"
		}
		textParts = append(textParts, part)
	}
	// Join the parts back into a string
	return strings.Join(textParts, "|"), nil
}

func sanitizeSocks5URL(u *url.URL) (string, error) {
	const redactedPlaceholder = "REDACTED"
	if u.User != nil {
		u.User = url.User(redactedPlaceholder)
		return u.String(), nil
	}
	return u.String(), nil
}
