// Copyright 2025 The Outline Authors
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

package httpconnect

import (
	"errors"
	"io"
	"net"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/transport"
)

var _ transport.StreamConn = (*pipeConn)(nil)

type pipeConn struct {
	reader     readCloseDeadliner
	writer     writeCloseDeadliner
	remoteAddr net.Addr
}

type readCloseDeadliner interface {
	io.ReadCloser
	SetDeadline(deadline time.Time) error
}

type writeCloseDeadliner interface {
	io.WriteCloser
	SetDeadline(deadline time.Time) error
}

func newPipeConn(writer writeCloseDeadliner, reader readCloseDeadliner, remoteAddr net.Addr) *pipeConn {
	return &pipeConn{
		reader:     reader,
		writer:     writer,
		remoteAddr: remoteAddr,
	}
}

func (p *pipeConn) Read(b []byte) (n int, err error) {
	return p.reader.Read(b)
}

func (p *pipeConn) Write(b []byte) (n int, err error) {
	return p.writer.Write(b)
}

func (p *pipeConn) CloseRead() error {
	return p.reader.Close()
}

func (p *pipeConn) CloseWrite() error {
	return p.writer.Close()
}

func (p *pipeConn) Close() error {
	return errors.Join(p.reader.Close(), p.writer.Close())
}

func (p *pipeConn) LocalAddr() net.Addr {
	return nil
}

func (p *pipeConn) RemoteAddr() net.Addr {
	return p.remoteAddr
}

func (p *pipeConn) SetDeadline(t time.Time) error {
	return errors.Join(p.writer.SetDeadline(t), p.reader.SetDeadline(t))
}

func (p *pipeConn) SetReadDeadline(t time.Time) error {
	return p.reader.SetDeadline(t)
}

func (p *pipeConn) SetWriteDeadline(t time.Time) error {
	return p.writer.SetDeadline(t)
}
