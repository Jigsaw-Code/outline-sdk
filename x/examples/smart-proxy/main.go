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
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Jigsaw-Code/outline-sdk/x/config"
	"github.com/Jigsaw-Code/outline-sdk/x/httpproxy"
	"github.com/Jigsaw-Code/outline-sdk/x/smart"
)

var debugLog log.Logger = *log.New(io.Discard, "", 0)

type stringArrayFlagValue []string

func (v *stringArrayFlagValue) String() string {
	return fmt.Sprint(*v)
}

func (v *stringArrayFlagValue) Set(value string) error {
	*v = append(*v, value)
	return nil
}

func supportsHappyEyeballs(dialer transport.StreamDialer) bool {
	// Some proxy protocols, most notably Shadowsocks, can't communicate connection success.
	// Our shadowsocks.StreamDialer will return a connection successfully as long as it can
	// connect to the proxy server, regardless of whether it can connect to the target.
	// This breaks HappyEyeballs.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	conn, err := dialer.DialStream(ctx, "invalid:0")
	cancel()
	if conn != nil {
		conn.Close()
	}
	// If the dialer returns success on an invalid address, it doesn't support Happy Eyeballs.
	return err != nil
}

func main() {
	verboseFlag := flag.Bool("v", false, "Enable debug output")
	addrFlag := flag.String("localAddr", "localhost:1080", "Local proxy address")
	configFlag := flag.String("config", "config.json", "Address of the config file")
	transportFlag := flag.String("transport", "", "The base transport for the connections")
	var domainsFlag stringArrayFlagValue
	flag.Var(&domainsFlag, "domain", "The test domains to find strategies.")

	flag.Parse()
	if *verboseFlag {
		debugLog = *log.New(os.Stderr, "", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)
	}

	if len(domainsFlag) == 0 {
		log.Fatal("Must specify flag --domain")
	}

	if *configFlag == "" {
		log.Fatal("Must specify flag --config")
	}

	finderConfig, err := os.ReadFile(*configFlag)
	if err != nil {
		log.Fatalf("Could not read config: %v", err)
	}

	configParser := config.NewDefaultConfigParser()
	packetDialer, err := configParser.WrapPacketDialer(&transport.UDPDialer{}, *transportFlag)
	if err != nil {
		log.Fatalf("Could not create packet dialer: %v", err)
	}
	streamDialer, err := configParser.WrapStreamDialer(&transport.TCPDialer{}, *transportFlag)
	if err != nil {
		log.Fatalf("Could not create stream dialer: %v", err)
	}
	if !supportsHappyEyeballs(streamDialer) {
		fmt.Println("⚠️ Warning: base transport is not compatible with Happy Eyeballs. Disabling IPv6.")
		innerDialer := streamDialer
		// Disable IPv6 if the dialer doesn't support HappyEyballs.
		streamDialer = transport.FuncStreamDialer(func(ctx context.Context, addr string) (transport.StreamConn, error) {
			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			if ip := net.ParseIP(host); ip != nil && ip.To4() == nil {
				return nil, fmt.Errorf("IPv6 not supported")
			}
			return innerDialer.DialStream(ctx, addr)
		})
	}
	finder := smart.StrategyFinder{
		LogWriter:    debugLog.Writer(),
		TestTimeout:  5 * time.Second,
		StreamDialer: streamDialer,
		PacketDialer: packetDialer,
	}

	fmt.Println("Finding strategy")
	startTime := time.Now()
	dialer, err := finder.NewDialer(context.Background(), domainsFlag, finderConfig)
	if err != nil {
		log.Fatalf("Failed to find dialer: %v", err)
	}
	fmt.Printf("Found strategy in %0.2fs\n", time.Since(startTime).Seconds())
	logDialer := transport.FuncStreamDialer(func(ctx context.Context, address string) (transport.StreamConn, error) {
		conn, err := dialer.DialStream(ctx, address)
		if err != nil {
			debugLog.Printf("Failed to dial %v: %v\n", address, err)
		}
		return conn, err
	})

	listener, err := net.Listen("tcp", *addrFlag)
	if err != nil {
		log.Fatalf("Could not listen on address %v: %v", *addrFlag, err)
	}
	defer listener.Close()
	fmt.Printf("Proxy listening on %v\n", listener.Addr().String())

	server := http.Server{
		Handler:  httpproxy.NewProxyHandler(logDialer),
		ErrorLog: &debugLog,
	}
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error running web server: %v", err)
		}
	}()

	// Wait for interrupt signal to stop the proxy.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig
	fmt.Println("Shutting down")
	// Gracefully shut down the server, with a 5s timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Failed to shutdown gracefully: %v", err)
	}
}
