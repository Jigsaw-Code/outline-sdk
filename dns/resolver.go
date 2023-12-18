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

package dns

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"golang.org/x/net/dns/dnsmessage"
)

// Resolver can query the DNS with a question, and obtain a DNS message as response.
// This abstraction helps hide the underlying transport protocol.
type Resolver interface {
	Query(ctx context.Context, q dnsmessage.Question) (*dnsmessage.Message, error)
}

// FuncResolver is a [Resolver] that uses the given function to query DNS.
type FuncResolver func(ctx context.Context, q dnsmessage.Question) (*dnsmessage.Message, error)

// Query implements the [Resolver] interface.
func (f FuncResolver) Query(ctx context.Context, q dnsmessage.Question) (*dnsmessage.Message, error) {
	return f(ctx, q)
}

// NewQuestion is a convenience function to create a [dnsmessage.Question].
func NewQuestion(domain string, qtype dnsmessage.Type) (*dnsmessage.Question, error) {
	name, err := dnsmessage.NewName(domain)
	if err != nil {
		return nil, fmt.Errorf("cannot parse domain name: %w", err)
	}
	return &dnsmessage.Question{
		Name:  name,
		Type:  qtype,
		Class: dnsmessage.ClassINET,
	}, nil
}

// Maximum DNS packet size.
// Value taken from https://dnsflagday.net/2020/.
const maxDNSPacketSize = 1232

// Creates a DNS request using the id and question and appends the bytes to buf.
func appendRequest(id uint16, q dnsmessage.Question, buf []byte) ([]byte, error) {
	b := dnsmessage.NewBuilder(buf, dnsmessage.Header{ID: id, RecursionDesired: true})
	if err := b.StartQuestions(); err != nil {
		return nil, fmt.Errorf("failed to start questions: %w", err)
	}
	if err := b.Question(q); err != nil {
		return nil, fmt.Errorf("failed to add question: %w", err)
	}
	if err := b.StartAdditionals(); err != nil {
		return nil, fmt.Errorf("failed to start additionals: %w", err)
	}

	var rh dnsmessage.ResourceHeader
	// Set the maximum payload size we support, as per https://datatracker.ietf.org/doc/html/rfc6891#section-4.3
	if err := rh.SetEDNS0(maxDNSPacketSize, dnsmessage.RCodeSuccess, false); err != nil {
		return nil, fmt.Errorf("failed to set EDNS(0) parameters: %w", err)
	}
	if err := b.OPTResource(rh, dnsmessage.OPTResource{}); err != nil {
		return nil, fmt.Errorf("failed to add OPT RR: %w", err)
	}

	buf, err := b.Finish()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize message: %w", err)
	}
	return buf, nil
}

// Fold case as clarified in https://datatracker.ietf.org/doc/html/rfc4343#section-3.
func foldCase(char byte) byte {
	if 'a' <= char && char <= 'z' {
		return char - 'a' + 'A'
	}
	return char
}

// equalASCIIName compares DNS name as specified in https://datatracker.ietf.org/doc/html/rfc1035#section-3.1 and
// https://datatracker.ietf.org/doc/html/rfc4343#section-3.
func equalASCIIName(x, y dnsmessage.Name) bool {
	if x.Length != y.Length {
		return false
	}
	for i := 0; i < int(x.Length); i++ {
		if foldCase(x.Data[i]) != foldCase(y.Data[i]) {
			return false
		}
	}
	return true
}

func checkResponse(reqID uint16, reqQues dnsmessage.Question, respHdr dnsmessage.Header, respQs []dnsmessage.Question) error {
	if !respHdr.Response {
		return errors.New("response bit not set")
	}

	// https://datatracker.ietf.org/doc/html/rfc5452#section-4.3
	if reqID != respHdr.ID {
		return fmt.Errorf("message id does not match. Expected %v, got %v", reqID, respHdr.ID)
	}

	// https://datatracker.ietf.org/doc/html/rfc5452#section-4.2
	if len(respQs) == 0 {
		return errors.New("no questions in response")
	}
	respQ := respQs[0]
	if reqQues.Type != respQ.Type || reqQues.Class != respQ.Class || !equalASCIIName(reqQues.Name, respQ.Name) {
		return errors.New("response question doesn't match request")
	}

	return nil
}

const maxMsgSize = 65535

// queryStream implements a DNS query over a stream protocol. It frames the messages by prepending them with a 2-byte length prefix.
func queryStream(conn io.ReadWriter, q dnsmessage.Question) (*dnsmessage.Message, error) {
	id := uint16(rand.Uint32())
	buf, err := appendRequest(id, q, make([]byte, 2, 514))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	if len(buf) > maxMsgSize {
		return nil, fmt.Errorf("message too large: %v bytes", len(buf))
	}
	binary.BigEndian.PutUint16(buf[:2], uint16(len(buf)-2))
	// TODO: Consider writer.ReadFrom(net.Buffers) in case the writer is a TCPConn.
	if _, err := conn.Write(buf); err != nil {
		return nil, fmt.Errorf("failed to write message: %w", err)
	}
	var msgLen uint16
	if err := binary.Read(conn, binary.BigEndian, &msgLen); err != nil {
		return nil, fmt.Errorf("failed to read message length: %w", err)
	}
	if int(msgLen) <= cap(buf) {
		buf = buf[:msgLen]
	} else {
		buf = make([]byte, msgLen)
	}
	if _, err = io.ReadFull(conn, buf); err != nil {
		return nil, fmt.Errorf("failed to read message: %w", err)
	}
	var msg dnsmessage.Message
	if err = msg.Unpack(buf); err != nil {
		return nil, fmt.Errorf("failed to unpack DNS response: %w", err)
	}
	if err := checkResponse(id, q, msg.Header, msg.Questions); err != nil {
		return nil, fmt.Errorf("invalid response: %w", err)
	}
	return &msg, nil
}

// queryDatagram implements a DNS query over a datagram protocol.
func queryDatagram(conn io.ReadWriter, q dnsmessage.Question) (*dnsmessage.Message, error) {
	id := uint16(rand.Uint32())
	buf, err := appendRequest(id, q, make([]byte, 0, 512))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	if len(buf) > maxMsgSize {
		return nil, fmt.Errorf("message too large: %v bytes", len(buf))
	}
	if _, err := conn.Write(buf); err != nil {
		return nil, fmt.Errorf("failed to write message: %w", err)
	}
	if cap(buf) >= maxDNSPacketSize {
		buf = buf[:maxDNSPacketSize]
	} else {
		buf = make([]byte, maxDNSPacketSize)
	}
	for {
		n, err := conn.Read(buf)
		if err != nil {
			return nil, fmt.Errorf("failed to read message: %w", err)
		}
		buf = buf[:n]
		var msg dnsmessage.Message
		if err = msg.Unpack(buf); err != nil {
			return nil, fmt.Errorf("failed to unpack DNS response: %w", err)
		}
		if err := checkResponse(id, q, msg.Header, msg.Questions); err != nil {
			continue
		}
		return &msg, nil
	}
}

// NewTCPResolver creates a [Resolver] that implements the [DNS-over-TCP] protocol, using a [transport.StreamDialer] for transport.
// It creates a new connection to the resolver for every request.
//
// [DNS-over-TCP]: https://datatracker.ietf.org/doc/html/rfc1035#section-4.2.2
func NewTCPResolver(sd transport.StreamDialer, resolverAddr string) Resolver {
	// See https://cs.opensource.google/go/go/+/master:src/net/dnsclient_unix.go;l=127;drc=6146a73d279d73b6138191929d2f1fad22188f51
	// TODO: Consider handling Authenticated Data.
	return FuncResolver(func(ctx context.Context, q dnsmessage.Question) (*dnsmessage.Message, error) {
		conn, err := sd.Dial(ctx, resolverAddr)
		if err != nil {
			return nil, fmt.Errorf("failed to dial resolver: %w", err)
		}
		// TODO: consider keeping the connection open for performance.
		// Need to think about security implications.
		defer conn.Close()
		if deadline, ok := ctx.Deadline(); ok {
			conn.SetDeadline(deadline)
		}
		return queryStream(conn, q)
	})
}

// NewUDPResolver creates a [Resolver] that implements the DNS-over-UDP protocol, using a [transport.PacketDialer] for transport.
// It creates a new connection to the resolver for every request.
//
// [DNS-over-UDP]: https://datatracker.ietf.org/doc/html/rfc1035#section-4.2.1
func NewUDPResolver(pd transport.PacketDialer, resolverAddr string) Resolver {
	// See https://cs.opensource.google/go/go/+/master:src/net/dnsclient_unix.go;l=100;drc=6146a73d279d73b6138191929d2f1fad22188f51
	return FuncResolver(func(ctx context.Context, q dnsmessage.Question) (*dnsmessage.Message, error) {
		conn, err := pd.Dial(ctx, resolverAddr)
		if err != nil {
			return nil, fmt.Errorf("failed to dial resolver: %w", err)
		}
		// TODO: reuse connection, as per https://datatracker.ietf.org/doc/html/rfc7766#section-6.2.1.
		defer conn.Close()
		if deadline, ok := ctx.Deadline(); ok {
			conn.SetDeadline(deadline)
		}
		return queryDatagram(conn, q)
	})
}

// NewTLSResolver creates a [Resolver] that implements the [DNS-over-TLS] protocol, using a [transport.StreamDialer]
// to connect to the resolverAddr the the resolverName as the TLS server name.
// It creates a new connection to the resolver for every request.
//
// [DNS-over-TLS]: https://datatracker.ietf.org/doc/html/rfc7858
func NewTLSResolver(sd transport.StreamDialer, resolverAddr string, resolverName string) Resolver {
	return FuncResolver(func(ctx context.Context, q dnsmessage.Question) (*dnsmessage.Message, error) {
		baseConn, err := sd.Dial(ctx, resolverAddr)
		if err != nil {
			return nil, fmt.Errorf("failed to dial resolver: %w", err)
		}
		tlsConn := tls.Client(baseConn, &tls.Config{
			ServerName: resolverName,
		})
		// TODO: reuse connection, as per https://datatracker.ietf.org/doc/html/rfc7766#section-6.2.1.
		defer tlsConn.Close()
		if deadline, ok := ctx.Deadline(); ok {
			tlsConn.SetDeadline(deadline)
		}
		return queryStream(tlsConn, q)
	})
}

// NewHTTPSResolver creates a [Resolver] that implements the [DNS-over-HTTPS] protocol, using a [transport.StreamDialer]
// to connect to the resolverAddr the url as the DoH template URI.
// It uses an internal HTTP client that reuses connections when possible.
//
// [DNS-over-HTTPS]: https://datatracker.ietf.org/doc/html/rfc8484
func NewHTTPSResolver(sd transport.StreamDialer, resolverAddr string, url string) Resolver {
	dialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
		if !strings.HasPrefix(network, "tcp") {
			// TODO: Support UDP for QUIC.
			return nil, fmt.Errorf("protocol not supported: %v", network)
		}
		return sd.Dial(ctx, resolverAddr)
	}
	// Copied from Intra: https://github.com/Jigsaw-Code/Intra/blob/d3554846a1146ae695e28a8ed6dd07f0cd310c5a/Android/tun2socks/intra/doh/doh.go#L213-L219
	httpClient := http.Client{
		Transport: &http.Transport{
			DialContext:           dialContext,
			ForceAttemptHTTP2:     true,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 20 * time.Second, // Same value as Android DNS-over-TLS
		},
	}
	return FuncResolver(func(ctx context.Context, q dnsmessage.Question) (*dnsmessage.Message, error) {
		buf, err := appendRequest(0, q, make([]byte, 0, 512))
		if err != nil {
			return nil, fmt.Errorf("failed to create DNS request: %w", err)
		}
		httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(buf))
		if err != nil {
			return nil, fmt.Errorf("failed to create HTTP request: %w", err)
		}
		const mimetype = "application/dns-message"
		httpReq.Header.Add("Accept", mimetype)
		httpReq.Header.Add("Content-Type", mimetype)
		httpResp, err := httpClient.Do(httpReq)
		if err != nil {
			return nil, fmt.Errorf("failed to get HTTP response: %w", err)
		}
		defer httpResp.Body.Close()
		if httpResp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("got HTTP status %v", httpResp.StatusCode)
		}
		response, err := io.ReadAll(httpResp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}
		var msg dnsmessage.Message
		if err = msg.Unpack(response); err != nil {
			return nil, fmt.Errorf("failed to unpack DNS response: %w", err)
		}
		if err := checkResponse(0, q, msg.Header, msg.Questions); err != nil {
			return nil, fmt.Errorf("invalid response: %w", err)
		}
		return &msg, nil
	})
}
