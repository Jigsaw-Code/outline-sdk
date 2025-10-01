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

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Jigsaw-Code/outline-sdk/x/mobileproxy"
	"github.com/Jigsaw-Code/outline-sdk/x/mobileproxy/psiphon"
	"github.com/Jigsaw-Code/outline-sdk/x/smart"
)

// RegisterErrorConfig registers a config that creates a dialer that always outputs an error.
// The config looks like "error: my error message".
func RegisterErrorConfig(opt *mobileproxy.SmartDialerOptions, name string) {
	opt.RegisterFallbackParser(name, func(ctx context.Context, yamlNode smart.YAMLNode) (transport.StreamDialer, string, error) {
		switch typed := yamlNode.(type) {
		case string:
			dialer := transport.FuncStreamDialer(func(ctx context.Context, addr string) (transport.StreamConn, error) {
				return nil, errors.New(typed)
			})
			return dialer, typed, nil
		default:
			return nil, "", fmt.Errorf("invalid error dialer config")
		}
	})
}

func main() {
	configFlag := flag.String("config", "", "Smart Dialer config")
	testDomainsFlag := flag.String("domains", "", "The test domains to find strategies.")
	addrFlag := flag.String("localAddr", "localhost:8080", "Local proxy address")
	urlProxyPrefixFlag := flag.String("proxyPath", "/proxy", "Path where to run the URL proxy. Set to empty (\"\") to disable it.")
	flag.Parse()

	// TODO(fortuna): add strategy cache.
	opts := mobileproxy.NewSmartDialerOptions(mobileproxy.NewListFromLines(*testDomainsFlag), *configFlag)
	opts.SetLogWriter(mobileproxy.NewStderrLogWriter())
	psiphon.RegisterFallbackParser(opts, "psiphon")
	RegisterErrorConfig(opts, "error")
	dialer, err := opts.NewStreamDialer()
	if err != nil {
		log.Fatalf("NewStreamDialerFromConfig failed: %v", err)
	}
	proxy, err := mobileproxy.RunProxy(*addrFlag, dialer)
	if err != nil {
		log.Fatalf("RunProxy failed: %v", err)
	}
	if *urlProxyPrefixFlag != "" {
		proxy.AddURLProxy(*urlProxyPrefixFlag, dialer)
	}
	log.Printf("Proxy listening on %v", proxy.Address())

	// Wait for interrupt signal to stop the proxy.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig
	log.Print("Shutting down")
	proxy.Stop(2)
}
