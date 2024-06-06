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
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/transport/tls"
	"github.com/Jigsaw-Code/outline-sdk/x/config"
	"gopkg.in/yaml.v3"
)

var debugLog log.Logger = *log.New(io.Discard, "", 0)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [flags...] <url>\n", path.Base(os.Args[0]))
		flag.PrintDefaults()
	}
}

type MeasurementConfig struct {
	NumAttempts *int     `yaml:"num_attempts,omitempty"`
	Domains     []string `yaml:"domains,omitempty"`
	ISPProxies  []string `yaml:"isp_proxies,omitempty"`
	Strategies  []string `yaml:"strategies,omitempty"`
}

type Measurement struct {
	Time         time.Time
	Domain       string
	ISP          string
	Country      string
	Strategy     string
	Attempt      int
	Img          string
	ErrorOp      string
	ErrorMessage string
}

type ISP struct {
	Transport   string
	Name        string
	CountryCode string
}

type ISPInfo struct {
	Data struct {
		IP          string
		ISP         string
		Carrier     string
		City        string
		Region      string
		CountryName string `json:"country_name,omitempty"`
		CountryCode string `json:"country_code,omitempty"`
	}
}

var transportToDialer = config.NewDefaultConfigToDialer()

func makeErrorMsg(err error, domain string) string {
	return strings.ReplaceAll(err.Error(), domain, "${DOMAIN}")
}

func newISP(ispProxy string) ISP {
	dialer, err := transportToDialer.NewStreamDialer(ispProxy)
	if err != nil {
		log.Fatalln("Could not create ISP dialer:", err)
	}
	dialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
		if !strings.HasPrefix(network, "tcp") {
			return nil, fmt.Errorf("protocol not supported: %v", network)
		}
		return dialer.DialStream(ctx, addr)
	}
	httpClient := &http.Client{
		Transport: &http.Transport{DialContext: dialContext},
		Timeout:   time.Duration(10) * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	req, err := http.NewRequest("GET", "https://checker.soax.com/api/ipinfo", nil)
	if err != nil {
		log.Fatalln("Failed to create request:", err)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Fatalln("HTTP request failed:", err)
	}
	var ispInfo ISPInfo
	func() {
		defer resp.Body.Close()
		jsonBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatalln("failed to get isp info:", err)
		}
		json.Unmarshal(jsonBytes, &ispInfo)
	}()
	return ISP{Transport: ispProxy, Name: ispInfo.Data.ISP, CountryCode: ispInfo.Data.CountryCode}
}

func main() {
	verboseFlag := flag.Bool("v", false, "Enable debug output")
	timeoutSecFlag := flag.Int("timeout", 15, "Timeout in seconds")
	configFlag := flag.String("config", "config.yml", "config to use")

	flag.Parse()

	if *verboseFlag {
		debugLog = *log.New(os.Stderr, "[DEBUG] ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)
	}

	cfg := MeasurementConfig{}
	configData, err := os.ReadFile(*configFlag)
	if err != nil {
		log.Fatalln("failed to read config:", err)
	}
	if err := yaml.Unmarshal(configData, &cfg); err != nil {
		log.Fatalln("Failed to parse config:", err)
	}

	// Collect ISPs
	isps := make([]ISP, 0, len(cfg.ISPProxies))
	for _, ispProxy := range cfg.ISPProxies {
		isp := newISP(ispProxy)
		debugLog.Printf("Loaded ISP \"%v\" (country: %v)\n", isp.Name, isp.CountryCode)
		isps = append(isps, isp)
	}

	// Collect domain IPs
	domainIP := make([]string, 0, len(cfg.Domains))
	for _, domain := range cfg.Domains {
		ip, err := net.ResolveIPAddr("ip4", domain)
		if err != nil {
			log.Fatalln("failed to resolve domain", domain, ":", err)
		}
		domainIP = append(domainIP, ip.String())
	}

	// msmtCh := make(chan Measurement)
	var pending atomic.Int32
	w := csv.NewWriter(os.Stdout)
	w.Write([]string{"domain", "country_code", "isp", "strategy", "attempt", "img", "error_op", "error_msg"})
	for _, isp := range isps {
		for di, domain := range cfg.Domains {
			for _, strategy := range cfg.Strategies {
				transport := isp.Transport
				if transport != "" && strategy != "" {
					transport += "|" + strategy
				}
				dialer, err := transportToDialer.NewStreamDialer(transport)
				if err != nil {
					log.Fatalf("Could not create dialer: %v\n", err)
				}

				numAttempts := 1
				if cfg.NumAttempts != nil {
					numAttempts = *cfg.NumAttempts
				}
				for attempt := 1; attempt <= numAttempts; attempt++ {
					m := Measurement{
						Time:     time.Now(),
						Domain:   domain,
						ISP:      isp.Name,
						Country:  isp.CountryCode,
						Strategy: strategy,
						Attempt:  attempt,
					}
					if strategy == "" {
						m.Strategy = "direct"
					}
					pending.Add(1)
					func(domain string, ip string) {
						ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*timeoutSecFlag)*time.Second)
						defer cancel()
						tcpConn, err := dialer.DialStream(ctx, net.JoinHostPort(ip, "443"))
						if err != nil {
							if err == context.Canceled {
								err = context.Cause(ctx)
							}
							m.ErrorOp = "connect"
							m.ErrorMessage = makeErrorMsg(err, domain)
							return
						}
						tlsConn, err := tls.WrapConn(ctx, tcpConn, domain)
						if err != nil {
							if err == context.Canceled {
								err = context.Cause(ctx)
							}
							m.ErrorOp = "tls"
							m.ErrorMessage = makeErrorMsg(err, domain)
							return
						}
						tlsConn.Close()
						// msmtCh <- m
					}(domain, domainIP[di])
					if m.ErrorOp == "" {
						m.Img = "✅"
					} else {
						m.Img = "❌"
					}
					w.Write([]string{m.Domain, m.Country, m.ISP, m.Strategy, fmt.Sprint(m.Attempt), m.Img, m.ErrorOp, m.ErrorMessage})
					w.Flush()

				}
			}
		}
	}

	// w := csv.NewWriter(os.Stdout)
	// w.Write([]string{"domain", "isp", "strategy", "attempt", "error_op", "error_msg"})
	// for {
	// 	m := <-msmtCh
	// 	w.Write([]string{m.Domain, m.ISP, m.Strategy, fmt.Sprint(m.Attempt), m.ErrorOp, m.ErrorMessage})
	// 	w.Flush()
	// 	if pending.Add(-1) == 0 {
	// 		break
	// 	}
	// }
}
