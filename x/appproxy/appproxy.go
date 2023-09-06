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

// Package appproxy provides convenience utilities to help applications run a local proxy
// and use that to configure their networking libraries.
//
// This package is suitable for use with Go Mobile, making it a convenient way to integrate with mobile apps.
package appproxy

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/x/config"
	"github.com/Jigsaw-Code/outline-sdk/x/httpproxy"
)

// Proxy enables you to get the actual address bound by the server and stop the service when no longer needed.
type Proxy struct {
	address string
	server  *http.Server
}

// Address returns the actual IP and port the server is bound to.
func (p *Proxy) Address() string {
	return p.address
}

// Stop gracefully stops the proxy service, waiting for at most timeout seconds before forcefully closing it.
// The function takes a timeoutSeconds number instead of a [time.Duration] so it's compatible with Go Mobile.
func (p *Proxy) Stop(timeoutSeconds int) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
	defer cancel()
	if err := p.server.Shutdown(ctx); err != nil {
		log.Fatalf("Failed to shutdown gracefully: %v", err)
		p.server.Close()
	}
}

// RunProxy runs a local web proxy that listens on localAddress, and uses the transportConfig to
// create a [transport.StreamDialer] that is used to connect to the requested destination.
func RunProxy(localAddress string, transportConfig string) (*Proxy, error) {
	dialer, err := config.NewStreamDialer(transportConfig)
	if err != nil {
		return nil, fmt.Errorf("could not create dialer: %w", err)
	}

	listener, err := net.Listen("tcp", localAddress)
	if err != nil {
		return nil, fmt.Errorf("could not listen on address %v: %v", localAddress, err)
	}

	server := &http.Server{Handler: httpproxy.NewConnectHandler(dialer)}
	go server.Serve(listener)

	return &Proxy{address: listener.Addr().String(), server: server}, nil
}
