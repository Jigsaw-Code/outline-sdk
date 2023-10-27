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

/*
Package config provides convenience functions to create dialer objects based on a text config.
This is experimental and mostly for illustrative purposes at this point.

Configurable transports simplifies the way you create and manage transports.
With the [config] package, you can use [NewPacketDialer] and [NewStreamDialer] to create dialers using a simple text string.

Key Benefits:

  - Ease of Use: Create transports effortlessly by providing a textual configuration, reducing boilerplate code.
  - Serialization: Easily share configurations with users or between different parts of your application, including your Go backend.
  - Dynamic Configuration: Set your app's transport settings at runtime.
  - DPI Evasion: Advanced nesting and configuration options help you evade Deep Packet Inspection (DPI).

# Config Format

The configuration string is composed of parts separated by the `|` symbol, which define nested dialers.
For example, `A|B` means dialer `B` takes dialer `A` as its input.
An empty string represents the direct TCP/UDP dialer, and is used as the input to the first cofigured dialer.

Each dialer configuration follows a URL format, where the scheme defines the type of Dialer. Supported formats include:

Shadowsocks proxy (compatible with Outline's access keys, package [shadowsocks])

	ss://[USERINFO]@[HOST]:[PORT]?prefix=[PREFIX]

SOCKS5 proxy (currently streams only, package [socks5])

	socks5://[HOST]:[PORT]

Stream split transport (streams only, package [split])

It takes the length of the prefix. The stream will be split when PREFIX_LENGTH bytes are first written.

	split:[PREFIX_LENGTH]

TLS transport (currently streams only, package [tls])

The sni parameter defines the name to be sent in the TLS SNI. It can be empty.
The certname parameter defines what name to validate against the server certificate.

	tls:sni=[SNI]&certname=[CERT_NAME]

# Examples

Packet splitting - To split outgoing streams on bytes 2 and 123, you can use:

	split:2|split:123

SOCKS5-over-TLS, with domain-fronting - To tunnel SOCKS5 over TLS, and set the SNI to decoy.example.com, while still validating against your host name, use:

	tls:sni=decoy.example.com&certname=[HOST]|socks5:[HOST]:[PORT]

Onion Routing with Shadowsocks - To route your traffic through three Shadowsocks servers, similar to [Onion Routing], use:

	ss://[USERINFO1]@[HOST1]:[PORT1]|ss://[USERINFO2]@[HOST2]:[PORT2]|ss://[USERINFO3]@[HOST3]:[PORT3]

In that case, HOST1 will be your entry node, and HOST3 will be your exit node.

DPI Evasion - To add packet splitting to a Shadowsocks server for enhanced DPI evasion, use:

	split:2|ss://[USERINFO]@[HOST]:[PORT]

[Onion Routing]: https://en.wikipedia.org/wiki/Onion_routing
*/
package config

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Jigsaw-Code/outline-sdk/transport/socks5"
	"github.com/Jigsaw-Code/outline-sdk/transport/split"
)

func parseConfigPart(oneDialerConfig string) (*url.URL, error) {
	oneDialerConfig = strings.TrimSpace(oneDialerConfig)
	if oneDialerConfig == "" {
		return nil, errors.New("empty config part")
	}
	// Make it "<scheme>:" it it's only "<scheme>" to parse as a URL.
	if !strings.Contains(oneDialerConfig, ":") {
		oneDialerConfig += ":"
	}
	url, err := url.Parse(oneDialerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config part: %w", err)
	}
	return url, nil
}

// NewStreamDialer creates a new [transport.StreamDialer] according to the given config.
func NewStreamDialer(transportConfig string) (transport.StreamDialer, error) {
	return WrapStreamDialer(&transport.TCPStreamDialer{}, transportConfig)
}

// WrapStreamDialer created a [transport.StreamDialer] according to transportConfig, using dialer as the
// base [transport.StreamDialer]. The given dialer must not be nil.
func WrapStreamDialer(dialer transport.StreamDialer, transportConfig string) (transport.StreamDialer, error) {
	if dialer == nil {
		return nil, errors.New("base dialer must not be nil")
	}
	transportConfig = strings.TrimSpace(transportConfig)
	if transportConfig == "" {
		return dialer, nil
	}
	var err error
	for _, part := range strings.Split(transportConfig, "|") {
		dialer, err = newStreamDialerFromPart(dialer, part)
		if err != nil {
			return nil, err
		}
	}
	return dialer, nil
}

func newStreamDialerFromPart(innerDialer transport.StreamDialer, oneDialerConfig string) (transport.StreamDialer, error) {
	url, err := parseConfigPart(oneDialerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config part: %w", err)
	}

	// Please keep scheme list sorted.
	switch strings.ToLower(url.Scheme) {
	case "socks5":
		endpoint := transport.StreamDialerEndpoint{Dialer: innerDialer, Address: url.Host}
		return socks5.NewStreamDialer(&endpoint)

	case "split":
		prefixBytesStr := url.Opaque
		prefixBytes, err := strconv.Atoi(prefixBytesStr)
		if err != nil {
			return nil, fmt.Errorf("prefixBytes is not a number: %v. Split config should be in split:<number> format", prefixBytesStr)
		}
		return split.NewStreamDialer(innerDialer, int64(prefixBytes))

	case "ss":
		return newShadowsocksStreamDialerFromURL(innerDialer, url)

	case "tls":
		return newTlsStreamDialerFromURL(innerDialer, url)

	default:
		return nil, fmt.Errorf("config scheme '%v' is not supported", url.Scheme)
	}
}

// NewPacketDialer creates a new [transport.PacketDialer] according to the given config.
func NewPacketDialer(transportConfig string) (dialer transport.PacketDialer, err error) {
	dialer = &transport.UDPPacketDialer{}
	transportConfig = strings.TrimSpace(transportConfig)
	if transportConfig == "" {
		return dialer, nil
	}
	for _, part := range strings.Split(transportConfig, "|") {
		dialer, err = newPacketDialerFromPart(dialer, part)
		if err != nil {
			return nil, err
		}
	}
	return dialer, nil
}

func newPacketDialerFromPart(innerDialer transport.PacketDialer, oneDialerConfig string) (transport.PacketDialer, error) {
	url, err := parseConfigPart(oneDialerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config part: %w", err)
	}

	// Please keep scheme list sorted.
	switch strings.ToLower(url.Scheme) {
	case "socks5":
		return nil, errors.New("socks5 is not supported for PacketDialers")

	case "split":
		return nil, errors.New("split is not supported for PacketDialers")

	case "ss":
		return newShadowsocksPacketDialerFromURL(innerDialer, url)

	case "tls":
		return nil, errors.New("tls is not yet supported for PacketDialers")

	default:
		return nil, fmt.Errorf("config scheme '%v' is not supported", url.Scheme)
	}
}

// NewpacketListener creates a new [transport.PacketListener] according to the given config,
// the config must contain only one "ss://" segment.
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
