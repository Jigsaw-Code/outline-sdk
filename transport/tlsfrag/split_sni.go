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
	"encoding/binary"
	"fmt"
	"regexp"
)

// Split SNI implemented FragFunc.

// sniSplit can be positive or negative
// positive splits forward in the sni, negative splits backward
// 2, example.com -> ex ample.com
// -5, example.com -> exam ple.com
// but must always return a positive index value in the clientHello
// if the sniSplit is longer than the length of the SNI then no split happens
// 15, example.com -> example.com

func MakeSplitSniFunc(sniSplit int) FragFunc {
	// takes in an int, and returns a FragFunc which splits on the SNI

	// 00 00 00 18 00 16 00 [00 0n] ** 00
	// represents the SNI extension + sni length + sni + next message
	// ** (with no 00) represents the domain name
	// https://datatracker.ietf.org/doc/html/rfc6066#section-3
	//sniHeader := []byte{0x00, 0x00, 0x00, 0x18, 0x00, 0x16, 0x00}

	pattern := `\x00\x00\x00\x18\x00\x16\x00`
	re := regexp.MustCompile(pattern)

	fragFunc := func(clientHello []byte) int {
		fmt.Printf("clientHello: %#x\n", clientHello)
		fmt.Printf("sniSplit: %d\n", sniSplit)

		isMatch := re.Match(clientHello)

		fmt.Printf("isMatch: %v\n", isMatch)

		sniExtensionIndex := re.FindIndex(clientHello)[0]
		sniLengthBytes := clientHello[sniExtensionIndex+7 : sniExtensionIndex+9]
		sniLength := int(binary.BigEndian.Uint16(sniLengthBytes))
		sniStartIndex := sniExtensionIndex + 9

		fmt.Printf("sniLength: %v\n", sniLength)
		fmt.Printf("sniStartIndex: %v\n", sniStartIndex)

		splitIndex := sniStartIndex + (sniSplit % sniLength)

		fmt.Printf("splitIndex: %v\n", splitIndex)

		return splitIndex
	}

	return fragFunc
}
