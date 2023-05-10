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

package lwip

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/Jigsaw-Code/outline-internal-sdk/transport"
	lwipLib "github.com/eycorsican/go-tun2socks/core"
)

const packetMTU = 1500

// LwIPTun2SocksDevice is an IPDevice that can translate IP packets to TCP/UDP traffic and vice versa. It uses the lwIP
// library to perform the translation.
//
// LwIPTun2SocksDevice must be a singleton object due to limitations in lwIP.
//
// To use a LwIPTun2SocksDevice:
//  1. Call NewTun2SocksDevice with two handlers for TCP and UDP traffic.
//  2. Write IP packets to the device. The device will translate the IP packets to TCP/UDP traffic and send them to the
//     appropriate handlers.
//  3. Read from the device to get the TCP/UDP responses, which will be in IP packet format.
type LwIPTun2SocksDevice struct {
	tcp     *tcpHandler
	udp     *udpHandler
	stack   lwipLib.LWIPStack
	outPipe *packetPipe
}

// NewTun2SocksDevice creates a new LwIPTun2SocksDevice. This device uses a StreamDialer `sd` to handle TCP streams and
// a PacketListener `pl` to handle UDP packets.
//
// You can only have one active LwIPTun2SocksDevice per process. If you try to create more than one, the behavior is
// undefined. However, you can close an existing LwIPTun2SocksDevice and create a new one.
func NewTun2SocksDevice(sd transport.StreamDialer, pl transport.PacketListener) (*LwIPTun2SocksDevice, error) {
	if sd == nil || pl == nil {
		return nil, errors.New("both sd and pl are required")
	}

	t2s := &LwIPTun2SocksDevice{
		tcp:     newTCPHandler(sd),
		udp:     newUDPHandler(pl, 30*time.Second),
		stack:   lwipLib.NewLWIPStack(),
		outPipe: newPacketPipe(packetMTU),
	}
	lwipLib.RegisterTCPConnHandler(t2s.tcp)
	lwipLib.RegisterUDPConnHandler(t2s.udp)
	lwipLib.RegisterOutputFn(t2s.handleIPResponseFromStack)

	return t2s, nil
}

// Close implements io.Closer and network.IPDevice. It closes the device, rendering it unusable for I/O.
//
// Close does not close other objects that are passed to the device, such as StreamDialer, PacketListener, Writer, or
// Reader. You are responsible for closing these objects yourself.
func (t2s *LwIPTun2SocksDevice) Close() error {
	err := t2s.stack.Close()
	t2s.outPipe.close()
	if err != nil {
		return fmt.Errorf("failed to close lwIP stack: %w", err)
	}
	return nil
}

// MTU implements network.IPDevice. It returns the maximum buffer size of a single IP packet that can be processed by
// this device.
func (t2s *LwIPTun2SocksDevice) MTU() int {
	return packetMTU
}

// lwIP responses from TCP/UDP connections (in IP Packets) callback function
func (t2s *LwIPTun2SocksDevice) handleIPResponseFromStack(b []byte) (int, error) {
	return t2s.outPipe.write(b)
}

// Read implements io.Reader and network.IPDevice. It reads one IP packet from the TCP/UDP response, blocking until a
// packet arrives or this device is closed. To prevent potential memory allocations, use the WriteTo function.
func (t2s *LwIPTun2SocksDevice) Read(p []byte) (int, error) {
	return t2s.outPipe.read(p)
}

// WriteTo implements io.WriterTo. It writes all IP packets from TCP/UDP responses to `w` until all data is written or
// an error occurs. This function will not allocate any intermediate buffers.
//
// WriteTo returns the total number of bytes written and any error encountered during the write.
func (t2s *LwIPTun2SocksDevice) WriteTo(w io.Writer) (int64, error) {
	return t2s.outPipe.writeTo(w)
}

// Write implements io.Writer and network.IPDevice. It writes a single IP packet to this device. The device will then
// translate the IP packet into a TCP or UDP traffic.
func (t2s *LwIPTun2SocksDevice) Write(b []byte) (int, error) {
	n, err := t2s.stack.Write(b)
	// workaround: lwip netstack did not use a standard error code
	if err != nil && err.Error() == "stack closed" {
		return n, net.ErrClosed
	}
	return n, err
}

// ReadFrom implements io.ReaderFrom. It reads all IP packets from `r` until EOF or error. EOF can occur if either this
// device is closed or `r` is closed, and it will not be treated as an error. This function will only allocate a single
// buffer of MTU() bytes.
//
// ReadFrom returns the total number of bytes read and any error encountered during the read.
func (t2s *LwIPTun2SocksDevice) ReadFrom(r io.Reader) (int64, error) {
	p := make([]byte, t2s.MTU())
	total := int64(0)
	for {
		n, err := r.Read(p)
		total += int64(n)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, os.ErrClosed) {
				break
			}
			return total, err
		}
		if _, err := t2s.Write(p); err != nil {
			if errors.Is(err, net.ErrClosed) {
				break
			}
			return total, err
		}
	}
	return total, nil
}
