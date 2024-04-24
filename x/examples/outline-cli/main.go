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
	"fmt"
	"io"
	"log"
	"os"
)

var logging = &struct {
	Debug, Info, Warn, Err *log.Logger
}{
	Debug: log.New(io.Discard, "[DEBUG] ", log.LstdFlags),
	Info:  log.New(os.Stdout, "[INFO] ", log.LstdFlags),
	Warn:  log.New(os.Stderr, "[WARN] ", log.LstdFlags),
	Err:   log.New(os.Stderr, "[ERROR] ", log.LstdFlags),
}

// ./app -transport "ss://..."
func main() {
	// Create tun device
	// Run copy loops
	// Assign IP and bring up
	// Set up routing
	fmt.Println("OutlineVPN CLI (experimental)")
	transportFlag := flag.String("transport", "", "Transport config")
	tunFlag := flag.String("tun", "outline233", "Name of the TUN device")
	dnsFlag := flag.String("dns", "9.9.9.9", "DNS server to use")
	flag.Parse()

	app := App{
		TransportConfig: *transportFlag,
		RoutingConfig: &RoutingConfig{
			TunDeviceName:        *tunFlag,
			TunDeviceIP:          "10.233.233.1",
			TunDeviceMTU:         1500, // todo: read this from netlink
			TunGatewayCIDR:       "10.233.233.2/32",
			RoutingTableID:       233,
			RoutingTablePriority: 23333,
			DNSServerIP:          *dnsFlag,
		},
	}

	tun, err := newTunDevice(*tunFlag, "10.233.233.1")
	if err != nil {
		return fmt.Errorf("failed to create tun device: %w", err)
	}
	defer tun.Close()

	if err := app.Run(); err != nil {
		logging.Err.Printf("%v\n", err)
	}
}
