// Copyright 2024 The Outline Authors
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
	"net/url"
	"strings"

	"github.com/Jigsaw-Code/outline-sdk/transport"
)

// ConfigModule enables the creation of network objects based on a config. The config is
// extensible by registering providers for config subtypes.
type ConfigModule struct {
	StreamDialers   ExtensibleProvider[transport.StreamDialer]
	PacketDialers   ExtensibleProvider[transport.PacketDialer]
	PacketListeners ExtensibleProvider[transport.PacketListener]
}

// NewDefaultConfigModule creates a [ConfigModule] with a set of default wrappers already registered.
func NewDefaultConfigModule() *ConfigModule {
	p := new(ConfigModule)

	p.StreamDialers.BaseInstance = &transport.TCPDialer{}
	p.PacketDialers.BaseInstance = &transport.UDPDialer{}
	p.PacketListeners.BaseInstance = &transport.UDPListener{}

	// Please keep the list in alphabetical order.
	registerDO53StreamDialer(&p.StreamDialers, "do53", p.StreamDialers.NewInstance, p.PacketDialers.NewInstance)
	registerDOHStreamDialer(&p.StreamDialers, "doh", p.StreamDialers.NewInstance)

	registerOverrideStreamDialer(&p.StreamDialers, "override", p.StreamDialers.NewInstance)
	registerOverridePacketDialer(&p.PacketDialers, "override", p.PacketDialers.NewInstance)

	registerSOCKS5StreamDialer(&p.StreamDialers, "socks5", p.StreamDialers.NewInstance)
	registerSOCKS5PacketDialer(&p.PacketDialers, "socks5", p.StreamDialers.NewInstance, p.PacketDialers.NewInstance)
	registerSOCKS5PacketListener(&p.PacketListeners, "socks5", p.StreamDialers.NewInstance, p.PacketDialers.NewInstance)

	registerSplitStreamDialer(&p.StreamDialers, "split", p.StreamDialers.NewInstance)

	registerShadowsocksStreamDialer(&p.StreamDialers, "ss", p.StreamDialers.NewInstance)
	registerShadowsocksPacketDialer(&p.PacketDialers, "ss", p.PacketDialers.NewInstance)
	registerShadowsocksPacketListener(&p.PacketListeners, "ss", p.PacketDialers.NewInstance)

	registerTLSStreamDialer(&p.StreamDialers, "tls", p.StreamDialers.NewInstance)

	registerTLSFragStreamDialer(&p.StreamDialers, "tlsfrag", p.StreamDialers.NewInstance)

	registerWebsocketStreamDialer(&p.StreamDialers, "ws", p.StreamDialers.NewInstance)
	registerWebsocketPacketDialer(&p.PacketDialers, "ws", p.StreamDialers.NewInstance)

	return p
}

// NewStreamDialer creates a [transport.StreamDialer] according to the config text.
func (p *ConfigModule) NewStreamDialer(ctx context.Context, configText string) (transport.StreamDialer, error) {
	config, err := ParseConfig(configText)
	if err != nil {
		return nil, err
	}
	return p.StreamDialers.NewInstance(ctx, config)
}

// NewPacketDialer creates a [transport.PacketDialer] according to the config text.
func (p *ConfigModule) NewPacketDialer(ctx context.Context, configText string) (transport.PacketDialer, error) {
	config, err := ParseConfig(configText)
	if err != nil {
		return nil, err
	}
	return p.PacketDialers.NewInstance(ctx, config)
}

// NewPacketListner creates a [transport.PacketListener] according to the config text.
func (p *ConfigModule) NewPacketListener(ctx context.Context, configText string) (transport.PacketListener, error) {
	config, err := ParseConfig(configText)
	if err != nil {
		return nil, err
	}
	return p.PacketListeners.NewInstance(ctx, config)
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

	var sanitized string
	for config != nil {
		var part string
		scheme := strings.ToLower(config.URL.Scheme)
		switch scheme {
		case "ss":
			part, err = sanitizeShadowsocksURL(config.URL)
			if err != nil {
				return "", err
			}
		case "socks5":
			part, err = sanitizeSOCKS5URL(&config.URL)
			if err != nil {
				return "", err
			}
		case "override", "split", "tls", "tlsfrag":
			// No sanitization needed
			part = config.URL.String()
		default:
			part = scheme + "://UNKNOWN"
		}
		if sanitized == "" {
			sanitized = part
		} else {
			sanitized = part + "|" + sanitized
		}
		config = config.BaseConfig
	}
	return sanitized, nil
}

func sanitizeSOCKS5URL(u *url.URL) (string, error) {
	const redactedPlaceholder = "REDACTED"
	if u.User != nil {
		u.User = url.User(redactedPlaceholder)
		return u.String(), nil
	}
	return u.String(), nil
}
