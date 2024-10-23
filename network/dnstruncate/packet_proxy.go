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

package dnstruncate

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"sync/atomic"

	"github.com/Jigsaw-Code/outline-sdk/internal/slicepool"
	"github.com/Jigsaw-Code/outline-sdk/network"
)

// From [RFC 1035], the DNS message header contains the following fields:
//
//		                              1  1  1  1  1  1
//		0  1  2  3  4  5  6  7  8  9  0  1  2  3  4  5
//
//	 +--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
//	 |                      ID                       |
//	 +--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
//	 |QR|   Opcode  |AA|TC|RD|RA|   Z    |   RCODE   |
//	 +--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
//	 |                    QDCOUNT                    |
//	 +--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
//	 |                    ANCOUNT                    |
//	 +--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
//	 |                    NSCOUNT                    |
//	 +--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
//	 |                    ARCOUNT                    |
//	 +--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
//
// [RFC 1035]: https://datatracker.ietf.org/doc/html/rfc1035#section-4.1.1
const (
	standardDNSPort = uint16(53) // https://datatracker.ietf.org/doc/html/rfc1035#section-4.2
	dnsUdpMinMsgLen = 12         // A DNS message must at least contain the header
	dnsUdpMaxMsgLen = 512        // https://datatracker.ietf.org/doc/html/rfc1035#section-2.3.4

	dnsUdpAnswerByte   = 2           // The byte in the header containing QR and TC bit
	dnsUdpResponseBit  = uint8(0x80) // The QR bit within dnsUdpAnswerByte
	dnsUdpTruncatedBit = uint8(0x02) // The TC bit within dnsUdpAnswerByte
	dnsUdpRCodeByte    = 3           // The byte in the header containing RCODE
	dnsUdpRCodeMask    = uint8(0x0f) // The RCODE bits within dnsUdpRCodeByte
	dnsQDCntStartByte  = 4           // The starting byte of QDCOUNT
	dnsQDCntEndByte    = 5           // The ending byte (inclusive) of QDCOUNT
	dnsARCntStartByte  = 6           // The starting byte of ANCOUNT
	dnsARCntEndByte    = 7           // The ending byte (inclusive) of ANCOUNT
)

// packetBufferPool is used to create buffers to modify DNS requests
var packetBufferPool = slicepool.MakePool(dnsUdpMaxMsgLen)

// dnsTruncateProxy is a network.PacketProxy that create dnsTruncateRequestHandler to handle DNS requests locally.
//
// Multiple goroutines may invoke methods on a dnsTruncateProxy simultaneously.
type dnsTruncateProxy struct {
}

// dnsTruncateRequestHandler is a network.PacketRequestSender that handles DNS requests in UDP protocol locally,
// without sending the requests to the actual DNS resolver. It sets the TC (truncated) bit in the DNS response header
// to tell the caller to resend the DNS request over TCP.
//
// Multiple goroutines may invoke methods on a dnsTruncateProxy simultaneously.
type dnsTruncateRequestHandler struct {
	closed     atomic.Bool
	respWriter network.PacketResponseReceiver
}

// Compilation guard against interface implementation
var _ network.PacketProxy = (*dnsTruncateProxy)(nil)
var _ network.PacketRequestSender = (*dnsTruncateRequestHandler)(nil)

// NewPacketProxy creates a new [network.PacketProxy] that can be used to handle DNS requests if the remote proxy
// doesn't support UDP traffic. It sets the TC (truncated) bit in the DNS response header to tell the caller to resend
// the DNS request over TCP.
//
// This [network.PacketProxy] should only be used if the remote proxy server doesn't support UDP traffic at all. Note
// that all other non-DNS UDP packets will be dropped by this [network.PacketProxy].
func NewPacketProxy() (network.PacketProxy, error) {
	return &dnsTruncateProxy{}, nil
}

// NewSession implements [network.PacketProxy].NewSession(). It creates a new [network.PacketRequestSender] that will
// set the TC (truncated) bit and write the response to `respWriter`.
func (p *dnsTruncateProxy) NewSession(respWriter network.PacketResponseReceiver) (network.PacketRequestSender, error) {
	if respWriter == nil {
		return nil, errors.New("respWriter is required")
	}
	return &dnsTruncateRequestHandler{
		respWriter: respWriter,
	}, nil
}

// Close implements [network.PacketRequestSender].Close(), and it closes the corresponding
// [network.PacketResponseReceiver].
func (h *dnsTruncateRequestHandler) Close() error {
	if !h.closed.CompareAndSwap(false, true) {
		return network.ErrClosed
	}
	h.respWriter.Close()
	return nil
}

// WriteTo implements [network.PacketRequestSender].WriteTo(). It parses a packet from p, and determines whether it is
// a valid DNS request. If so, it will write the DNS response with TC (truncated) bit set to the corresponding
// [network.PacketResponseReceiver] passed to NewSession. If it is not a valid DNS request, the packet will be
// discarded and returns an error.
func (h *dnsTruncateRequestHandler) WriteTo(p []byte, destination netip.AddrPort) (int, error) {
	if h.closed.Load() {
		return 0, network.ErrClosed
	}
	if destination.Port() != standardDNSPort {
		return 0, fmt.Errorf("UDP traffic to non-DNS port %v is not supported: %w", destination.Port(), network.ErrPortUnreachable)
	}
	if len(p) < dnsUdpMinMsgLen {
		return 0, fmt.Errorf("invalid DNS message of length %v, it must be at least %v bytes", len(p), dnsUdpMinMsgLen)
	}

	// Allocate buffer from slicepool, because `go build -gcflags="-m"` shows a local array will escape to heap
	slice := packetBufferPool.LazySlice()
	buf := slice.Acquire()
	defer slice.Release()

	// We need to copy p into buf because "WriteTo must not modify p, even temporarily".
	n := copy(buf, p)

	// Set "Response", "Truncated" and "NoError"
	// Note: gopacket is a good library doing this kind of things. But it will increase the binary size a lot.
	//       If we decide to use gopacket in the future, please evaluate the binary size and runtime memory consumption.
	buf[dnsUdpAnswerByte] |= (dnsUdpResponseBit | dnsUdpTruncatedBit)
	buf[dnsUdpRCodeByte] &= ^dnsUdpRCodeMask

	// Copy QDCOUNT to ANCOUNT. This is an incorrect workaround for some DNS clients (such as Windows 7);
	// because without these clients won't retry over TCP.
	//
	// For reference: https://github.com/eycorsican/go-tun2socks/blob/master/proxy/dnsfallback/udp.go#L59-L63
	copy(buf[dnsARCntStartByte:dnsARCntEndByte+1], buf[dnsQDCntStartByte:dnsQDCntEndByte+1])

	return h.respWriter.WriteFrom(buf[:n], net.UDPAddrFromAddrPort(destination))
}
