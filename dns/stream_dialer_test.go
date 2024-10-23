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

package dns

import (
	"context"
	"errors"
	"net/netip"
	"testing"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/dns/dnsmessage"
)

func TestNewStreamDialer(t *testing.T) {
	resolver := FuncResolver(func(ctx context.Context, q dnsmessage.Question) (*dnsmessage.Message, error) {
		resp := new(dnsmessage.Message)
		resp.Header.Response = true
		resp.Questions = []dnsmessage.Question{q}
		answerRR := dnsmessage.Resource{
			Header: dnsmessage.ResourceHeader{Name: q.Name, Type: q.Type, Class: q.Class, TTL: 0},
		}
		switch q.Type {
		case dnsmessage.TypeA:
			answerRR.Body = &dnsmessage.AResource{A: netip.MustParseAddr("127.0.0.1").As4()}
		case dnsmessage.TypeAAAA:
			answerRR.Body = &dnsmessage.AAAAResource{AAAA: netip.MustParseAddr("::1").As16()}
		default:
			return nil, errors.New("bad query type: " + q.Type.String())
		}
		resp.Answers = []dnsmessage.Resource{answerRR}
		resp.Authorities = []dnsmessage.Resource{}
		resp.Additionals = []dnsmessage.Resource{}
		return resp, nil
	})
	addrs := []string{}
	baseDialer := transport.FuncStreamDialer(func(ctx context.Context, addr string) (transport.StreamConn, error) {
		addrs = append(addrs, addr)
		return nil, errors.New("not implemented")
	})
	dialer, err := NewStreamDialer(resolver, baseDialer)
	require.NoError(t, err)
	conn, err := dialer.DialStream(context.Background(), "localhost:8080")
	require.Error(t, err)
	require.Nil(t, conn)
	require.Equal(t, []string{"[::1]:8080", "127.0.0.1:8080"}, addrs)
}

func TestNewStreamDialer_NoResolver(t *testing.T) {
	_, err := NewStreamDialer(nil, &transport.TCPDialer{})
	require.Error(t, err)
}

func TestNewStreamDialer_NoDialer(t *testing.T) {
	_, err := NewStreamDialer(FuncResolver(nil), nil)
	require.Error(t, err)
}
