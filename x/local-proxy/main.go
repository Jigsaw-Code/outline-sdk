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
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Jigsaw-Code/outline-sdk/transport/shadowsocks"
	"github.com/Jigsaw-Code/outline-sdk/transport/socks5"
	"github.com/Jigsaw-Code/outline-sdk/x/local-proxy/proxy/httpproxywrapper"
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
	addrFlag := flag.String("addr", ":54321", "Local proxy addr")
	flag.Parse()

	dialer, err := makeStreamDialer(*transportFlag)
	if err != nil {
		log.Fatalf("Could not create dialer: %+v\n", err)
	}

	proxy := httpproxy.NewConnectHandler(dialer)
	proxy.OnError = httpproxy.OnError

	if err := proxy.StartServer(*addrFlag); err != nil {
		log.Fatal("Proxy server failed")
	}

	log.Println("Proxy server started on ", proxy.GetAddr())

	// Wait for kill/interrupt signal to stop the proxy with a timeout of 5 seconds.

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, os.Kill)

	<-sig
	log.Println("Shutting down the proxy server")

	serverStopped := make(chan struct{})

	go func() {
		if err = proxy.StopServer(); err != nil {
			log.Println("Failed to stop server: ", err.Error())
		}

		close(serverStopped)
	}()

	select {
	case <-time.After(5 * time.Second):
		log.Println("Shutdown timed out")
	case <-serverStopped:
	}
}
