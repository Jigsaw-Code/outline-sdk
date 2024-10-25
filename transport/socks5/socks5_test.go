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

package socks5

import (
	"bytes"
	"io"
	"net/netip"
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
			input:   []byte{addrTypeIPv4, 192, 168, 1, 1, 0x01, 0xF4},
			want:    &address{IP: netip.MustParseAddr("192.168.1.1"), Port: 500},
			wantErr: false,
		},
		{
			name: "IPv6 Full",
			input: []byte{
				addrTypeIPv6,
				0x20, 0x01, 0x0d, 0xb8, // first 4 bytes of the IPv6 address
				0x00, 0x00, 0x00, 0x00, // middle zeroes are often omitted in shorthand notation
				0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x01, // last segment with the "1"
				0x04, 0xD2, // port number 1234
			},
			want:    &address{IP: netip.MustParseAddr("2001:db8::1"), Port: 1234},
			wantErr: false,
		},
		{
			name: "IPv6 Compressed",
			input: []byte{
				addrTypeIPv6,
				0xfe, 0x80, 0x00, 0x00, // first 4 bytes with "fe80", and then three zeroed segments
				0x00, 0x00, 0x00, 0x00,
				0x02, 0x04, 0x61, 0xff, // "0204:61ff"
				0xfe, 0x9d, 0xf1, 0x56, // "fe9d:f156"
				0x00, 0x50, // port number 80 in hexadecimal
			},
			want:    &address{IP: netip.MustParseAddr("fe80::204:61ff:fe9d:f156"), Port: 80},
			wantErr: false,
		},
		{
			name: "IPv6 Loopback",
			input: []byte{
				addrTypeIPv6,
				0x00, 0x00, 0x00, 0x00, // eight zeroed-out segments
				0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x01, // last segment is "0001"
				0x1F, 0x90, // port number 8080 in hexadecimal
			},
			want:    &address{IP: netip.IPv6Loopback(), Port: 8080},
			wantErr: false,
		},
		{
			name: "Domain Short",
			input: []byte{
				addrTypeDomainName,                                    // Address type for domain name
				0x0b,                                                  // Length of the domain name "example.com" which is 11 characters
				'e', 'x', 'a', 'm', 'p', 'l', 'e', '.', 'c', 'o', 'm', // The domain name "example.com"
				0x23, 0x28, // Port number 9000 in hexadecimal
			},
			want:    &address{Name: "example.com", Port: 9000},
			wantErr: false,
		},
		{
			name:    "Domain Long",
			input:   append([]byte{addrTypeDomainName, 0x3B}, append([]byte("very-long-domain-name-used-for-testing-purposes.example.com"), 0x00, 0x50)...),
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

func BenchmarkReadAddr(b *testing.B) {
	tests := []struct {
		name  string
		input []byte
	}{
		{
			name:  "IPv4",
			input: append([]byte{addrTypeIPv4}, append(netip.AddrFrom4([4]byte{192, 168, 1, 1}).AsSlice(), []byte{0x00, 0x50}...)...),
		},
		{
			name:  "IPv6",
			input: append([]byte{addrTypeIPv6}, append(netip.AddrFrom16([16]byte{0x20, 0x01, 0x0d, 0xb8, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}).AsSlice(), []byte{0x1F, 0x90}...)...),
		},
		{
			name:  "Domain",
			input: append([]byte{addrTypeDomainName, 0x0b}, append([]byte("example.com"), []byte{0x23, 0x28}...)...),
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			reader := bytes.NewReader(tt.input)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := reader.Seek(0, io.SeekStart); err != nil {
					b.Error("Seek failed:", err)
				}
				if _, err := readAddr(reader); err != nil {
					b.Error("readAddr failed:", err)
				}
			}
		})
	}
}

func compareAddresses(a1, a2 *address) bool {
	if a1 == nil || a2 == nil {
		return a1 == a2
	}
	if (a1.IP != netip.Addr{}) && a1.IP != a2.IP || a1.Name != a2.Name || a1.Port != a2.Port {
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
