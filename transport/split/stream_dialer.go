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

package split

import (
	"context"
	"errors"
	"io"

	"github.com/Jigsaw-Code/outline-sdk/transport"
)

type splitWriter struct {
	writer      io.Writer
	splitPoint  int64
	bytesCopied int64
}

var _ io.ReaderFrom = (*splitWriter)(nil)

func (w *splitWriter) ReadFrom(source io.Reader) (int64, error) {
	var written int64
	defer func() {
		w.bytesCopied += written
	}()
	n, err := io.CopyN(w.writer, source, w.splitPoint-w.bytesCopied)
	written += n
	if err != nil {
		return n, err
	}
	n, err = io.Copy(w.writer, source)
	written += n
	return n, err
}

func (w *splitWriter) Write(data []byte) (int, error) {
	var written int
	prefixLength := w.splitPoint - w.bytesCopied
	defer func() {
		w.bytesCopied += int64(written)
	}()
	if 0 < prefixLength && prefixLength < int64(len(data)) {
		n, err := w.writer.Write(data[:prefixLength])
		written += n
		if err != nil {
			return n, err
		}
		data = data[prefixLength:]
	}
	n, err := w.writer.Write(data)
	written += n
	return written, err
}

type splitDialer struct {
	dialer     transport.StreamDialer
	splitPoint int64
}

var _ transport.StreamDialer = (*splitDialer)(nil)

// NewStreamDialer creates a client that splits the outgoing strean at byte splitpoint.
func NewStreamDialer(dialer transport.StreamDialer, splitPoint int64) (transport.StreamDialer, error) {
	if dialer == nil {
		return nil, errors.New("argument dialer must not be nil")
	}
	return &splitDialer{dialer: dialer, splitPoint: splitPoint}, nil
}

func (d *splitDialer) Dial(ctx context.Context, remoteAddr string) (transport.StreamConn, error) {
	innerConn, err := d.dialer.Dial(ctx, remoteAddr)
	if err != nil {
		return nil, err
	}
	return transport.WrapConn(innerConn, innerConn, &splitWriter{writer: innerConn, splitPoint: d.splitPoint}), nil
}
