// Copyright 2024 The Outline Authors
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

package configurl

import (
	"context"
	"encoding/binary"
	"fmt"
	"regexp"
	"strconv"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Jigsaw-Code/outline-sdk/transport/tlsfrag"
)

// Writing this here since tlsfrag.MakeSplitSniFunc is not accessible without a release
// this allows direct testing through smart dialer
// DO NOT SUBMIT
// TODO delete this in favor of transport/tlsfrag/split_sni.go

// -------------- COPY ZONE -----------------

// Split SNI implemented FragFunc.

// sniSplit can be positive or negative
// positive splits forward in the sni, negative splits backward
// 2, example.com -> ex ample.com
// -5, example.com -> exam ple.com
// but must always return a positive index value in the clientHello
// if the sniSplit is longer than the length of the SNI then no split happens
// 15, example.com -> example.com

// extract just the SNI extension from a client hello
func getSNIExtension(clientHello []byte) []byte {
	// 6 bytes client hello start
	// 32 bytes randomness
	// 1 byte session id
	// 2 bytes cipher suite length
	// n bytes cipher suites
	// 2 bytes compression info
	// 2 bytes extension length

	cipherSuiteLengthIndex := 6 + 32 + 1
	cipherSuiteLength := int(binary.BigEndian.Uint16(clientHello[cipherSuiteLengthIndex : cipherSuiteLengthIndex+2]))

	fmt.Printf("cipher: %#v %v\n", cipherSuiteLengthIndex, cipherSuiteLength)

	extensionLengthIndex := cipherSuiteLengthIndex + 2 + cipherSuiteLength + 2
	extensionsLength := int(binary.BigEndian.Uint16(clientHello[extensionLengthIndex : extensionLengthIndex+2]))

	fmt.Printf("extensions: %#v %v\n", extensionLengthIndex, extensionsLength)

	extensionContent := clientHello[extensionLengthIndex+2 : extensionLengthIndex+2+extensionsLength]

	fmt.Printf("extensionContent: %#v\n", extensionContent)

	return extensionContent
}

func MakeSplitSniFunc(sniSplit int) tlsfrag.FragFunc {
	// takes in an int, and returns a FragFunc which splits on the SNI

	// 00 00 00 18 00 16 00 [00 0n] ** 00
	// represents the SNI extension + sni length + sni + next message
	// ** (with no 00) represents the domain name
	// https://datatracker.ietf.org/doc/html/rfc6066#section-3
	//sniHeader := []byte{0x00, 0x00, 0x00, 0x18, 0x00, 0x16, 0x00}

	//            a   b   c   d   e   f   g  h  i
	pattern := `\x00\x00\x00\x18\x00\x16\x00`
	// a b = assigned value for server name extension
	// c d = length of following server name extensino
	// e f = length of first (and only) list entry
	// g = entry type DNS hostname
	// h i = length of hostname

	re := regexp.MustCompile(pattern)

	fragFunc := func(clientHello []byte) int {
		fmt.Printf("clientHello: %#x\n", clientHello)
		fmt.Printf("sniSplit: %d\n", sniSplit)

		ext := getSNIExtension(clientHello)

		fmt.Printf("extensionContent: %#v\n", ext)

		isMatch := re.Match(clientHello)
		fmt.Printf("isMatch: %v\n", isMatch)

		if isMatch {
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
		return 0
	}

	return fragFunc
}

// -------------- COPY ZONE -----------------

func registerSNIFragStreamDialer(r TypeRegistry[transport.StreamDialer], typeID string, newSD BuildFunc[transport.StreamDialer]) {
	r.RegisterType(typeID, func(ctx context.Context, config *Config) (transport.StreamDialer, error) {
		sd, err := newSD(ctx, config.BaseConfig)
		if err != nil {
			return nil, err
		}
		lenStr := config.URL.Opaque
		sniSplit, err := strconv.Atoi(lenStr)
		if err != nil {
			return nil, fmt.Errorf("invalid snifrag option: %v. It should be in snifrag:<number> format", lenStr)
		}

		fragFunc := MakeSplitSniFunc(sniSplit)
		return tlsfrag.NewStreamDialerFunc(sd, fragFunc)
	})
}
