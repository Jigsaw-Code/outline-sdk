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
// TODO move this function into transport/tlsfrag/split_sni.go
func MakeSplitSniFunc(sniSplit int) tlsfrag.FragFunc {
	// takes in an int, and returns a FragFunc which splits the on the sni

	pattern := `\x00\x00\x00\x18\x00\x16\x00`
	re := regexp.MustCompile(pattern)

	fragFunc := func(clientHello []byte) int {
		fmt.Printf("clientHello: %#x\n", clientHello)
		fmt.Printf("sniSplit: %d\n", sniSplit)

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
