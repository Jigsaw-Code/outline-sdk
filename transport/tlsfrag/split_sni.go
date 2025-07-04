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

import "fmt"

// Split SNI implemented FragFunc.

// splitSNI can be positive or negative
// positive splits forward in the sni, negative splits backward
// 2, example.com -> ex ample.com
// -5, example.com -> exam ple.com
// but must always return a positive index value in the payload

// 00 00 00 18 00 16 00 00 13 ** 00
// represents the SNI extension + sni + next message
// ** (with no 00) represents the domain name

// https://datatracker.ietf.org/doc/html/rfc6066#section-3

func MakeSplitSniFunc(sniSplit int) FragFunc {
	// takes in an int, and returns a FragFunc which splits on the sni

	fragFunc := func(payload []byte) int {
		fmt.Printf("clientHello: %#x\n", payload)
		fmt.Printf("sniSplit: %d\n", sniSplit)
		return sniSplit
	}

	return fragFunc
}
