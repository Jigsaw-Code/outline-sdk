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
	"bytes"
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadAddr(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		want    *address
		wantErr bool
	}{

		{
			name:    "IPv4 Example",
			input:   append([]byte{addrTypeIPv4}, append(net.IPv4(192, 168, 1, 1).To4(), []byte{0x01, 0xF4}...)...),
			want:    &address{IP: net.IPv4(192, 168, 1, 1), Port: 500},
			wantErr: false,
		},
		{
			name:    "IPv6 Full",
			input:   append([]byte{addrTypeIPv6}, append(net.ParseIP("2001:db8::1").To16(), []byte{0x04, 0xD2}...)...),
			want:    &address{IP: net.ParseIP("2001:db8::1"), Port: 1234},
			wantErr: false,
		},
		{
			name:    "IPv6 Compressed",
			input:   append([]byte{addrTypeIPv6}, append(net.ParseIP("fe80::204:61ff:fe9d:f156").To16(), []byte{0x00, 0x50}...)...),
			want:    &address{IP: net.ParseIP("fe80::204:61ff:fe9d:f156"), Port: 80},
			wantErr: false,
		},
		{
			name:    "IPv6 Loopback",
			input:   append([]byte{addrTypeIPv6}, append(net.IPv6loopback.To16(), []byte{0x1F, 0x90}...)...),
			want:    &address{IP: net.IPv6loopback, Port: 8080},
			wantErr: false,
		},
		{
			name:    "Domain Short",
			input:   append([]byte{addrTypeDomainName, 0x0b}, append([]byte("example.com"), []byte{0x23, 0x28}...)...),
			want:    &address{Name: "example.com", Port: 9000},
			wantErr: false,
		},
		{
			name:    "Domain Long",
			input:   append([]byte{addrTypeDomainName, 0x3B}, append([]byte("very-long-domain-name-used-for-testing-purposes.example.com"), []byte{0x00, 0x50}...)...),
			want:    &address{Name: "very-long-domain-name-used-for-testing-purposes.example.com", Port: 80},
			wantErr: false,
		},
		{
			name:    "Unrecognized Address Type",
			input:   []byte{0x00},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "Short Input",
			input:   []byte{addrTypeIPv4},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader(tt.input)
			got, err := readAddr(r)
			if tt.wantErr {
				require.Error(t, err, "Expected an error but got none")
			} else {
				require.NoError(t, err, "Did not expect an error but got one")
			}

			if !tt.wantErr && !compareAddresses(got, tt.want) {
				t.Errorf("readAddr() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func compareAddresses(a1, a2 *address) bool {
	if a1 == nil || a2 == nil {
		return a1 == a2
	}
	if a1.IP != nil && !a1.IP.Equal(a2.IP) || a1.Name != a2.Name || a1.Port != a2.Port {
		return false
	}
	return true
}

func TestAppendSOCKS5Address_IPv4(t *testing.T) {
	b := []byte{}
	b, err := appendSOCKS5Address(b, "8.8.8.8:853")
	require.NoError(t, err)
	// 853 = 0x355
	require.EqualValues(t, []byte{1, 8, 8, 8, 8, 0x3, 0x55}, b)
}

func TestAppendSOCKS5Address_IPv6(t *testing.T) {
	b := []byte{}
	b, err := appendSOCKS5Address(b, "[2001:4860:4860::8888]:853")
	require.NoError(t, err)
	require.EqualValues(t, []byte{0x04, 0x20, 0x01, 0x48, 0x60, 0x48, 0x60, 0, 0, 0, 0, 0, 0, 0, 0, 0x88, 0x88, 0x3, 0x55}, b)
}

func TestAppendSOCKS5Address_DomainName(t *testing.T) {
	b := []byte{}
	b, err := appendSOCKS5Address(b, "dns.google:853")
	require.NoError(t, err)
	require.EqualValues(t, []byte{0x03, byte(len("dns.google")), 'd', 'n', 's', '.', 'g', 'o', 'o', 'g', 'l', 'e', 0x3, 0x55}, b)
}

func TestAppendSOCKS5Address_NotHostPort(t *testing.T) {
	_, err := appendSOCKS5Address([]byte{}, "fsdfksajdhfjk")
	require.Error(t, err)
}

func TestAppendSOCKS5Address_BadPort(t *testing.T) {
	_, err := appendSOCKS5Address([]byte{}, "dns.google:dns")
	require.Error(t, err)
}

func TestAppendSOCKS5Address_DomainNameTooLong(t *testing.T) {
	_, err := appendSOCKS5Address([]byte{}, strings.Repeat("1234567890", 26)+":53")
	require.Error(t, err)
}
