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

package outline

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Make sure ParseKey returns error for an invalid access key
func TestParseKeyInvalidString(t *testing.T) {
	inputs := []string{
		"",                    // empty string
		" ",                   // blank string
		"\t\n",                // blank string
		"what is this?",       // random string
		"https://example.com", // random https link
	}

	for _, in := range inputs {
		out, err := ParseAccessKey(in)
		require.Error(t, err)
		require.Nil(t, out)
	}
}

// Make sure ParseKey works for a normal Outline access key
func TestParseKeyNormalKey(t *testing.T) {
	cases := []struct {
		input  string
		host   string
		port   int
		prefix []byte
	}{
		{
			// standard access key (chacha encryption)
			input: "ss://Y2hhY2hhMjAtaWV0Zi1wb2x5MTMwNTpteXBhc3M@test.google.com:1234/?outline=1",
			host:  "test.google.com",
			port:  1234,
		},
		{
			// access key with AES encryption
			input: "ss://YWVzLTEyOC1nY206bXlwYXNz@127.0.0.1:4321/?plugin=v2ray-plugin",
			host:  "127.0.0.1",
			port:  4321,
		},
		{
			// access key with tags
			input: "ss://YWVzLTE5Mi1nY206bXlwYXNz@[fe80:0:0:4444:5555:6666:7777:8888]:9999/?outline=1#Test%20Server",
			host:  "fe80:0:0:4444:5555:6666:7777:8888",
			port:  9999,
		},
		{
			// access key with a prefix
			input:  "ss://QUVTLTI1Ni1nY206bXlwYXNz@xxx.www.outline.yyy.zzz:80/?outline=1&prefix=HTTP%2F1.1%20#random-server",
			host:   "xxx.www.outline.yyy.zzz",
			port:   80,
			prefix: []byte("HTTP/1.1 "),
		},
	}

	for _, c := range cases {
		out, err := ParseAccessKey(c.input)
		require.NoError(t, err)
		require.NotNil(t, out)

		require.Exactly(t, c.host, out.Hostname)
		require.Exactly(t, c.port, out.Port)
		require.Equal(t, c.prefix, []byte(out.Prefix))
	}
}
