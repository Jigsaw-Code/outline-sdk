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
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/textproto"
	"os"
	"path"
	"strings"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/x/configurl"
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
	var headersFlag stringArrayFlagValue
	flag.Var(&headersFlag, "H", "Raw HTTP Header line to add. It must not end in \\r\\n")
	timeoutSecFlag := flag.Int("timeout", 5, "Timeout in seconds")

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

	dialer, err := configurl.NewDefaultConfigToDialer().NewStreamDialer(*transportFlag)
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
	httpClient := &http.Client{
		Transport: &http.Transport{DialContext: dialContext},
		Timeout:   time.Duration(*timeoutSecFlag) * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, err := http.NewRequest(*methodFlag, url, nil)
	if err != nil {
		log.Fatalln("Failed to create request:", err)
	}
	headerText := strings.Join(headersFlag, "\r\n") + "\r\n\r\n"
	h, err := textproto.NewReader(bufio.NewReader(strings.NewReader(headerText))).ReadMIMEHeader()
	if err != nil {
		log.Fatalf("invalid header line: %v", err)
	}
	for name, values := range h {
		for _, value := range values {
			req.Header.Add(name, value)
		}
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
