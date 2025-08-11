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

package tlsfrag

import (
	"bytes"

	"src.agwa.name/tlshacks"
)

// sniSplit can be positive or negative
// positive splits forward in the sni, negative splits backward
// 2, example.com -> ex ample.com
// -5, example.com -> exam ple.com
// but must always return a positive index value in the clientHello

func MakeSplitSniFunc(sniSplitOffset int) FragFunc {

	fragFunc := func(clientHello []byte) int {
		hello := tlshacks.UnmarshalClientHello(clientHello)
		// Failed parse
		if hello == nil {
			return 0
		}

		var serverName string
		// Find the Server Name Indication extension (type 0)
		for _, ext := range hello.Extensions {
			if ext.Type == 0 { // 0 is the type for the ServerNameData extension
				if sni, ok := ext.Data.(*tlshacks.ServerNameData); ok {
					if len(sni.HostName) > 0 {
						// We only care about the first hostname.
						serverName = sni.HostName
					}
				}
				// We found the SNI extension, so we can stop searching.
				break
			}
		}

		if serverName == "" {
			// No SNI, don't split.
			return 0
		}
		sniLength := len(serverName)

		// Adjust sniSplits that are negative or longer than sniLength to the correct value
		sniSplitOffset = sniSplitOffset % sniLength

		sniIndex := bytes.Index(clientHello, []byte(serverName))
		if sniIndex == -1 {
			// This should not happen if parsing was successful and ServerName is not empty.
			// But as a safeguard, don't split.
			return 0
		}

		return sniIndex + sniSplitOffset
	}

	return fragFunc
}
