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

package dns

import (
	"bytes"
	"context"
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
	"github.com/Jigsaw-Code/outline-sdk/transport/tls"
	"golang.org/x/net/dns/dnsmessage"
)

var (
	ErrBadRequest  = errors.New("request input is invalid")
	ErrDial        = errors.New("dial DNS resolver failed")
	ErrSend        = errors.New("send DNS message failed")
	ErrReceive     = errors.New("receive DNS message failed")
	ErrBadResponse = errors.New("response message is invalid")
)

// nestedError allows us to use errors.Is and still preserve the error cause.
// This is unlike fmt.Errorf, which creates a new error and preserves the cause,
// but you can't specify the type of the resulting top-level error.
type nestedError struct {
	is      error
	wrapped error
}

func (e *nestedError) Is(target error) bool { return target == e.is }

func (e *nestedError) Unwrap() error { return e.wrapped }

func (e *nestedError) Error() string { return e.is.Error() + ": " + e.wrapped.Error() }

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
// The input domain is interpreted as fully-qualified. If the end "." is missing, it's added.
func NewQuestion(domain string, qtype dnsmessage.Type) (*dnsmessage.Question, error) {
	fullDomain := domain
	if len(domain) == 0 || domain[len(domain)-1] != '.' {
		fullDomain += "."
	}
	name, err := dnsmessage.NewName(fullDomain)
	if err != nil {
		return nil, fmt.Errorf("cannot parse domain name: %w", err)
	}
	return &dnsmessage.Question{
		Name:  name,
		Type:  qtype,
		Class: dnsmessage.ClassINET,
	}, nil
}

// Maximum UDP message size that we support.
// The value is taken from https://dnsflagday.net/2020/, which says:
// "An EDNS buffer size of 1232 bytes will avoid fragmentation on nearly all current networks.
// This is based on an MTU of 1280, which is required by the IPv6 specification, minus 48 bytes
// for the IPv6 and UDP headers".
const maxUDPMessageSize = 1232

// appendRequest appends the bytes a DNS request using the id and question to buf.
func appendRequest(id uint16, q dnsmessage.Question, buf []byte) ([]byte, error) {
	b := dnsmessage.NewBuilder(buf, dnsmessage.Header{ID: id, RecursionDesired: true})
	if err := b.StartQuestions(); err != nil {
		return nil, fmt.Errorf("start questions failed: %w", err)
	}
	if err := b.Question(q); err != nil {
		return nil, fmt.Errorf("add question failed: %w", err)
	}
	if err := b.StartAdditionals(); err != nil {
		return nil, fmt.Errorf("start additionals failed: %w", err)
	}

	var rh dnsmessage.ResourceHeader
	// Set the maximum payload size we support, as per https://datatracker.ietf.org/doc/html/rfc6891#section-4.3
	if err := rh.SetEDNS0(maxUDPMessageSize, dnsmessage.RCodeSuccess, false); err != nil {
		return nil, fmt.Errorf("set EDNS(0) failed: %w", err)
	}
	if err := b.OPTResource(rh, dnsmessage.OPTResource{}); err != nil {
		return nil, fmt.Errorf("add OPT RR failed: %w", err)
	}

	buf, err := b.Finish()
	if err != nil {
		return nil, fmt.Errorf("message serialization failed: %w", err)
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
		return errors.New("response had no questions")
	}
	respQ := respQs[0]
	if reqQues.Type != respQ.Type || reqQues.Class != respQ.Class || !equalASCIIName(reqQues.Name, respQ.Name) {
		return errors.New("response question doesn't match request")
	}

	return nil
}

// queryDatagram implements a DNS query over a datagram protocol.
func queryDatagram(conn io.ReadWriter, q dnsmessage.Question) (*dnsmessage.Message, error) {
	// Reference: https://cs.opensource.google/go/go/+/master:src/net/dnsclient_unix.go?q=func:dnsPacketRoundTrip&ss=go%2Fgo
	id := uint16(rand.Uint32())
	buf, err := appendRequest(id, q, make([]byte, 0, maxUDPMessageSize))
	if err != nil {
		return nil, &nestedError{ErrBadRequest, fmt.Errorf("append request failed: %w", err)}
	}
	if _, err := conn.Write(buf); err != nil {
		return nil, &nestedError{ErrSend, err}
	}
	buf = buf[:cap(buf)]
	var returnErr error
	for {
		n, err := conn.Read(buf)
		// Handle bad io.Reader.
		if err == io.EOF && n > 0 {
			err = nil
		}
		if err != nil {
			return nil, &nestedError{ErrReceive, errors.Join(returnErr, fmt.Errorf("read message failed: %w", err))}
		}
		var msg dnsmessage.Message
		if err := msg.Unpack(buf[:n]); err != nil {
			returnErr = errors.Join(returnErr, err)
			// Ignore invalid packets that fail to parse. It could be injected.
			continue
		}
		if err := checkResponse(id, q, msg.Header, msg.Questions); err != nil {
			returnErr = errors.Join(returnErr, err)
			continue
		}
		return &msg, nil
	}
}

// queryStream implements a DNS query over a stream protocol. It frames the messages by prepending them with a 2-byte length prefix.
func queryStream(conn io.ReadWriter, q dnsmessage.Question) (*dnsmessage.Message, error) {
	// Reference: https://cs.opensource.google/go/go/+/master:src/net/dnsclient_unix.go?q=func:dnsStreamRoundTrip&ss=go%2Fgo
	id := uint16(rand.Uint32())
	buf, err := appendRequest(id, q, make([]byte, 2, 514))
	if err != nil {
		return nil, &nestedError{ErrBadRequest, fmt.Errorf("append request failed: %w", err)}
	}
	// Buffer length must fit in a uint16.
	if len(buf) > 1<<16-1 {
		return nil, &nestedError{ErrBadRequest, fmt.Errorf("message too large: %v bytes", len(buf))}
	}
	binary.BigEndian.PutUint16(buf[:2], uint16(len(buf)-2))

	// TODO: Consider writer.ReadFrom(net.Buffers) in case the writer is a TCPConn.
	if _, err := conn.Write(buf); err != nil {
		return nil, &nestedError{ErrSend, err}
	}

	var msgLen uint16
	if err := binary.Read(conn, binary.BigEndian, &msgLen); err != nil {
		return nil, &nestedError{ErrReceive, fmt.Errorf("read message length failed: %w", err)}
	}
	if int(msgLen) <= cap(buf) {
		buf = buf[:msgLen]
	} else {
		buf = make([]byte, msgLen)
	}
	if _, err = io.ReadFull(conn, buf); err != nil {
		return nil, &nestedError{ErrReceive, fmt.Errorf("read message failed: %w", err)}
	}

	var msg dnsmessage.Message
	if err = msg.Unpack(buf); err != nil {
		return nil, &nestedError{ErrBadResponse, fmt.Errorf("response failed to unpack: %w", err)}
	}
	if err := checkResponse(id, q, msg.Header, msg.Questions); err != nil {
		return nil, &nestedError{ErrBadResponse, err}
	}
	return &msg, nil
}

func ensurePort(address string, defaultPort string) string {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		// Failed to parse as host:port. Assume address is a host.
		return net.JoinHostPort(address, defaultPort)
	}
	if port == "" {
		return net.JoinHostPort(host, defaultPort)
	}
	return address
}

// NewUDPResolver creates a [Resolver] that implements the DNS-over-UDP protocol, using a [transport.PacketDialer] for transport.
// It uses a different port for every request.
//
// [DNS-over-UDP]: https://datatracker.ietf.org/doc/html/rfc1035#section-4.2.1
func NewUDPResolver(pd transport.PacketDialer, resolverAddr string) Resolver {
	resolverAddr = ensurePort(resolverAddr, "53")
	return FuncResolver(func(ctx context.Context, q dnsmessage.Question) (*dnsmessage.Message, error) {
		conn, err := pd.DialPacket(ctx, resolverAddr)
		if err != nil {
			return nil, &nestedError{ErrDial, err}
		}
		defer conn.Close()
		if deadline, ok := ctx.Deadline(); ok {
			conn.SetDeadline(deadline)
		}
		return queryDatagram(conn, q)
	})
}

type streamResolver struct {
	NewConn func(context.Context) (transport.StreamConn, error)
}

func (r *streamResolver) Query(ctx context.Context, q dnsmessage.Question) (*dnsmessage.Message, error) {
	conn, err := r.NewConn(ctx)
	if err != nil {
		return nil, &nestedError{ErrDial, err}
	}
	// TODO: reuse connection, as per https://datatracker.ietf.org/doc/html/rfc7766#section-6.2.1.
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		conn.SetDeadline(deadline)
	}
	return queryStream(conn, q)
}

// NewTCPResolver creates a [Resolver] that implements the [DNS-over-TCP] protocol, using a [transport.StreamDialer] for transport.
// It creates a new connection to the resolver for every request.
//
// [DNS-over-TCP]: https://datatracker.ietf.org/doc/html/rfc1035#section-4.2.2
func NewTCPResolver(sd transport.StreamDialer, resolverAddr string) Resolver {
	// TODO: Consider handling Authenticated Data.
	resolverAddr = ensurePort(resolverAddr, "53")
	return &streamResolver{
		NewConn: func(ctx context.Context) (transport.StreamConn, error) {
			return sd.DialStream(ctx, resolverAddr)
		},
	}
}

// NewTLSResolver creates a [Resolver] that implements the [DNS-over-TLS] protocol, using a [transport.StreamDialer]
// to connect to the resolverAddr, and the resolverName as the TLS server name.
// It creates a new connection to the resolver for every request.
//
// [DNS-over-TLS]: https://datatracker.ietf.org/doc/html/rfc7858
func NewTLSResolver(sd transport.StreamDialer, resolverAddr string, resolverName string) Resolver {
	resolverAddr = ensurePort(resolverAddr, "853")
	return &streamResolver{
		NewConn: func(ctx context.Context) (transport.StreamConn, error) {
			baseConn, err := sd.DialStream(ctx, resolverAddr)
			if err != nil {
				return nil, err
			}
			return tls.WrapConn(ctx, baseConn, resolverName)
		},
	}
}

// NewHTTPSResolver creates a [Resolver] that implements the [DNS-over-HTTPS] protocol, using a [transport.StreamDialer]
// to connect to the resolverAddr, and the url as the DoH template URI.
// It uses an internal HTTP client that reuses connections when possible.
//
// [DNS-over-HTTPS]: https://datatracker.ietf.org/doc/html/rfc8484
func NewHTTPSResolver(sd transport.StreamDialer, resolverAddr string, url string) Resolver {
	resolverAddr = ensurePort(resolverAddr, "443")
	dialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
		if !strings.HasPrefix(network, "tcp") {
			// TODO: Support UDP for QUIC.
			return nil, fmt.Errorf("protocol not supported: %v", network)
		}
		conn, err := sd.DialStream(ctx, resolverAddr)
		if err != nil {
			return nil, &nestedError{ErrDial, err}
		}
		return conn, nil
	}
	// TODO: add mechanism to close idle connections.
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
		// Prepare request.
		buf, err := appendRequest(0, q, make([]byte, 0, 512))
		if err != nil {
			return nil, &nestedError{ErrBadRequest, fmt.Errorf("append request failed: %w", err)}
		}
		httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(buf))
		if err != nil {
			return nil, &nestedError{ErrBadRequest, fmt.Errorf("create HTTP request failed: %w", err)}
		}
		const mimetype = "application/dns-message"
		httpReq.Header.Add("Accept", mimetype)
		httpReq.Header.Add("Content-Type", mimetype)

		// Send request and get response.
		httpResp, err := httpClient.Do(httpReq)
		if err != nil {
			return nil, &nestedError{ErrReceive, fmt.Errorf("failed to get HTTP response: %w", err)}
		}
		defer httpResp.Body.Close()
		if httpResp.StatusCode != http.StatusOK {
			return nil, &nestedError{ErrReceive, fmt.Errorf("got HTTP status %v", httpResp.StatusCode)}
		}
		response, err := io.ReadAll(httpResp.Body)
		if err != nil {
			return nil, &nestedError{ErrReceive, fmt.Errorf("failed to read response: %w", err)}
		}

		// Process response.
		var msg dnsmessage.Message
		if err = msg.Unpack(response); err != nil {
			return nil, &nestedError{ErrBadResponse, fmt.Errorf("failed to unpack DNS response: %w", err)}
		}
		if err := checkResponse(0, q, msg.Header, msg.Questions); err != nil {
			return nil, &nestedError{ErrBadResponse, err}
		}
		return &msg, nil
	})
}
