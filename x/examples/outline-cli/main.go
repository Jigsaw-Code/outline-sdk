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
	"net"
	"os"
	"os/signal"
	"sync"

	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

const OUTLINE_TUN_NAME = "outline233"
const OUTLINE_TUN_IP = "10.233.233.1"
const OUTLINE_TUN_MTU = 1500 // todo: we can read this from netlink
const OUTLINE_GW_SUBNET = "10.233.233.2/32"
const OUTLINE_GW_IP = "10.233.233.2"
const OUTLINE_ROUTING_PRIORITY = 23333
const OUTLINE_ROUTING_TABLE = 233

// ./app -transport "ss://..."
func main() {
	fmt.Println("OutlineVPN CLI (experimental-10161603)")

	transportFlag := flag.String("transport", "", "Transport config")
	flag.Parse()

	bgWait := &sync.WaitGroup{}
	defer bgWait.Wait()

	tun, err := NewTunDevice(OUTLINE_TUN_NAME, OUTLINE_TUN_IP)
	if err != nil {
		fmt.Printf("fatal error: %v\n", err)
		return
	}
	defer tun.Close()

	ss, err := NewOutlineDevice(*transportFlag)
	if err != nil {
		fmt.Printf("fatal error: %v", err)
		return
	}
	defer ss.Close()

	ss.Refresh()

	bgWait.Add(1)
	go func() {
		defer bgWait.Done()
		if err := ss.RelayTraffic(tun); err != nil {
			fmt.Printf("Traffic bridge destroyed: %v\n", err)
		}
	}()

	err = setupRouting()
	if err != nil {
		return
	}
	defer cleanUpRouting()

	svrIpCidr := ss.GetServerIP().String() + "/32"
	r, err := setupIpRule(svrIpCidr)
	if err != nil {
		return
	}
	defer cleanUpRule(r)

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, unix.SIGTERM, unix.SIGHUP)
	s := <-sigc
	fmt.Printf("\nReceived %v, cleaning up resources...\n", s)
}

func setupRouting() error {
	fmt.Println("configuring outline routing table...")
	tun, err := netlink.LinkByName(OUTLINE_TUN_NAME)
	if err != nil {
		fmt.Printf("fatal error: %v\n", err)
		return err
	}

	dst, err := netlink.ParseIPNet(OUTLINE_GW_SUBNET)
	if err != nil {
		fmt.Printf("fatal error: %v\n", err)
		return err
	}
	r := netlink.Route{
		LinkIndex: tun.Attrs().Index,
		Table:     OUTLINE_ROUTING_TABLE,
		Dst:       dst,
		Src:       net.ParseIP(OUTLINE_TUN_IP),
		Scope:     netlink.SCOPE_LINK,
	}
	fmt.Printf("\trouting only from %v to %v through nic %v...\n", r.Src, r.Dst, r.LinkIndex)
	err = netlink.RouteAdd(&r)
	if err != nil {
		fmt.Printf("fatal error: %v\n", err)
		return err
	}

	r = netlink.Route{
		LinkIndex: tun.Attrs().Index,
		Table:     OUTLINE_ROUTING_TABLE,
		Gw:        net.ParseIP(OUTLINE_GW_IP),
	}
	fmt.Printf("\tdefault routing entry via gw %v through nic %v...\n", r.Gw, r.LinkIndex)
	err = netlink.RouteAdd(&r)
	if err != nil {
		fmt.Printf("fatal error: %v\n", err)
		return err
	}

	fmt.Println("routing table has been successfully configured")
	return nil
}

func cleanUpRouting() error {
	fmt.Println("cleaning up outline routing table...")
	filter := netlink.Route{Table: OUTLINE_ROUTING_TABLE}
	routes, err := netlink.RouteListFiltered(netlink.FAMILY_V4, &filter, netlink.RT_FILTER_TABLE)
	if err != nil {
		fmt.Printf("fatal error: %v\n", err)
		return err
	}
	var lastErr error = nil
	for _, route := range routes {
		if err := netlink.RouteDel(&route); err != nil {
			fmt.Printf("fatal error: %v\n", err)
			lastErr = err
		}
	}
	if lastErr == nil {
		fmt.Println("routing table has been reset")
	}
	return lastErr
}

func setupIpRule(svrIp string) (*netlink.Rule, error) {
	fmt.Println("adding ip rule for outline routing table...")
	dst, err := netlink.ParseIPNet(svrIp)
	if err != nil {
		fmt.Printf("fatal error: %v\n", err)
		return nil, err
	}
	rule := netlink.NewRule()
	rule.Priority = OUTLINE_ROUTING_PRIORITY
	rule.Family = netlink.FAMILY_V4
	rule.Table = OUTLINE_ROUTING_TABLE
	rule.Dst = dst
	rule.Invert = true
	fmt.Printf("+from all not to %v via table %v...\n", rule.Dst, rule.Table)
	err = netlink.RuleAdd(rule)
	if err != nil {
		fmt.Printf("fatal error: %v\n", err)
		return nil, err
	}
	fmt.Println("ip rule for outline routing table created")
	return rule, nil
}

func cleanUpRule(rule *netlink.Rule) error {
	fmt.Println("cleaning up ip rule of routing table...")
	err := netlink.RuleDel(rule)
	if err != nil {
		fmt.Printf("fatal error: %v\n", err)
		return err
	}
	fmt.Println("ip rule of routing table deleted")
	return nil
}
