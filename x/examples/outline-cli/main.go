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

//go:build linux

package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"

	"golang.org/x/sys/unix"
)

const OUTLINE_TUN_NAME = "outline233"
const OUTLINE_TUN_IP = "10.233.233.1"
const OUTLINE_TUN_MTU = 1500 // todo: we can read this from netlink
const OUTLINE_GW_SUBNET = "10.233.233.2/32"
const OUTLINE_GW_IP = "10.233.233.2"
const OUTLINE_ROUTING_PRIORITY = 23333
const OUTLINE_ROUTING_TABLE = 233
const OUTLINE_DNS_SERVER = "9.9.9.9"

// ./app -transport "ss://..."
func main() {
	fmt.Println("OutlineVPN CLI (experimental)")

	transportFlag := flag.String("transport", "", "Transport config")
	flag.Parse()

	// this WaitGroup must Wait() after tun is closed
	trafficCopyWg := &sync.WaitGroup{}
	defer trafficCopyWg.Wait()

	tun, err := NewTunDevice(OUTLINE_TUN_NAME, OUTLINE_TUN_IP)
	if err != nil {
		fmt.Printf("[error] failed to create tun device: %v\n", err)
		return
	}
	defer tun.Close()

	// disable IPv6 before resolving Shadowsocks server IP
	prevIPv6, err := enableIPv6(false)
	if err != nil {
		fmt.Printf("[error] failed to disable IPv6: %v\n", err)
		return
	}
	defer enableIPv6(prevIPv6)

	ss, err := NewOutlineDevice(*transportFlag)
	if err != nil {
		fmt.Printf("[error] failed to create Outline device: %v", err)
		return
	}
	defer ss.Close()

	ss.Refresh()

	// Copy the traffic from tun device to OutlineDevice bidirectionally
	trafficCopyWg.Add(2)
	go func() {
		defer trafficCopyWg.Done()
		written, err := io.Copy(ss, tun)
		fmt.Printf("[info] tun -> OutlineDevice stopped: %v %v\n", written, err)
	}()
	go func() {
		defer trafficCopyWg.Done()
		written, err := io.Copy(tun, ss)
		fmt.Printf("[info] OutlineDevice -> tun stopped: %v %v\n", written, err)
	}()

	if err := setSystemDNSServer(OUTLINE_DNS_SERVER); err != nil {
		fmt.Printf("[error] failed to configure system DNS: %v", err)
		return
	}
	defer restoreSystemDNSServer()

	if err := startRouting(ss.GetServerIP().String(),
		OUTLINE_TUN_NAME,
		OUTLINE_GW_SUBNET,
		OUTLINE_TUN_IP,
		OUTLINE_GW_IP,
		OUTLINE_ROUTING_TABLE,
		OUTLINE_ROUTING_PRIORITY); err != nil {
		fmt.Printf("[error] failed to configure routing: %v", err)
		return
	}
	defer stopRouting(OUTLINE_ROUTING_TABLE)

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, unix.SIGTERM, unix.SIGHUP)
	s := <-sigc
	fmt.Printf("\nReceived %v, cleaning up resources...\n", s)
}
