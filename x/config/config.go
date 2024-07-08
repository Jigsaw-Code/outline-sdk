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
	"github.com/Jigsaw-Code/outline-sdk/transport/tlsfrag"
)

// ConfigToDialer enables the creation of stream and packet dialers based on a config. The config is
// extensible by registering wrappers for config subtypes.
type ConfigToDialer struct {
	// Base StreamDialer to create direct stream connections. If you need direct stream connections, this must not be nil.
	BaseStreamDialer transport.StreamDialer
	// Base PacketDialer to create direct packet connections. If you need direct packet connections, this must not be nil.
	BasePacketDialer transport.PacketDialer
	sdBuilders       map[string]NewStreamDialerFunc
	pdBuilders       map[string]NewPacketDialerFunc
}

// NewStreamDialerFunc wraps a Dialer based on the wrapConfig. The innerSD and innerPD functions can provide a base Stream and Packet Dialers if needed.
type NewStreamDialerFunc func(innerSD func() (transport.StreamDialer, error), innerPD func() (transport.PacketDialer, error), wrapConfig *url.URL) (transport.StreamDialer, error)

// NewPacketDialerFunc wraps a Dialer based on the wrapConfig. The innerSD and innerPD functions can provide a base Stream and Packet Dialers if needed.
type NewPacketDialerFunc func(innerSD func() (transport.StreamDialer, error), innerPD func() (transport.PacketDialer, error), wrapConfig *url.URL) (transport.PacketDialer, error)

// NewDefaultConfigToDialer creates a [ConfigToDialer] with a set of default wrappers already registered.
func NewDefaultConfigToDialer() *ConfigToDialer {
	p := new(ConfigToDialer)
	p.BaseStreamDialer = &transport.TCPDialer{}
	p.BasePacketDialer = &transport.UDPDialer{}

	// Please keep the list in alphabetical order.
	p.RegisterStreamDialerType("do53", wrapStreamDialerWithDO53)

	p.RegisterStreamDialerType("doh", wrapStreamDialerWithDOH)

	p.RegisterStreamDialerType("override", wrapStreamDialerWithOverride)
	p.RegisterPacketDialerType("override", wrapPacketDialerWithOverride)

	p.RegisterStreamDialerType("socks5", wrapStreamDialerWithSOCKS5)
	p.RegisterPacketDialerType("socks5", wrapPacketDialerWithSOCKS5)

	p.RegisterStreamDialerType("split", wrapStreamDialerWithSplit)

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

// RegisterStreamDialerType will register a wrapper for stream dialers under the given subtype.
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

// RegisterPacketDialerType will register a wrapper for packet dialers under the given subtype.
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

func parseConfig(configText string) ([]*url.URL, error) {
	parts := strings.Split(strings.TrimSpace(configText), "|")
	if len(parts) == 1 && parts[0] == "" {
		return []*url.URL{}, nil
	}
	urls := make([]*url.URL, 0, len(parts))
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
		urls = append(urls, url)
	}
	return urls, nil
}

// NewStreamDialer creates a [Dialer] according to transportConfig, using dialer as the
// base [Dialer]. The given dialer must not be nil.
func (p *ConfigToDialer) NewStreamDialer(transportConfig string) (transport.StreamDialer, error) {
	parts, err := parseConfig(transportConfig)
	if err != nil {
		return nil, err
	}
	return p.newStreamDialer(parts)
}

// NewPacketDialer creates a [Dialer] according to transportConfig, using dialer as the
// base [Dialer]. The given dialer must not be nil.
func (p *ConfigToDialer) NewPacketDialer(transportConfig string) (transport.PacketDialer, error) {
	parts, err := parseConfig(transportConfig)
	if err != nil {
		return nil, err
	}
	return p.newPacketDialer(parts)
}

func (p *ConfigToDialer) newStreamDialer(configParts []*url.URL) (transport.StreamDialer, error) {
	if len(configParts) == 0 {
		if p.BaseStreamDialer == nil {
			return nil, fmt.Errorf("base StreamDialer must not be nil")
		}
		return p.BaseStreamDialer, nil
	}
	thisURL := configParts[len(configParts)-1]
	innerConfig := configParts[:len(configParts)-1]
	newDialer, ok := p.sdBuilders[thisURL.Scheme]
	if !ok {
		return nil, fmt.Errorf("config scheme '%v' is not supported for Stream Dialers", thisURL.Scheme)
	}
	newSD := func() (transport.StreamDialer, error) {
		return p.newStreamDialer(innerConfig)
	}
	newPD := func() (transport.PacketDialer, error) {
		return p.newPacketDialer(innerConfig)
	}
	return newDialer(newSD, newPD, thisURL)
}

func (p *ConfigToDialer) newPacketDialer(configParts []*url.URL) (transport.PacketDialer, error) {
	if len(configParts) == 0 {
		if p.BasePacketDialer == nil {
			return nil, fmt.Errorf("base PacketDialer must not be nil")
		}
		return p.BasePacketDialer, nil
	}
	thisURL := configParts[len(configParts)-1]
	innerConfig := configParts[:len(configParts)-1]
	newDialer, ok := p.pdBuilders[thisURL.Scheme]
	if !ok {
		return nil, fmt.Errorf("config scheme '%v' is not supported for Packet Dialers", thisURL.Scheme)
	}
	newSD := func() (transport.StreamDialer, error) {
		return p.newStreamDialer(innerConfig)
	}
	newPD := func() (transport.PacketDialer, error) {
		return p.newPacketDialer(innerConfig)
	}
	return newDialer(newSD, newPD, thisURL)
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

func SanitizeConfig(transportConfig string) (string, error) {
	parts, err := parseConfig(transportConfig)
	if err != nil {
		return "", err
	}

	// Do nothing if the config is empty
	if len(parts) == 0 {
		return "", nil
	}

	// Iterate through each part
	textParts := make([]string, len(parts))
	for i, u := range parts {
		scheme := strings.ToLower(u.Scheme)
		switch scheme {
		case "ss":
			textParts[i], err = sanitizeShadowsocksURL(u)
			if err != nil {
				return "", err
			}
		case "socks5":
			textParts[i], err = sanitizeSocks5URL(u)
			if err != nil {
				return "", err
			}
		case "override", "split", "tls", "tlsfrag":
			// No sanitization needed
			textParts[i] = u.String()
		default:
			textParts[i] = scheme + "://UNKNOWN"
		}
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
