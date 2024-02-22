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
	"io"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/Jigsaw-Code/outline-sdk/x/mobileproxy"
)

func main() {
	transportFlag := flag.String("transport", "", "Transport config")
	flag.Parse()

	urlToFetch := flag.Arg(0)
	if urlToFetch == "" {
		log.Fatal("Need to pass the URL to fetch in the command-line")
	}

	dialer, err := mobileproxy.NewStreamDialerFromConfig(*transportFlag)
	if err != nil {
		log.Fatalf("NewStreamDialerFromConfig failed: %v", err)
	}
	proxy, err := mobileproxy.RunProxy("localhost:0", dialer)
	if err != nil {
		log.Fatalf("RunProxy failed: %v", err)
	}

	httpClient := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(&url.URL{Scheme: "http", Host: proxy.Address()})}}

	resp, err := httpClient.Get(urlToFetch)
	if err != nil {
		log.Fatalf("URL GET failed: %v", err)
	}
	defer resp.Body.Close()

	_, err = io.Copy(os.Stdout, resp.Body)
	if err != nil {
		log.Fatalf("Read of page body failed: %v", err)
	}

	proxy.Stop(5)
}
