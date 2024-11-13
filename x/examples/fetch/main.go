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
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/textproto"
	"os"
	"path"
	"strings"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/x/configurl"
	"github.com/lmittmann/tint"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"golang.org/x/term"
)

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

func overrideAddress(original string, newHost string, newPort string) (string, error) {
	host, port, err := net.SplitHostPort(original)
	if err != nil {
		return "", fmt.Errorf("invalid address: %w", err)
	}
	if newHost != "" {
		host = newHost
	}
	if newPort != "" {
		port = newPort
	}
	return net.JoinHostPort(host, port), nil
}

func main() {
	verboseFlag := flag.Bool("v", false, "Enable debug output")
	tlsKeyLogFlag := flag.String("tls-key-log", "", "Filename to write the TLS key log to allow for decryption on Wireshark")
	protoFlag := flag.String("proto", "h1", "HTTP version to use (h1, h2, h3)")
	transportFlag := flag.String("transport", "", "Transport config")
	addressFlag := flag.String("address", "", "Address to connect to. If empty, use the URL authority")
	methodFlag := flag.String("method", "GET", "The HTTP method to use")
	var headersFlag stringArrayFlagValue
	flag.Var(&headersFlag, "H", "Raw HTTP Header line to add. It must not end in \\r\\n")
	timeoutSecFlag := flag.Int("timeout", 5, "Timeout in seconds")

	flag.Parse()

	logLevel := slog.LevelInfo
	if *verboseFlag {
		logLevel = slog.LevelDebug
	}
	slog.SetDefault(slog.New(tint.NewHandler(
		os.Stderr,
		&tint.Options{NoColor: !term.IsTerminal(int(os.Stderr.Fd())), Level: logLevel},
	)))

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
		slog.Error("Need to pass the URL to fetch in the command-line")
		flag.Usage()
		os.Exit(1)
	}

	httpClient := &http.Client{
		Timeout: time.Duration(*timeoutSecFlag) * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	defer httpClient.CloseIdleConnections()

	var tlsConfig tls.Config
	if *tlsKeyLogFlag != "" {
		f, err := os.Create(*tlsKeyLogFlag)
		if err != nil {
			slog.Error("Failed to creare TLS key log file", "error", err)
			os.Exit(1)
		}
		defer f.Close()
		tlsConfig.KeyLogWriter = f
	}
	providers := configurl.NewDefaultProviders()
	if *protoFlag == "h1" || *protoFlag == "h2" {
		dialer, err := providers.NewStreamDialer(context.Background(), *transportFlag)
		if err != nil {
			slog.Error("Could not create dialer", "error", err)
			os.Exit(1)
		}
		dialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
			addressToDial, err := overrideAddress(addr, overrideHost, overridePort)
			if err != nil {
				return nil, fmt.Errorf("invalid address: %w", err)
			}
			if !strings.HasPrefix(network, "tcp") {
				return nil, fmt.Errorf("protocol not supported: %v", network)
			}
			return dialer.DialStream(ctx, addressToDial)
		}
		if *protoFlag == "h1" {
			tlsConfig.NextProtos = []string{"http/1.1"}
			httpClient.Transport = &http.Transport{
				DialContext:     dialContext,
				TLSClientConfig: &tlsConfig,
			}
		} else if *protoFlag == "h2" {
			tlsConfig.NextProtos = []string{"h2"}
			httpClient.Transport = &http.Transport{
				DialContext:       dialContext,
				TLSClientConfig:   &tlsConfig,
				ForceAttemptHTTP2: true,
			}
		}
	} else if *protoFlag == "h3" {
		listener, err := providers.NewPacketListener(context.Background(), *transportFlag)
		if err != nil {
			slog.Error("Could not create listener", "error", err)
			os.Exit(1)
		}
		conn, err := listener.ListenPacket(context.Background())
		if err != nil {
			slog.Error("Could not create PacketConn", "error", err)
			os.Exit(1)
		}
		quicTransport := &quic.Transport{
			Conn: conn,
		}
		defer quicTransport.Close()
		httpTransport := &http3.Transport{
			TLSClientConfig: &tlsConfig,
			Dial: func(ctx context.Context, addr string, tlsConf *tls.Config, quicConf *quic.Config) (quic.EarlyConnection, error) {
				addressToDial, err := overrideAddress(addr, overrideHost, overridePort)
				if err != nil {
					return nil, fmt.Errorf("invalid address: %w", err)
				}
				udpAddr, err := net.ResolveUDPAddr("udp", addressToDial)
				if err != nil {
					return nil, err
				}
				return quicTransport.DialEarly(ctx, udpAddr, tlsConf, quicConf)
			},
			Logger: slog.Default(),
		}
		defer httpTransport.Close()
		httpClient.Transport = httpTransport
	} else {
		slog.Error("Invalid HTTP protocol", "proto", *protoFlag)
		os.Exit(1)
	}

	req, err := http.NewRequest(*methodFlag, url, nil)
	if err != nil {
		slog.Error("Failed to create request", "error", err)
		os.Exit(1)
	}
	headerText := strings.Join(headersFlag, "\r\n") + "\r\n\r\n"
	h, err := textproto.NewReader(bufio.NewReader(strings.NewReader(headerText))).ReadMIMEHeader()
	if err != nil {
		slog.Error("Invalid header line", "error", err)
		os.Exit(1)
	}
	for name, values := range h {
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		slog.Error("HTTP request failed", "error", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if *verboseFlag {
		slog.Info("HTTP Proto", "version", resp.Proto)
		slog.Info("HTTP Status", "status", resp.Status)
		for k, v := range resp.Header {
			slog.Debug("Header", "key", k, "value", v)
		}
	}

	_, err = io.Copy(os.Stdout, resp.Body)
	fmt.Println()
	if err != nil {
		slog.Error("Read of page body failed", "error", err)
		os.Exit(1)
	}
}
