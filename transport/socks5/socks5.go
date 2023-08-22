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

package socks5

import (
	"errors"
	"fmt"
	"net"
	"strconv"
)

const addrTypeIPv4 = 0x01
const addrTypeDomainName = 0x03
const addrTypeIPv6 = 0x04

func appendSOCKS5Address(b []byte, address string) ([]byte, error) {
	host, portStr, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	portNum, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, err
	}
	if ip := net.ParseIP(host); ip != nil {
		if ip4 := ip.To4(); ip4 != nil {
			b = append(b, addrTypeIPv4)
			b = append(b, ip4...)
		} else if ip6 := ip.To16(); ip6 != nil {
			b = append(b, addrTypeIPv6)
			b = append(b, ip6...)
		} else {
			// This should never happen.
			return nil, errors.New("IP address not IPv4 or IPv6")
		}
	} else {
		if len(host) > 255 {
			return nil, fmt.Errorf("domain name length = %v is over 255", len(host))
		}
		b = append(b, addrTypeDomainName)
		b = append(b, byte(len(host)))
		b = append(b, host...)
	}
	b = append(b, byte(portNum>>8), byte(portNum))
	return b, nil
}
