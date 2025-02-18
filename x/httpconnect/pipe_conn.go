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
	"github.com/Jigsaw-Code/outline-sdk/transport"
	"io"
)

var _ transport.StreamConn = (*pipeConn)(nil)

// pipeConn is a [transport.StreamConn] that overrides the Read and Write functions with the provided [io.ReadCloser] and [io.WriteCloser], respectively.
// The CloseRead, CloseWrite, and Close functions first close the [io.ReadCloser] and [io.WriteCloser], and then call the corresponding functions on the connection.
type pipeConn struct {
	reader io.ReadCloser
	writer io.WriteCloser
	transport.StreamConn
}

func (p *pipeConn) Read(b []byte) (n int, err error) {
	return p.reader.Read(b)
}

func (p *pipeConn) Write(b []byte) (n int, err error) {
	return p.writer.Write(b)
}

func (p *pipeConn) CloseRead() error {
	return errors.Join(p.reader.Close(), p.StreamConn.CloseRead())
}

func (p *pipeConn) CloseWrite() error {
	return errors.Join(p.writer.Close(), p.StreamConn.CloseWrite())
}

func (p *pipeConn) Close() error {
	return errors.Join(p.reader.Close(), p.writer.Close(), p.StreamConn.Close())
}
