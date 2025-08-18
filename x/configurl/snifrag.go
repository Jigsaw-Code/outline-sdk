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
	"bytes"
	"context"
	"fmt"
	"strconv"

	"src.agwa.name/tlshacks"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Jigsaw-Code/outline-sdk/transport/tlsfrag"
)

// Writing this here since tlsfrag.MakeSplitSniFunc is not accessible without a release
// this allows direct testing through smart dialer
// DO NOT SUBMIT
// TODO delete this in favor of transport/tlsfrag/split_sni.go

// -------------- COPY ZONE -----------------

func MakeSplitSniFunc(sniSplit int) tlsfrag.FragFunc {

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
