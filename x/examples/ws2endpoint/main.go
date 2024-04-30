// Copyright 2024 Jigsaw Operations LLC
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
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Jigsaw-Code/outline-sdk/x/config"
	"golang.org/x/net/websocket"
)

func main() {
	addrFlag := flag.String("localAddr", "localhost:8080", "Local proxy address")
	transportFlag := flag.String("transport", "", "Transport config")
	endpointFlag := flag.String("endpoint", "", "Address of the target endpoint")
	pathPrefix := flag.String("path", "/", "Path where to run the Websocket forwarder")
	flag.Parse()

	dialer, err := config.NewDefaultConfigParser().WrapStreamDialer(&transport.TCPDialer{}, *transportFlag)
	if err != nil {
		log.Fatalf("Could not create dialer: %v", err)
	}
	if *endpointFlag == "" {
		log.Fatal("Must specify flag -endpoint")
	}
	endpoint := transport.StreamDialerEndpoint{Dialer: dialer, Address: *endpointFlag}

	listener, err := net.Listen("tcp", *addrFlag)
	if err != nil {
		log.Fatalf("Could not listen on address %v: %v", *addrFlag, err)
	}
	defer listener.Close()
	log.Printf("Proxy listening on %v\n", listener.Addr().String())

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Got request: %v\n", r)
		handler := func(wsConn *websocket.Conn) {
			targetConn, err := endpoint.ConnectStream(r.Context())
			if err != nil {
				log.Printf("Failed to upgrade: %v\n", err)
				w.WriteHeader(http.StatusBadGateway)
				return
			}
			defer targetConn.Close()
			go func() {
				io.Copy(targetConn, wsConn)
				targetConn.CloseWrite()
			}()
			io.Copy(wsConn, targetConn)
			wsConn.Close()
		}
		websocket.Server{Handler: handler}.ServeHTTP(w, r)
	})
	server := http.Server{Handler: http.StripPrefix(*pathPrefix, handler)}
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error running web server: %v", err)
		}
	}()

	// Wait for interrupt signal to stop the proxy.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig
	log.Println("Shutting down")
	// Gracefully shut down the server, with a 5s timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Failed to shutdown gracefully: %v", err)
	}
}
