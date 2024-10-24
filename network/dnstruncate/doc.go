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

/*
Package dnstruncate functions as an alternative implementation that handles DNS requests if the remote server doesn't
support UDP traffic. This is done by always setting the TC (truncated) bit in the DNS response header; it tells the
caller to resend the DNS request using TCP instead of UDP. As a result, no UDP requests are made to the remote server.

This implementation is ported from the [go-tun2socks' dnsfallback.NewUDPHandler].

Note that UDP traffic that are not DNS requests are dropped.

To create a [network.PacketProxy] that handles DNS requests locally:

	proxy, err := dnstruncate.NewPacketProxy()
	if err != nil {
		// handle error
	}

This `proxy` can then be used in, for example, lwip2transport.ConfigureDevice.

[go-tun2socks' dnsfallback.NewUDPHandler]: https://github.com/eycorsican/go-tun2socks/blob/master/proxy/dnsfallback/udp.go
*/
package dnstruncate
