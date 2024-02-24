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
	"flag"
	"log"
	"os"
	"os/signal"

	"github.com/Jigsaw-Code/outline-sdk/x/mobileproxy"
)

func main() {
	transportFlag := flag.String("transport", "", "Transport config")
	addrFlag := flag.String("localAddr", "localhost:8080", "Local proxy address")
	urlProxyPrefixFlag := flag.String("proxyPath", "/proxy", "Path where to run the URL proxy. Set to empty (\"\") to disable it.")
	flag.Parse()

	dialer, err := mobileproxy.NewStreamDialerFromConfig(*transportFlag)
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
