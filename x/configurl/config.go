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
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Jigsaw-Code/outline-sdk/transport/tlsfrag"
)

// ConfigToDialer enables the creation of stream and packet dialers based on a config. The config is
// extensible by registering wrappers for config subtypes.
type ConfigToDialer struct {
	NewBaseStreamDialer func(ctx context.Context) (transport.StreamDialer, error)
	NewBasePacketDialer func(ctx context.Context) (transport.PacketDialer, error)
	NewBasePacketConn   func(ctx context.Context) (net.PacketConn, error)

	sdBuilders map[string]NewStreamDialerFunc
	pdBuilders map[string]NewPacketDialerFunc
	pcBuilders map[string]NewPacketConnFunc
}

type ConfigToStreamDialer interface {
	NewStreamDialer(ctx context.Context, config *Config) (transport.StreamDialer, error)
}

// NewStreamDialerFunc creates a [transport.StreamDialer] based on the config.
type NewStreamDialerFunc func(ctx context.Context, config *Config) (transport.StreamDialer, error)

// NewPacketDialerFunc creates a [transport.PacketDialer] based on the config.
type NewPacketDialerFunc func(ctx context.Context, config *Config) (transport.PacketDialer, error)

// NewPacketConnFunc creates a [net.PacketConn] based on the wrapConfig. The innerSD and innerPD functions can provide a base Stream and Packet Dialers if needed.
type NewPacketConnFunc func(ctx context.Context, config *Config) (net.PacketConn, error)

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
	tcpDialer := &transport.TCPDialer{}
	p.NewBaseStreamDialer = func(ctx context.Context) (transport.StreamDialer, error) {
		return tcpDialer, nil
	}
	udpDialer := &transport.UDPDialer{}
	p.NewBasePacketDialer = func(ctx context.Context) (transport.PacketDialer, error) {
		return udpDialer, nil
	}

	p.NewBasePacketConn = func(ctx context.Context) (net.PacketConn, error) {
		return net.ListenUDP("", &net.UDPAddr{})
	}

	// Please keep the list in alphabetical order.
	p.RegisterStreamDialerType("do53", newDO53StreamDialerFactory(p.NewStreamDialer, p.NewPacketDialer))
	p.RegisterStreamDialerType("doh", newDOHStreamDialerFactory(p.NewStreamDialer))

	p.RegisterStreamDialerType("override", newOverrideStreamDialerFactory(p.NewStreamDialer))
	p.RegisterPacketDialerType("override", newOverridePacketDialerFactory(p.NewPacketDialer))

	p.RegisterStreamDialerType("socks5", newSOCKS5StreamDialerFactory(p.NewStreamDialer))
	p.RegisterPacketDialerType("socks5", newSOCKS5PacketDialerFactory(p.NewStreamDialer, p.NewPacketDialer))
	p.RegisterPacketConnType("socks5", newSOCKS5PacketConnFactory(p.NewStreamDialer, p.NewPacketDialer))

	p.RegisterStreamDialerType("split", newSplitStreamDialerFactory(p.NewStreamDialer))

	p.RegisterStreamDialerType("ss", wrapStreamDialerWithShadowsocks)
	p.RegisterPacketDialerType("ss", wrapPacketDialerWithShadowsocks)

	p.RegisterStreamDialerType("tls", wrapStreamDialerWithTLS)

	p.RegisterStreamDialerType("tlsfrag", func(innerSD func() (transport.StreamDialer, error), innerPD func() (transport.PacketDialer, error), wrapConfig *url.URL) (transport.StreamDialer, error) {
		sd, err := innerSD()
		if err != nil {
			return nil, err
		}
		lenStr := wrapConfig.Opaque
		fixedLen, err := strconv.Atoi(lenStr)
		if err != nil {
			return nil, fmt.Errorf("invalid tlsfrag option: %v. It should be in tlsfrag:<number> format", lenStr)
		}
		return tlsfrag.NewFixedLenStreamDialer(sd, fixedLen)
	})

	p.RegisterStreamDialerType("ws", wrapStreamDialerWithWebSocket)
	p.RegisterPacketDialerType("ws", wrapPacketDialerWithWebSocket)

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

// RegisterPacketConnType will register a factory for packet conns under the given subtype.
func (p *ConfigToDialer) RegisterPacketConnType(subtype string, newPacketConn NewPacketConnFunc) error {
	if p.pcBuilders == nil {
		p.pcBuilders = make(map[string]NewPacketConnFunc)
	}

	if _, found := p.pcBuilders[subtype]; found {
		return fmt.Errorf("config parser %v for PacketConn added twice", subtype)
	}
	p.pcBuilders[subtype] = newPacketConn
	return nil
}

func (p *ConfigToDialer) newBaseStreamDialer(ctx context.Context) (transport.StreamDialer, error) {
	if p.NewBaseStreamDialer == nil {
		return nil, errors.New("base stream dialer not configured")
	}
	return p.NewBaseStreamDialer(ctx)
}

// NewStreamDialer creates a [transport.StreamDialer] according to the config.
func (p *ConfigToDialer) NewStreamDialer(ctx context.Context, config *Config) (transport.StreamDialer, error) {
	if config == nil {
		return p.newBaseStreamDialer(ctx)
	}

	newDialer, ok := p.sdBuilders[config.URL.Scheme]
	if !ok {
		return nil, fmt.Errorf("config scheme '%v' is not supported for Stream Dialers", config.URL.Scheme)
	}
	return newDialer(ctx, config)
}

func (p *ConfigToDialer) newBasePacketDialer(ctx context.Context) (transport.PacketDialer, error) {
	if p.NewBasePacketDialer == nil {
		return nil, errors.New("base packet dialer not configured")
	}
	return p.NewBasePacketDialer(ctx)
}

func (p *ConfigToDialer) NewPacketDialer(ctx context.Context, config *Config) (transport.PacketDialer, error) {
	if config == nil {
		return p.newBasePacketDialer(ctx)
	}

	newDialer, ok := p.pdBuilders[config.URL.Scheme]
	if !ok {
		return nil, fmt.Errorf("config scheme '%v' is not supported for Packet Dialers", config.URL.Scheme)
	}
	return newDialer(ctx, config)
}

func (p *ConfigToDialer) newBasePacketConn(ctx context.Context) (net.PacketConn, error) {
	if p.NewBasePacketConn == nil {
		return nil, errors.New("base packet conn not configured")
	}
	return p.NewBasePacketConn(ctx)
}

// NewPacketConn creates a [net.PacketConn] according to transportConfig, using dialer as the
// base [Dialer]. The given dialer must not be nil.
func (p *ConfigToDialer) NewPacketConn(ctx context.Context, config *Config) (net.PacketConn, error) {
	if config == nil {
		return p.newBasePacketConn(ctx)
	}
	newPacketConn, ok := p.pcBuilders[config.URL.Scheme]
	if !ok {
		return nil, fmt.Errorf("config scheme '%v' is not supported for Stream Dialers", config.URL.Scheme)
	}
	return newPacketConn(ctx, config)
}

// NewpacketListener creates a new [transport.PacketListener] according to the given config,
// the config must contain only one "ss://" segment.
// TODO: make NewPacketListener configurable.
func NewPacketListener(transportConfig string) (transport.PacketListener, error) {
	parts, err := parseConfig(transportConfig)
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, errors.New("config is required")
	}
	if len(parts) > 1 {
		return nil, errors.New("multi-part config is not supported")
	}

	url := parts[0]
	// Please keep scheme list sorted.
	switch strings.ToLower(url.Scheme) {
	case "ss":
		// TODO: support nested dialer, the last part must be "ss://"
		return newShadowsocksPacketListenerFromURL(url)
	default:
		return nil, fmt.Errorf("config scheme '%v' is not supported", url.Scheme)
	}
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
			part, err = sanitizeShadowsocksURL(&config.URL)
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
