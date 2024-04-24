// Copyright 2023 Jigsaw Operations LLC
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

package config

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Jigsaw-Code/outline-sdk/transport/split"
	"github.com/Jigsaw-Code/outline-sdk/transport/tlsfrag"
)

// ConfigParser enables the creation of stream and packet dialers based on a config. The config is
// extensible by registering wrappers for config subtypes.
type ConfigParser struct {
	sdWrapers  map[string]WrapStreamDialerFunc
	pdWrappers map[string]WrapPacketDialerFunc
}

// NewDefaultConfigParser creates a [ConfigParser] with a set of default wrappers already registered.
func NewDefaultConfigParser() *ConfigParser {
	p := new(ConfigParser)

	// Please keep the list in alphabetical order.

	p.RegisterStreamDialerWrapper("override", wrapStreamDialerWithOverride)
	p.RegisterPacketDialerWrapper("override", wrapPacketDialerWithOverride)

	p.RegisterStreamDialerWrapper("socks5", wrapStreamDialerWithSOCKS5)
	p.RegisterPacketDialerWrapper("socks5", func(baseDialer transport.PacketDialer, wrapConfig *url.URL) (transport.PacketDialer, error) {
		return nil, errors.New("socks5 is not supported for PacketDialers")
	})

	p.RegisterStreamDialerWrapper("split", func(baseDialer transport.StreamDialer, wrapConfig *url.URL) (transport.StreamDialer, error) {
		prefixBytesStr := wrapConfig.Opaque
		prefixBytes, err := strconv.Atoi(prefixBytesStr)
		if err != nil {
			return nil, fmt.Errorf("prefixBytes is not a number: %v. Split config should be in split:<number> format", prefixBytesStr)
		}
		return split.NewStreamDialer(baseDialer, int64(prefixBytes))
	})
	p.RegisterPacketDialerWrapper("split", func(baseDialer transport.PacketDialer, wrapConfig *url.URL) (transport.PacketDialer, error) {
		return nil, errors.New("split is not supported for PacketDialers")
	})

	p.RegisterStreamDialerWrapper("ss", wrapStreamDialerWithShadowsocks)
	p.RegisterPacketDialerWrapper("ss", wrapPacketDialerWithShadowsocks)

	p.RegisterStreamDialerWrapper("tls", wrapStreamDialerWithTLS)
	p.RegisterPacketDialerWrapper("tls", func(baseDialer transport.PacketDialer, wrapConfig *url.URL) (transport.PacketDialer, error) {
		return nil, errors.New("tls is not supported for PacketDialers")
	})

	p.RegisterStreamDialerWrapper("tlsfrag", func(baseDialer transport.StreamDialer, wrapConfig *url.URL) (transport.StreamDialer, error) {
		lenStr := wrapConfig.Opaque
		fixedLen, err := strconv.Atoi(lenStr)
		if err != nil {
			return nil, fmt.Errorf("invalid tlsfrag option: %v. It should be in tlsfrag:<number> format", lenStr)
		}
		return tlsfrag.NewFixedLenStreamDialer(baseDialer, fixedLen)
	})
	p.RegisterPacketDialerWrapper("tlsfrag", func(baseDialer transport.PacketDialer, wrapConfig *url.URL) (transport.PacketDialer, error) {
		return nil, errors.New("tlsfrag is not supported for PacketDialers")
	})

	return p
}

// WrapStreamDialerFunc wraps a [transport.StreamDialer] based on the wrapConfig.
type WrapStreamDialerFunc func(dialer transport.StreamDialer, wrapConfig *url.URL) (transport.StreamDialer, error)

// RegisterStreamDialerWrapper will register a wrapper for stream dialers under the given subtype.
func (p *ConfigParser) RegisterStreamDialerWrapper(subtype string, wrapper WrapStreamDialerFunc) error {
	if p.sdWrapers == nil {
		p.sdWrapers = make(map[string]WrapStreamDialerFunc)
	}

	if _, found := p.sdWrapers[subtype]; found {
		return fmt.Errorf("config parser %v for StreamDialer added twice", subtype)
	}
	p.sdWrapers[subtype] = wrapper
	return nil
}

// WrapPacketDialerFunc wraps a [transport.PacketDialer] based on the wrapConfig.
type WrapPacketDialerFunc func(dialer transport.PacketDialer, wrapConfig *url.URL) (transport.PacketDialer, error)

// RegisterPacketDialerWrapper will register a wrapper for packet dialers under the given subtype.
func (p *ConfigParser) RegisterPacketDialerWrapper(subtype string, wrapper WrapPacketDialerFunc) error {
	if p.pdWrappers == nil {
		p.pdWrappers = make(map[string]WrapPacketDialerFunc)
	}

	if _, found := p.pdWrappers[subtype]; found {
		return fmt.Errorf("config parser %v for PacketDialer added twice", subtype)
	}
	p.pdWrappers[subtype] = wrapper
	return nil
}

func parseConfigPart(oneDialerConfig string) (*url.URL, error) {
	oneDialerConfig = strings.TrimSpace(oneDialerConfig)
	if oneDialerConfig == "" {
		return nil, errors.New("empty config part")
	}
	// Make it "<scheme>:" if it's only "<scheme>" to parse as a URL.
	if !strings.Contains(oneDialerConfig, ":") {
		oneDialerConfig += ":"
	}
	url, err := url.Parse(oneDialerConfig)
	if err != nil {
		return nil, fmt.Errorf("part is not a valid URL: %w", err)
	}
	return url, nil
}

// WrapStreamDialer creates a [transport.StreamDialer] according to transportConfig, using dialer as the
// base [transport.StreamDialer]. The given dialer must not be nil.
func (p *ConfigParser) WrapStreamDialer(dialer transport.StreamDialer, transportConfig string) (transport.StreamDialer, error) {
	if dialer == nil {
		return nil, errors.New("base dialer must not be nil")
	}
	transportConfig = strings.TrimSpace(transportConfig)
	if transportConfig == "" {
		return dialer, nil
	}
	for _, part := range strings.Split(transportConfig, "|") {
		url, err := parseConfigPart(part)
		if err != nil {
			return nil, err
		}
		w, ok := p.sdWrapers[url.Scheme]
		if !ok {
			return nil, fmt.Errorf("config scheme '%v' is not supported", url.Scheme)
		}
		dialer, err = w(dialer, url)
		if err != nil {
			return nil, err
		}
	}
	return dialer, nil
}

// WrapPacketDialer creates a [transport.PacketDialer] according to transportConfig, using dialer as the
// base [transport.PacketDialer]. The given dialer must not be nil.
func (p *ConfigParser) WrapPacketDialer(dialer transport.PacketDialer, transportConfig string) (transport.PacketDialer, error) {
	if dialer == nil {
		return nil, errors.New("base dialer must not be nil")
	}
	transportConfig = strings.TrimSpace(transportConfig)
	if transportConfig == "" {
		return dialer, nil
	}
	for _, part := range strings.Split(transportConfig, "|") {
		url, err := parseConfigPart(part)
		if err != nil {
			return nil, err
		}
		w, ok := p.pdWrappers[url.Scheme]
		if !ok {
			return nil, fmt.Errorf("config scheme '%v' is not supported", url.Scheme)
		}
		dialer, err = w(dialer, url)
		if err != nil {
			return nil, err
		}
	}
	return dialer, nil
}

// NewpacketListener creates a new [transport.PacketListener] according to the given config,
// the config must contain only one "ss://" segment.
// TODO: make NewPacketListener configurable.
func NewPacketListener(transportConfig string) (transport.PacketListener, error) {
	if transportConfig = strings.TrimSpace(transportConfig); transportConfig == "" {
		return nil, errors.New("config is required")
	}
	if strings.Contains(transportConfig, "|") {
		return nil, errors.New("multi-part config is not supported")
	}

	url, err := parseConfigPart(transportConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	// Please keep scheme list sorted.
	switch strings.ToLower(url.Scheme) {
	case "ss":
		// TODO: support nested dialer, the last part must be "ss://"
		return newShadowsocksPacketListenerFromURL(url)
	default:
		return nil, fmt.Errorf("config scheme '%v' is not supported", url.Scheme)
	}
}

func SanitizeConfig(transportConfig string) (string, error) {
	// Do nothing if the config is empty
	if transportConfig == "" {
		return "", nil
	}
	// Split the string into parts
	parts := strings.Split(transportConfig, "|")

	// Iterate through each part
	for i, part := range parts {
		u, err := parseConfigPart(part)
		if err != nil {
			return "", fmt.Errorf("failed to parse config part: %w", err)
		}
		scheme := strings.ToLower(u.Scheme)
		switch scheme {
		case "ss":
			parts[i], _ = sanitizeShadowsocksURL(u)
		case "socks5":
			parts[i], _ = sanitizeSocks5URL(u)
		case "override", "split", "tls", "tlsfrag":
			// No sanitization needed
			parts[i] = u.String()
		default:
			parts[i] = scheme + "://UNKNOWN"
		}
	}
	// Join the parts back into a string
	return strings.Join(parts, "|"), nil
}

func sanitizeSocks5URL(u *url.URL) (string, error) {
	const redactedPlaceholder = "REDACTED"
	if u.User != nil {
		u.User = url.User(redactedPlaceholder)
		return u.String(), nil
	}
	return u.String(), nil
}
