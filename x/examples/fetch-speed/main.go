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
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/x/config"
)

var debugLog log.Logger = *log.New(io.Discard, "", 0)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [flags...] <url>\n", path.Base(os.Args[0]))
		flag.PrintDefaults()
	}
}

func main() {
	verboseFlag := flag.Bool("v", false, "Enable debug output")
	transportFlag := flag.String("transport", "", "Transport config")
	addressFlag := flag.String("address", "", "Address to connect to. If empty, use the URL authority")
	methodFlag := flag.String("method", "GET", "The HTTP method to use")

	flag.Parse()

	if *verboseFlag {
		debugLog = *log.New(os.Stderr, "[DEBUG] ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)
	}
	var overrideHost, overridePort string
	if *addressFlag != "" {
		var err error
		overrideHost, overridePort, err = net.SplitHostPort(*addressFlag)
		if err != nil {
			// Fail to parse. Assume the flag is host only.
			overrideHost = *addressFlag
			overridePort = ""
		}
	}

	url := flag.Arg(0)
	if url == "" {
		log.Println("Need to pass the URL to fetch in the command-line")
		flag.Usage()
		os.Exit(1)
	}

	dialer, err := config.NewStreamDialer(*transportFlag)
	if err != nil {
		log.Fatalf("Could not create dialer: %v\n", err)
	}
	dialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, fmt.Errorf("invalid address: %w", err)
		}
		if overrideHost != "" {
			host = overrideHost
		}
		if overridePort != "" {
			port = overridePort
		}
		if !strings.HasPrefix(network, "tcp") {
			return nil, fmt.Errorf("protocol not supported: %v", network)
		}
		return dialer.DialStream(ctx, net.JoinHostPort(host, port))
	}
	httpClient := &http.Client{Transport: &http.Transport{DialContext: dialContext}, Timeout: 5 * time.Second}

	req, err := http.NewRequest(*methodFlag, url, nil)
	if err != nil {
		log.Fatalln("Failed to create request:", err)
	}

	// Start timing the download
	startTime := time.Now()

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Fatalf("HTTP request failed: %v\n", err)
	}
	defer resp.Body.Close()

	// Use a buffer to read the response body
	buf := new(bytes.Buffer)

	written, err := io.Copy(buf, resp.Body)
	fmt.Println()
	if err != nil {
		log.Fatalf("Read of page body failed: %v\n", err)
	}

	// Calculate the download speed
	endTime := time.Now()
	duration := endTime.Sub(startTime).Seconds()
	downloadSpeed := float64(written) / duration

	writtenMB := float64(written) / 1048576

	fmt.Printf("\nDownloaded %.2f MB in %.2fs\n", writtenMB, duration)
	fmt.Printf("\nDownloaded Speed: %.2f MB/s\n", downloadSpeed/1048576)

	if *verboseFlag {
		for k, v := range resp.Header {
			debugLog.Printf("%v: %v", k, v)
		}
	}
}
