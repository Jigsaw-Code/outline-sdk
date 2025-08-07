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
	"encoding/binary"
	"fmt"
	"regexp"

	"src.agwa.name/tlshacks"
)

// Split SNI implemented FragFunc.

// sniSplit can be positive or negative
// positive splits forward in the sni, negative splits backward
// 2, example.com -> ex ample.com
// -5, example.com -> exam ple.com
// but must always return a positive index value in the clientHello
// if the sniSplit is longer than the length of the SNI then no split happens
// 15, example.com -> example.com

// extract just the SNI extension from a client hello
func getSNIExtension(clientHello []byte) ([]byte, error) {
	// 6 bytes client hello start
	// 32 bytes randomness
	// 1 byte session id
	// 2 bytes cipher suite length
	// n bytes cipher suites
	// 2 bytes compression info
	// 2 bytes extension length

	fmt.Printf("clientHello: %#x\n", clientHello)

	helloTypeLength := 1
	helloLengthLength := 3
	tlsVersionLength := 2
	clientRandomLength := 32

	sessionDataIndex := helloTypeLength + helloLengthLength + tlsVersionLength + clientRandomLength

	sessionDataLength := int(clientHello[sessionDataIndex])

	fmt.Printf("session: %#v %v, %#x\n", sessionDataIndex, sessionDataLength, clientHello[sessionDataIndex-1:sessionDataIndex])

	cipherSuiteLengthIndex := sessionDataIndex + 1 + sessionDataLength
	cipherSuiteLength := int(binary.BigEndian.Uint16(clientHello[cipherSuiteLengthIndex : cipherSuiteLengthIndex+2]))

	fmt.Printf("cipher: %#v %v, %#x\n", cipherSuiteLengthIndex, cipherSuiteLength, clientHello[cipherSuiteLengthIndex:cipherSuiteLengthIndex+2])

	extensionLengthIndex := cipherSuiteLengthIndex + 2 + cipherSuiteLength + 2
	extensionsLength := int(binary.BigEndian.Uint16(clientHello[extensionLengthIndex : extensionLengthIndex+2]))

	fmt.Printf("extensions: %#v %v\n", extensionLengthIndex, extensionsLength)

	allExtensionContent := clientHello[extensionLengthIndex+2 : extensionLengthIndex+2+extensionsLength]

	fmt.Printf("extensionContent: %#v\n", allExtensionContent)

	//firstExtIdentifier = allExtensionContent[0:2]
	//if firstExtIdentifier != []byte{0x00, 0x00} {
	//	return nil, Error("no SNI extension found in client hello")
	//)

	// sniExtLength

	// sniExtension =

	//return extensionContent, nil
	return nil, nil
}

// The regex for a potential domain name.
const domainRegexPattern = `(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,6}|xn--[a-z0-9-]+(?:\.[a-z0-9-]+)*`

// Pre-compile the regex once at the package level.
var domainRegex = regexp.MustCompile(domainRegexPattern)

// findFirstDomainIndex takes a byte slice and returns the starting index of the
// first string that matches the domain regex. If no match is found, it returns -1.
func findFirstDomainIndex(data []byte) int {
	// Use the pre-compiled regex directly.
	matchIndexes := domainRegex.FindStringIndex(string(data))
	if matchIndexes == nil {
		return -1
	}

	return matchIndexes[0]
}

func OldMakeSplitSniFunc(sniSplit int) FragFunc {
	// takes in an int, and returns a FragFunc which splits on the SNI

	// 00 00 00 18 00 16 00 [00 0n] ** 00
	// represents the SNI extension + sni length + sni + next message
	// ** (with no 00) represents the domain name
	// https://datatracker.ietf.org/doc/html/rfc6066#section-3
	//sniHeader := []byte{0x00, 0x00, 0x00, 0x18, 0x00, 0x16, 0x00}

	//            a   b   c   d   e   f   g   h   i
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

		ext, _ := getSNIExtension(clientHello)

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

func MakeSplitSniFunc(sniSplit int) FragFunc {

	fragFunc := func(clientHello []byte) int {
		hello := tlshacks.UnmarshalClientHello(clientHello)
		// Failed parse
		if hello == nil {
			return 0
		}

		var serverName string
		// Find the Server Name Indication extension (type 0)
		for _, ext := range hello.Extensions {
			if ext.Type == 0 { // 0 is the type for server_name extension
				// The content of the SNI extension is a ServerNameList.
				// See RFC 6066, Section 3.
				if len(ext.Data) < 2 {
					break // Malformed extension, cannot parse.
				}
				// First 2 bytes: length of the server_name_list.
				listLen := int(binary.BigEndian.Uint16(ext.Data)[0:2])
				if listLen != len(ext.Data)-2 {
					break // Malformed extension.
				}

				serverNameList := ext.Data[2:]
				// We only care about the first name in the list.
				if len(serverNameList) < 3 {
					break // Malformed list.
				}
				nameType := serverNameList[0]
				nameLen := int(binary.BigEndian.Uint16(serverNameList[1:3]))
				if nameLen > len(serverNameList)-3 {
					break // Malformed name entry.
				}
				if nameType == 0 { // 0 is for host_name
					serverName = string(serverNameList[3 : 3+nameLen])
				}
				// We found the SNI extension, so we can stop searching.
				break
			}
		}

		if serverName == "" {
			// No SNI, don't split.
			return 0
		}

		sniIndex := bytes.Index(clientHello, []byte(serverName))
		if sniIndex == -1 {
			// This should not happen if parsing was successful and ServerName is not empty.
			// But as a safeguard, don't split.
			return 0
		}

		sniLength := len(serverName)
		splitOffset := sniSplit
		if splitOffset < 0 {
			// Handle negative split values, which count from the end of the SNI.
			splitOffset = sniLength + splitOffset
		}

		if splitOffset <= 0 || splitOffset >= sniLength {
			// Invalid split point (outside the SNI), don't split.
			return 0
		}

		return sniIndex + splitOffset
	}

	return fragFunc
}
