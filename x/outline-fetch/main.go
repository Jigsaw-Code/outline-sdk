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

package main

import (
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Jigsaw-Code/outline-sdk/transport/shadowsocks"
	"github.com/Jigsaw-Code/outline-sdk/transport/socks5"
)

func makeStreamDialer(transportConfig string) (transport.StreamDialer, error) {
	if transportConfig == "" {
		return &transport.TCPStreamDialer{}, nil
	}

	accessKeyURL, err := url.Parse(transportConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse access key: %w", err)
	}
	switch accessKeyURL.Scheme {

	case "socks5":
		return socks5.NewStreamDialer(&transport.TCPEndpoint{Address: accessKeyURL.Host})

	case "ss":
		cipherInfoBytes, err := base64.URLEncoding.WithPadding(base64.NoPadding).DecodeString(accessKeyURL.User.String())
		if err != nil {
			return nil, fmt.Errorf("failed to decode cipher info [%v]: %v", accessKeyURL.User.String(), err)
		}
		cipherName, secret, found := strings.Cut(string(cipherInfoBytes), ":")
		if !found {
			return nil, fmt.Errorf("invalid cipher info: no ':' separator")
		}
		cryptoKey, err := shadowsocks.NewEncryptionKey(cipherName, secret)
		if err != nil {
			return nil, fmt.Errorf("failed to create cipher: %w", err)
		}

		dialer, err := shadowsocks.NewStreamDialer(&transport.TCPEndpoint{Address: accessKeyURL.Host}, cryptoKey)
		if err != nil {
			return nil, err
		}

		prefixStr := accessKeyURL.Query().Get("prefix")
		if len(prefixStr) > 0 {
			prefix, err := parseStringPrefix(prefixStr)
			if err != nil {
				return nil, fmt.Errorf("failed to parse prefix: %w", err)
			}
			dialer.SaltGenerator = shadowsocks.NewPrefixSaltGenerator(prefix)
		}
		return dialer, nil

	case "split":
		return nil, fmt.Errorf("split is not yet implemented")
		// TODO(fortuna): enable the code below after the split transport is submitted.
		// splitPoint, err := strconv.Atoi(accessKeyURL.Host)
		// if err != nil {
		// 	return nil, fmt.Errorf("splitPoint is not a number: %v. Split config should be in split:<number> format", accessKeyURL.Host)
		// }
		// return split.NewStreamDialer(&transport.TCPStreamDialer{}, int64(splitPoint))

	default:
		return nil, fmt.Errorf("access key scheme %v:// is not supported", accessKeyURL.Scheme)
	}
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

func main() {
	transportFlag := flag.String("transport", "", "Transport config")
	flag.Parse()

	url := flag.Arg(0)
	if url == "" {
		log.Fatal("Need to pass the URL to fetch in the command-line")
	}

	dialer, err := makeStreamDialer(*transportFlag)
	if err != nil {
		log.Fatalf("Could not create dialer: %v", err)
	}
	httpClient := &http.Client{Transport: &http.Transport{DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
		if !strings.HasPrefix(network, "tcp") {
			return nil, fmt.Errorf("protocol not supported: %v", network)
		}
		return dialer.Dial(ctx, addr)
	}}}

	resp, err := httpClient.Get(url)
	if err != nil {
		log.Fatalf("URL GET failed: %v", err)
	}
	defer resp.Body.Close()

	_, err = io.Copy(os.Stdout, resp.Body)
	if err != nil {
		log.Fatalf("Read of page body failed: %v", err)
	}
}
