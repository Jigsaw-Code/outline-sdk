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
	"fmt"
	"net"
	"strconv"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Jigsaw-Code/outline-sdk/transport/shadowsocks"
)

func NewOutlinePacketListener(accessKey string) (transport.PacketListener, error) {
	config, err := ParseAccessKey(accessKey)
	if err != nil {
		return nil, fmt.Errorf("access key in invalid: %w", err)
	}

	ssAddress := net.JoinHostPort(config.Hostname, strconv.Itoa(config.Port))
	return shadowsocks.NewPacketListener(&transport.UDPEndpoint{Address: ssAddress}, config.CryptoKey)
}
