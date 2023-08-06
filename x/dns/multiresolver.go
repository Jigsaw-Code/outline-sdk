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

package multiresolver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/miekg/dns"
)

// TestDNSOverTCPResolver connects to a DNS resolver over TCP.
func TestDNSOverTCPResolver(ctx context.Context, testDomain string) error {
	client := dns.Client{}
	client.Net = "tcp"
	msg := dns.Msg{}
	msg.SetQuestion(dns.Fqdn(testDomain), dns.TypeA)

	response, _, err := client.Exchange(&msg, "8.8.8.8:53")
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return err
	}

	for _, answer := range response.Answer {
		fmt.Printf("%v\n", answer)
	}

	return nil
}

func resolve(ctx context.Context, testDomain string, use_tcp bool, c chan net.IP) (net.IP, error) {
	tcpClient := dns.Client{}
	if use_tcp {
		tcpClient.Net = "tcp"
	}
	msg := dns.Msg{}
	msg.SetQuestion(dns.Fqdn(testDomain), dns.TypeA)

	response, _, err := tcpClient.Exchange(&msg, "8.8.8.8:53")
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return nil, err
	}

	for _, answer := range response.Answer {
		fmt.Printf("%v\n", answer)
		if a, ok := answer.(*dns.A); ok {
			fmt.Printf("use_tcp:%v A record IP: %s\n", use_tcp, a.A)
			c <- a.A
			return a.A, nil
		}
	}

	return nil, nil
}

// multiResolver attempts resolving a given domain using both TCP and UDP concurrently.
func multiResolver(ctx context.Context, testDomain string) (net.IP, error) {
	var c chan net.IP = make(chan net.IP)
	go resolve(ctx, testDomain, true /* tcp */, c)
	go resolve(ctx, testDomain, false /* udp */, c)
	select {
	case ip := <-c:
		return ip, nil
	case <-time.After(time.Second):
		return nil, errors.New("UDP resolution did not find an A record")
	}
}
