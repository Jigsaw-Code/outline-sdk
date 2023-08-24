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
	"strings"

	"github.com/Jigsaw-Code/outline-sdk/x/examples/internal/config"
)

func main() {
	transportFlag := flag.String("transport", "", "Transport config")
	flag.Parse()

	url := flag.Arg(0)
	if url == "" {
		log.Fatal("Need to pass the URL to fetch in the command-line")
	}

	dialer, err := config.MakeStreamDialer(*transportFlag)
	if err != nil {
		log.Fatalf("Could not create dialer: %v", err)
	}
	httpClient := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				if !strings.HasPrefix(network, "tcp") {
					return nil, fmt.Errorf("protocol not supported: %v", network)
				}
				return dialer.Dial(ctx, addr)
			}}}

	resp, err := httpClient.Get(url)
	if err != nil {
		log.Fatalf("URL GET failed: %v", err)
	}
	defer resp.Body.Close()

	_, err = io.Copy(os.Stdout, resp.Body)
	if err != nil {
		log.Fatalf("Read of page body failed: %v", err)
	}
}
