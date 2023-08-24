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
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"

	"github.com/Jigsaw-Code/outline-sdk/x/examples/internal/config"
	"github.com/Jigsaw-Code/outline-sdk/x/httpproxy"
)

func main() {
	transportFlag := flag.String("transport", "", "Transport config")
	addrFlag := flag.String("localAddr", "localhost:1080", "Local proxy address")
	flag.Parse()

	dialer, err := config.MakeStreamDialer(*transportFlag)
	if err != nil {
		log.Fatalf("Could not create dialer: %+v\n", err)
	}

	listener, err := net.Listen("tcp", *addrFlag)
	if err != nil {
		log.Fatalf("Could not listen on address %v: %v", *addrFlag, err)
	}
	fmt.Printf("Proxy listening on %v\n", listener.Addr().String())

	go func() {
		if err := http.Serve(listener, httpproxy.NewConnectHandler(dialer)); err != nil {
			log.Fatalf("Error starting web server: %v", err)
		}
	}()

	// Wait for interrupt signal to stop the proxy with a timeout of 5 seconds.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig
}
