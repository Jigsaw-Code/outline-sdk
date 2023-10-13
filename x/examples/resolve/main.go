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
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path"
	"strings"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Jigsaw-Code/outline-sdk/x/config"
)

var debugLog log.Logger = *log.New(io.Discard, "", 0)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [flags...] <domain>\n", path.Base(os.Args[0]))
		flag.PrintDefaults()
	}
}

func cleanDNSError(err error, resolverAddr string) error {
	dnsErr := &net.DNSError{}
	if resolverAddr != "" && errors.As(err, &dnsErr) {
		dnsErr.Server = resolverAddr
		return dnsErr
	}
	return err
}

func main() {
	verboseFlag := flag.Bool("v", false, "Enable debug output")
	typeFlag := flag.String("type", "A", "The type of the query (A, AAAA, CNAME, NS or TXT).")
	resolverFlag := flag.String("resolver", "", "The address of the recursive DNS resolver to use in host:port format. If the port is missing, it's assumed to be 53")
	transportFlag := flag.String("transport", "", "The transport for the connection to the recursive DNS resolver")
	tcpFlag := flag.Bool("tcp", false, "Force TCP when querying the DNS resolver")

	flag.Parse()
	if *verboseFlag {
		debugLog = *log.New(os.Stderr, "[DEBUG] ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)
	}

	domain := strings.TrimSpace(flag.Arg(0))
	if domain == "" {
		log.Fatal("Need to pass the domain to resolve in the command-line")
	}

	resolverAddr := *resolverFlag
	if resolverAddr != "" && !strings.Contains(resolverAddr, ":") {
		resolverAddr = net.JoinHostPort(resolverAddr, "53")
	}

	var err error
	var packetDialer transport.PacketDialer
	if !*tcpFlag {
		packetDialer, err = config.NewPacketDialer(*transportFlag)
		if err != nil {
			log.Fatalf("Could not create packet dialer: %v", err)
		}
	}
	streamDialer, err := config.NewStreamDialer(*transportFlag)
	if err != nil {
		log.Fatalf("Could not create stream dialer: %v", err)
	}

	resolver := net.Resolver{PreferGo: true}
	resolver.Dial = func(ctx context.Context, network, sysResolverAddr string) (net.Conn, error) {
		dialAddr := sysResolverAddr
		if resolverAddr != "" {
			dialAddr = resolverAddr
		}
		if strings.HasPrefix(network, "tcp") || *tcpFlag {
			debugLog.Printf("Dial TCP: %v", dialAddr)
			return streamDialer.Dial(ctx, dialAddr)
		}
		debugLog.Printf("Dial UDP: %v", dialAddr)
		return packetDialer.Dial(ctx, dialAddr)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	switch strings.ToUpper(*typeFlag) {
	case "A":
		ips, err := resolver.LookupIP(ctx, "ip4", domain)
		err = cleanDNSError(err, resolverAddr)
		if err != nil {
			log.Fatalf("Failed to lookup IPs: %v", err)
		}
		for _, ip := range ips {
			fmt.Println(ip.String())
		}
	case "AAAA":
		ips, err := resolver.LookupIP(ctx, "ip6", domain)
		err = cleanDNSError(err, resolverAddr)
		if err != nil {
			log.Fatalf("Failed to lookup IPs: %v", err)
		}
		for _, ip := range ips {
			fmt.Println(ip.String())
		}
	case "CNAME":
		cname, err := resolver.LookupCNAME(ctx, domain)
		err = cleanDNSError(err, resolverAddr)
		if err != nil {
			log.Fatalf("Failed to lookup CNAME: %v", err)
		}
		fmt.Println(cname)
	case "NS":
		nss, err := resolver.LookupNS(ctx, domain)
		err = cleanDNSError(err, resolverAddr)
		if err != nil {
			log.Fatalf("Failed to lookup NS: %v", err)
		}
		for _, ns := range nss {
			fmt.Println(ns.Host)
		}
	case "TXT":
		lines, err := resolver.LookupTXT(ctx, domain)
		err = cleanDNSError(err, resolverAddr)
		if err != nil {
			log.Fatalf("Failed to lookup NS: %v", err)
		}
		for _, line := range lines {
			fmt.Println(line)
		}
	default:
		log.Fatalf("Invalid query type %v", *typeFlag)
	}
}
