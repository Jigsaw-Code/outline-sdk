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
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAppendSOCKS5Address_IPv4(t *testing.T) {
	var b bytes.Buffer
	err := writeSOCKS5Address(&b, "8.8.8.8:853")
	require.NoError(t, err)
	// 853 = 0x355
	require.EqualValues(t, []byte{1, 8, 8, 8, 8, 0x3, 0x55}, b.Bytes())
}

func TestAppendSOCKS5Address_IPv6(t *testing.T) {
	var b bytes.Buffer
	err := writeSOCKS5Address(&b, "[2001:4860:4860::8888]:853")
	require.NoError(t, err)
	require.EqualValues(t, []byte{0x04, 0x20, 0x01, 0x48, 0x60, 0x48, 0x60, 0, 0, 0, 0, 0, 0, 0, 0, 0x88, 0x88, 0x3, 0x55}, b.Bytes())
}

func TestAppendSOCKS5Address_DomainName(t *testing.T) {
	var b bytes.Buffer
	err := writeSOCKS5Address(&b, "dns.google:853")
	require.NoError(t, err)
	require.EqualValues(t, []byte{0x03, byte(len("dns.google")), 'd', 'n', 's', '.', 'g', 'o', 'o', 'g', 'l', 'e', 0x3, 0x55}, b.Bytes())
}

func TestAppendSOCKS5Address_NotHostPort(t *testing.T) {
	err := writeSOCKS5Address(&bytes.Buffer{}, "fsdfksajdhfjk")
	require.Error(t, err)
}

func TestAppendSOCKS5Address_BadPort(t *testing.T) {
	err := writeSOCKS5Address(&bytes.Buffer{}, "dns.google:dns")
	require.Error(t, err)
}

func TestAppendSOCKS5Address_DomainNameTooLong(t *testing.T) {
	err := writeSOCKS5Address(&bytes.Buffer{}, strings.Repeat("1234567890", 26)+":53")
	require.Error(t, err)
}
