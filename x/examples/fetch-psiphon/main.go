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
	"path"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/x/psiphon"
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
	configFlag := flag.String("config", "", "Config file")
	methodFlag := flag.String("method", "GET", "The HTTP method to use")

	flag.Parse()

	if *verboseFlag {
		debugLog = *log.New(os.Stderr, "[DEBUG] ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)
	}

	url := flag.Arg(0)
	if url == "" {
		log.Println("Need to pass the URL to fetch in the command-line")
		flag.Usage()
		os.Exit(1)
	}

	if *configFlag == "" {
		log.Println("Need to pass config file in the command-line")
		flag.Usage()
		os.Exit(1)
	}
	configJSON, err := os.ReadFile(*configFlag)
	if err != nil {
		log.Fatalf("Could not read config file: %v\n", err)
	}
	dialer, err := psiphon.NewStreamDialer(configJSON)
	if err != nil {
		log.Fatalf("Could not create dialer: %v\n", err)
	}
	defer dialer.Close()
	// TODO: Wait for tunnels to be active.
	time.Sleep(2 * time.Second)
	dialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
		return dialer.DialStream(ctx, addr)
	}
	httpClient := &http.Client{Transport: &http.Transport{DialContext: dialContext}, Timeout: 5 * time.Second}

	req, err := http.NewRequest(*methodFlag, url, nil)
	if err != nil {
		log.Fatalln("Failed to create request:", err)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Fatalf("HTTP request failed: %v\n", err)
	}
	defer resp.Body.Close()

	if *verboseFlag {
		for k, v := range resp.Header {
			debugLog.Printf("%v: %v", k, v)
		}
	}

	_, err = io.Copy(os.Stdout, resp.Body)
	fmt.Println()
	if err != nil {
		log.Fatalf("Read of page body failed: %v\n", err)
	}
}
