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

// Package config provides convenience functions to create [transport.StreamDialer] and [transport.PacketDialer]
// objects based on a text config. This is experimental and mostly for illustrative purposes at this point.
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

// NewStreamDialer creates a new [transport.StreamDialer] according to the given config.
func NewStreamDialer(transportConfig string) (dialer transport.StreamDialer, err error) {
	dialer = &transport.TCPStreamDialer{}
	transportConfig = strings.TrimSpace(transportConfig)
	if transportConfig == "" {
		return dialer, nil
	}
	for _, part := range strings.Split(transportConfig, "|") {
		dialer, err = newStreamDialerFromPart(dialer, part)
		if err != nil {
			return nil, err
		}
	}
	return dialer, nil
}

func newStreamDialerFromPart(innerDialer transport.StreamDialer, oneDialerConfig string) (transport.StreamDialer, error) {
	oneDialerConfig = strings.TrimSpace(oneDialerConfig)

	if oneDialerConfig == "" {
		return nil, errors.New("empty config part")
	}

	url, err := url.Parse(oneDialerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config part: %w", err)
	}

	switch url.Scheme {
	case "socks5":
		endpoint := transport.StreamDialerEndpoint{Dialer: innerDialer, Address: url.Host}
		return socks5.NewStreamDialer(&endpoint)

	case "ss":
		return newShadowsocksStreamDialerFromURL(innerDialer, url)

	case "split":
		prefixBytesStr := url.Opaque
		prefixBytes, err := strconv.Atoi(prefixBytesStr)
		if err != nil {
			return nil, fmt.Errorf("prefixBytes is not a number: %v. Split config should be in split:<number> format", prefixBytesStr)
		}
		return split.NewStreamDialer(innerDialer, int64(prefixBytes))

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
	oneDialerConfig = strings.TrimSpace(oneDialerConfig)

	if oneDialerConfig == "" {
		return nil, errors.New("empty config part")
	}

	url, err := url.Parse(oneDialerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config part: %w", err)
	}

	switch url.Scheme {
	case "socks5":
		return nil, errors.New("socks5 is not supported for PacketDialers")

	case "ss":
		return newShadowsocksPacketDialerFromURL(innerDialer, url)

	case "split":
		return nil, errors.New("split is not supported for PacketDialers")

	default:
		return nil, fmt.Errorf("config scheme '%v' is not supported", url.Scheme)
	}
}

// NewpacketListener creates a new [transport.PacketListener] according to the given config,
// the config must contain only one "ss://" segment.
func NewpacketListener(transportConfig string) (transport.PacketListener, error) {
	if transportConfig = strings.TrimSpace(transportConfig); transportConfig == "" {
		return nil, errors.New("config is required")
	}
	if strings.Contains(transportConfig, "|") {
		return nil, errors.New("multi-part config is not supported")
	}

	url, err := url.Parse(transportConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	if url.Scheme != "ss" {
		return nil, errors.New("config scheme must be 'ss' for a PacketListener")
	}

	// TODO: support nested dialer, the last part must be "ss://"
	return newShadowsocksPacketListenerFromURL(url)
}
