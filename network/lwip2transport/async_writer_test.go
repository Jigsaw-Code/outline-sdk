// Copyright 2026 The Outline Authors
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

package lwip2transport

import (
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.getoutline.org/sdk/network"
)

const testTimeout = 5 * time.Second

// TestWriteToDoesNotBlockOnSlowWriter is the reason asyncWriter exists: lwIP calls
// forwardOutgoingIPPacket while holding its global stack lock, so that callback must return without
// waiting for the destination write to complete.
func TestWriteToDoesNotBlockOnSlowWriter(t *testing.T) {
	h := &errTcpUdpHandler{err: errors.New("not supported")}
	d := reConfigurelwIPDeviceForTest(t, h, h)
	defer d.Close()

	w := newBlockingWriter()
	go d.WriteTo(w)

	// The first packet parks the background writer inside w.Write.
	forwardWithinTimeout(t, d, []byte{0x01})
	waitForWriter(t, w)

	// While that write is stuck, the callback must still return promptly.
	for i := range 10 {
		forwardWithinTimeout(t, d, []byte{byte(i)})
	}

	close(w.release)
}

// TestWriteToPreservesPacketOrder makes sure the queue does not reorder packets.
func TestWriteToPreservesPacketOrder(t *testing.T) {
	h := &errTcpUdpHandler{err: errors.New("not supported")}
	d := reConfigurelwIPDeviceForTest(t, h, h)
	defer d.Close()

	const pktCount = 100
	w := newRecordingWriter(pktCount)
	go d.WriteTo(w)

	for i := range pktCount {
		forwardWithinTimeout(t, d, []byte{byte(i), 0xff})
	}
	for i := range pktCount {
		select {
		case got := <-w.written:
			require.Equal(t, []byte{byte(i), 0xff}, got)
		case <-time.After(testTimeout):
			t.Fatalf("only %d of %d packets reached the writer", i, pktCount)
		}
	}
}

// TestWriteToCopiesPacketBeforeReturning verifies that the buffer handed to forwardOutgoingIPPacket
// is safe to reuse as soon as it returns. On the lwIP path that buffer aliases the C pbuf payload,
// which is freed once the output callback returns.
func TestWriteToCopiesPacketBeforeReturning(t *testing.T) {
	h := &errTcpUdpHandler{err: errors.New("not supported")}
	d := reConfigurelwIPDeviceForTest(t, h, h)
	defer d.Close()

	w := newBlockingWriter()
	go d.WriteTo(w)

	pkt := []byte{0x01, 0x02, 0x03, 0x04}
	forwardWithinTimeout(t, d, pkt)
	waitForWriter(t, w)

	// Simulate lwIP recycling the pbuf, then let the write proceed.
	copy(pkt, []byte{0x09, 0x09, 0x09, 0x09})
	close(w.release)

	select {
	case got := <-w.got:
		require.Equal(t, []byte{0x01, 0x02, 0x03, 0x04}, got)
	case <-time.After(testTimeout):
		t.Fatal("the writer was never called")
	}
}

// TestWriteToReturnsWriterError verifies that an error from the background writer terminates
// WriteTo. It surfaces at least one packet late, so the test keeps feeding packets until it does.
func TestWriteToReturnsWriterError(t *testing.T) {
	h := &errTcpUdpHandler{err: errors.New("not supported")}
	d := reConfigurelwIPDeviceForTest(t, h, h)
	defer d.Close()

	writeErr := errors.New("tun is gone")
	errCh := make(chan error, 1)
	go func() {
		_, err := d.WriteTo(newErrWriter(writeErr))
		errCh <- err
	}()

	// Feed packets until WriteTo goes away; the final call unblocks when the device is closed.
	fed := make(chan struct{})
	go func() {
		defer close(fed)
		for {
			if _, err := d.forwardOutgoingIPPacket([]byte{0x01}); err != nil {
				return
			}
		}
	}()

	select {
	case err := <-errCh:
		require.ErrorIs(t, err, writeErr)
	case <-time.After(testTimeout):
		t.Fatal("WriteTo did not report the write error")
	}

	require.NoError(t, d.Close())
	select {
	case <-fed:
	case <-time.After(testTimeout):
		t.Fatal("forwardOutgoingIPPacket did not return after the device was closed")
	}
}

// TestCloseUnblocksWriteTo verifies that closing the device stops an idle WriteTo.
func TestCloseUnblocksWriteTo(t *testing.T) {
	h := &errTcpUdpHandler{err: errors.New("not supported")}
	d := reConfigurelwIPDeviceForTest(t, h, h)

	done := make(chan error, 1)
	go func() {
		_, err := d.WriteTo(io.Discard)
		done <- err
	}()

	require.NoError(t, d.Close())
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(testTimeout):
		t.Fatal("WriteTo did not return after the device was closed")
	}
}

// TestAsyncWriterClose verifies that Close stops the background goroutine, is idempotent, and
// reports the write error it observed.
func TestAsyncWriterClose(t *testing.T) {
	h := &errTcpUdpHandler{err: errors.New("not supported")}
	d := reConfigurelwIPDeviceForTest(t, h, h)
	defer d.Close()

	aw := d.newAsyncWriter(io.Discard)
	require.NoError(t, aw.Close())
	select {
	case <-aw.exit:
	default:
		t.Fatal("the background writer is still running after Close")
	}
	require.NoError(t, aw.Close(), "Close must be idempotent")

	// Writes after Close are rejected rather than queued forever.
	n, err := aw.Write([]byte{0x01})
	require.Exactly(t, 0, n)
	require.ErrorIs(t, err, network.ErrClosed)

	// Close reports the error the background writer hit. Wait for the write to actually happen:
	// Close drops whatever is still queued, so a packet closed out from under the writer never
	// produces an error at all.
	writeErr := errors.New("tun is gone")
	fw := newErrWriter(writeErr)
	failing := d.newAsyncWriter(fw)
	_, err = failing.Write([]byte{0x01})
	require.NoError(t, err, "the first write only queues the packet")
	select {
	case <-fw.called:
	case <-time.After(testTimeout):
		t.Fatal("the writer was never called")
	}
	require.ErrorIs(t, failing.Close(), writeErr)
}

// TestReadForwardedPacket verifies the Read path still hands the packet over synchronously.
func TestReadForwardedPacket(t *testing.T) {
	h := &errTcpUdpHandler{err: errors.New("not supported")}
	d := reConfigurelwIPDeviceForTest(t, h, h)
	defer d.Close()

	forwarded := make(chan int, 1)
	go func() {
		n, err := d.forwardOutgoingIPPacket([]byte{0x01, 0x02, 0x03})
		require.NoError(t, err)
		forwarded <- n
	}()

	buf := make([]byte, packetMTU)
	n, err := d.Read(buf)
	require.NoError(t, err)
	require.Exactly(t, 3, n)
	require.Equal(t, []byte{0x01, 0x02, 0x03}, buf[:n])

	select {
	case n := <-forwarded:
		require.Exactly(t, 3, n)
	case <-time.After(testTimeout):
		t.Fatal("forwardOutgoingIPPacket did not return after the packet was read")
	}
}

// waitForWriter blocks until the background writer has entered a Write call.
func waitForWriter(t *testing.T, w *blockingWriter) {
	t.Helper()
	select {
	case <-w.entered:
	case <-time.After(testTimeout):
		t.Fatal("the writer was never called")
	}
}

// forwardWithinTimeout calls forwardOutgoingIPPacket and fails the test if it does not return.
func forwardWithinTimeout(t *testing.T, d *lwIPDevice, pkt []byte) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		defer close(done)
		n, err := d.forwardOutgoingIPPacket(pkt)
		require.NoError(t, err)
		require.Exactly(t, len(pkt), n)
	}()
	select {
	case <-done:
	case <-time.After(testTimeout):
		t.Fatal("forwardOutgoingIPPacket blocked; the lwIP stack lock would be held for this long")
	}
}

// blockingWriter signals that Write was entered, parks until release is closed, and only then reads
// the packet it was given. Reading late is deliberate: it proves the packet contents outlive the
// return of forwardOutgoingIPPacket.
type blockingWriter struct {
	entered chan struct{}
	release chan struct{}
	got     chan []byte
}

func newBlockingWriter() *blockingWriter {
	return &blockingWriter{
		entered: make(chan struct{}, 1),
		release: make(chan struct{}),
		got:     make(chan []byte, 1),
	}
}

func (w *blockingWriter) Write(p []byte) (int, error) {
	select {
	case w.entered <- struct{}{}:
	default:
	}
	<-w.release
	select {
	case w.got <- append([]byte(nil), p...):
	default:
	}
	return len(p), nil
}

// recordingWriter reports every packet it is given, in order.
type recordingWriter struct {
	written chan []byte
}

func newRecordingWriter(depth int) *recordingWriter {
	return &recordingWriter{written: make(chan []byte, depth)}
}

func (w *recordingWriter) Write(p []byte) (int, error) {
	w.written <- append([]byte(nil), p...)
	return len(p), nil
}

// errWriter fails every write and reports that it was called at least once.
type errWriter struct {
	err    error
	called chan struct{}
}

func newErrWriter(err error) *errWriter {
	return &errWriter{err: err, called: make(chan struct{}, 1)}
}

func (w *errWriter) Write(p []byte) (int, error) {
	select {
	case w.called <- struct{}{}:
	default:
	}
	return 0, w.err
}
