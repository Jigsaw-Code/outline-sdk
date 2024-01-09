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
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Jigsaw-Code/outline-sdk/transport/socks5"
	"github.com/Jigsaw-Code/outline-sdk/transport/split"
	"github.com/Jigsaw-Code/outline-sdk/transport/tlsfrag"
)

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

	case "tlsfrag":
		lenStr := url.Opaque
		fixedLen, err := strconv.Atoi(lenStr)
		if err != nil {
			return nil, fmt.Errorf("invalid tlsfrag option: %v. It should be in tlsfrag:<number> format", lenStr)
		}
		return tlsfrag.NewFixedLenStreamDialer(innerDialer, fixedLen)

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
		case "ss", "socks5", "vless":
			parts[i], _ = sanitizeURL(u)
		case "split", "tls", "tlsfrag":
			// No sanitization needed
			parts[i] = u.String()
		default:
			parts[i] = scheme + "://UNKNOWN"
		}
	}

	// Join the parts back into a string
	return strings.Join(parts, "|"), nil
}

func sanitizeURL(u *url.URL) (string, error) {
	const redactedPlaceholder = "REDACTED"
	scheme := strings.ToLower(u.Scheme)
	if u.User != nil {
		u.User = url.User(redactedPlaceholder)
		return u.String(), nil
	} else {
		newURL, err := checkBase64Encoding(u)
		if err != nil {
			return scheme + ":" + redactedPlaceholder, err
		}
		if newURL.User != nil {
			newURL.User = url.User(redactedPlaceholder)
			return newURL.String(), nil
		} else {
			// If no user info is found, return the scheme and redacted placeholder
			return scheme + ":" + redactedPlaceholder, nil
		}
	}
}

// checkBase64Encoding checks if url has scheme://host#fragment with host being base64
// encoded string. iIf so, then decodes the base64 string and parses it as a url
func checkBase64Encoding(u *url.URL) (*url.URL, error) {
	// check if url has scheme://host#fragment format with host being base64 encoded string
	if u.Host != "" {
		decoded, err := base64.URLEncoding.WithPadding(base64.NoPadding).DecodeString(u.Host)
		fmt.Printf("decoded string: %v\n", string(decoded))
		if err != nil {
			// If decoding fails, return the original url with error
			return u, fmt.Errorf("failed to decode host string [%v]: %w", u.String(), err)
		}
		newURL, err := url.Parse(strings.ToLower(u.Scheme) + "://" + string(decoded) + u.Fragment)
		fmt.Println(newURL)
		if err != nil {
			// if parsing fails, return the original url with error
			return u, fmt.Errorf("failed to parse config part: %w", err)
		}
		// extend this check to see if decoded string contains contains other valid fields
		if newURL.User != nil {
			return newURL, nil
		} else {
			return u, nil
		}
	}
	return u, nil
}
