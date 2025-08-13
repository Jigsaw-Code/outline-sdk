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
	"log"

	"github.com/Jigsaw-Code/getsni"
)

// sniSplit can be positive or negative
// positive splits forward in the sni, negative splits backward
// 2, example.com -> ex ample.com
// -5, example.com -> exam ple.com
// but must always return a positive index value in the clientHello

func MakeSplitSniFunc(sniSplitOffset int) FragFunc {

	fragFunc := func(clientHello []byte) int {
		sni, err := getsni.GetSNI(clientHello)

		log.Printf("SNI: %v %v\n", sni, err)

		if err != nil || sni == "" {
			return 0
		}

		sniLength := len(sni)

		// Adjust sniSplits that are negative or longer than sniLength to the correct value
		sniSplitOffset = sniSplitOffset % sniLength

		sniIndex := bytes.Index(clientHello, []byte(sni))
		if sniIndex == -1 {
			// This should not happen if parsing was successful and ServerName is not empty.
			// But as a safeguard, don't split.
			return 0
		}

		return sniIndex + sniSplitOffset
	}

	return fragFunc
}
