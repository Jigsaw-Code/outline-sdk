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
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Jigsaw-Code/outline-sdk/transport/shadowsocks"
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
		config, err := parseShadowsocksURL(url)
		if err != nil {
			return nil, err
		}
		endpoint := &transport.StreamDialerEndpoint{Dialer: innerDialer, Address: config.serverAddress}
		dialer, err := shadowsocks.NewStreamDialer(endpoint, config.cryptoKey)
		if err != nil {
			return nil, err
		}
		if len(config.prefix) > 0 {
			dialer.SaltGenerator = shadowsocks.NewPrefixSaltGenerator(config.prefix)
		}
		return dialer, nil

	case "split":
		prefixBytesStr := url.Opaque
		prefixBytes, err := strconv.Atoi(prefixBytesStr)
		if err != nil {
			return nil, fmt.Errorf("prefixBytes is not a number: %v. Split config should be in split:<number> format", prefixBytesStr)
		}
		return split.NewStreamDialer(innerDialer, int64(prefixBytes))

	default:
		return nil, fmt.Errorf("access key scheme %v:// is not supported", url.Scheme)
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
		return nil, errors.New("SOCKS5 PacketDialer is not implemented")

	case "ss":
		config, err := parseShadowsocksURL(url)
		if err != nil {
			return nil, err
		}
		endpoint := &transport.PacketDialerEndpoint{Dialer: innerDialer, Address: config.serverAddress}
		listener, err := shadowsocks.NewPacketListener(endpoint, config.cryptoKey)
		if err != nil {
			return nil, err
		}
		dialer := transport.PacketListenerDialer{Listener: listener}
		return dialer, nil

	case "split":
		return nil, errors.New("split is not supported for PacketDialers")

	default:
		return nil, fmt.Errorf("access key scheme %v:// is not supported", url.Scheme)
	}
}

type shadowsocksConfig struct {
	serverAddress string
	cryptoKey     *shadowsocks.EncryptionKey
	prefix        []byte
}

func parseShadowsocksURL(url *url.URL) (*shadowsocksConfig, error) {
	config := &shadowsocksConfig{}
	if url.Host == "" {
		return nil, errors.New("host not specified")
	}
	config.serverAddress = url.Host
	cipherInfoBytes, err := base64.URLEncoding.WithPadding(base64.NoPadding).DecodeString(url.User.String())
	if err != nil {
		return nil, fmt.Errorf("failed to decode cipher info [%v]: %w", url.User.String(), err)
	}
	cipherName, secret, found := strings.Cut(string(cipherInfoBytes), ":")
	if !found {
		return nil, errors.New("invalid cipher info: no ':' separator")
	}
	config.cryptoKey, err = shadowsocks.NewEncryptionKey(cipherName, secret)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}
	prefixStr := url.Query().Get("prefix")
	if len(prefixStr) > 0 {
		config.prefix, err = parseStringPrefix(prefixStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse prefix: %w", err)
		}
	}
	return config, nil
}

func parseStringPrefix(utf8Str string) ([]byte, error) {
	runes := []rune(utf8Str)
	rawBytes := make([]byte, len(runes))
	for i, r := range runes {
		if (r & 0xFF) != r {
			return nil, fmt.Errorf("character out of range: %d", r)
		}
		rawBytes[i] = byte(r)
	}
	return rawBytes, nil
}
