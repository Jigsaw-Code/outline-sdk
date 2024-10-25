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

package shadowsocks

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/chacha20poly1305"
)

// Overhead for cipher chacha20poly1305
const testCipherOverhead = 16

func TestCipherReaderAuthenticationFailure(t *testing.T) {
	key, err := NewEncryptionKey(CHACHA20IETFPOLY1305, "test secret")
	require.NoError(t, err)

	clientReader := strings.NewReader("Fails Authentication")
	reader := NewReader(clientReader, key)
	_, err = reader.Read(make([]byte, 1))
	if err == nil {
		t.Fatalf("Expected authentication failure, got %v", err)
	}
}

func TestCipherReaderUnexpectedEOF(t *testing.T) {
	key, err := NewEncryptionKey(CHACHA20IETFPOLY1305, "test secret")
	require.NoError(t, err)

	clientReader := strings.NewReader("short")
	server := NewReader(clientReader, key)
	_, err = server.Read(make([]byte, 10))
	require.Equal(t, io.ErrUnexpectedEOF, err)
}

func TestCipherReaderEOF(t *testing.T) {
	key, err := NewEncryptionKey(CHACHA20IETFPOLY1305, "test secret")
	require.NoError(t, err)

	clientReader := strings.NewReader("")
	server := NewReader(clientReader, key)
	_, err = server.Read(make([]byte, 10))
	if err != io.EOF {
		t.Fatalf("Expected EOF, got %v", err)
	}
	_, err = server.Read([]byte{})
	if err != io.EOF {
		t.Fatalf("Expected EOF, got %v", err)
	}
}

func encryptBlocks(key *EncryptionKey, salt []byte, blocks [][]byte) (io.Reader, error) {
	var ssText bytes.Buffer
	aead, err := key.NewAEAD(salt)
	if err != nil {
		return nil, fmt.Errorf("Failed to create AEAD: %v", err)
	}
	ssText.Write(salt)
	// buf must fit the larges block ciphertext
	buf := make([]byte, 2+100+testCipherOverhead)
	var expectedCipherSize int
	nonce := make([]byte, chacha20poly1305.NonceSize)
	for _, block := range blocks {
		ssText.Write(aead.Seal(buf[:0], nonce, []byte{0, byte(len(block))}, nil))
		nonce[0]++
		expectedCipherSize += 2 + testCipherOverhead
		ssText.Write(aead.Seal(buf[:0], nonce, block, nil))
		nonce[0]++
		expectedCipherSize += len(block) + testCipherOverhead
	}
	if ssText.Len() != key.SaltSize()+expectedCipherSize {
		return nil, fmt.Errorf("cipherText has size %v. Expected %v", ssText.Len(), key.SaltSize()+expectedCipherSize)
	}
	return &ssText, nil
}

func TestCipherReaderGoodReads(t *testing.T) {
	key, err := NewEncryptionKey(CHACHA20IETFPOLY1305, "test secret")
	require.NoError(t, err)

	salt := []byte("12345678901234567890123456789012")
	if len(salt) != key.SaltSize() {
		t.Fatalf("Salt has size %v. Expected %v", len(salt), key.SaltSize())
	}
	ssText, err := encryptBlocks(key, salt, [][]byte{
		[]byte("[First Block]"),
		[]byte(""), // Corner case: empty block
		[]byte("[Third Block]")})
	if err != nil {
		t.Fatal(err)
	}

	reader := NewReader(ssText, key)
	plainText := make([]byte, len("[First Block]")+len("[Third Block]"))
	n, err := io.ReadFull(reader, plainText)
	if err != nil {
		t.Fatalf("Failed to fully read plain text. Got %v bytes: %v", n, err)
	}
	_, err = reader.Read([]byte{})
	if err != io.EOF {
		t.Fatalf("Expected EOF, got %v", err)
	}
	_, err = reader.Read(make([]byte, 1))
	if err != io.EOF {
		t.Fatalf("Expected EOF, got %v", err)
	}
}

func TestCipherReaderClose(t *testing.T) {
	key, err := NewEncryptionKey(CHACHA20IETFPOLY1305, "test secret")
	require.NoError(t, err)

	pipeReader, pipeWriter := io.Pipe()
	server := NewReader(pipeReader, key)
	result := make(chan error)
	go func() {
		_, err := server.Read(make([]byte, 10))
		result <- err
	}()
	pipeWriter.Close()
	err = <-result
	if err != io.EOF {
		t.Fatalf("Expected ErrUnexpectedEOF, got %v", err)
	}
}

func TestCipherReaderCloseError(t *testing.T) {
	key, err := NewEncryptionKey(CHACHA20IETFPOLY1305, "test secret")
	require.NoError(t, err)

	pipeReader, pipeWriter := io.Pipe()
	server := NewReader(pipeReader, key)
	result := make(chan error)
	go func() {
		_, err := server.Read(make([]byte, 10))
		result <- err
	}()
	pipeWriter.CloseWithError(fmt.Errorf("xx!!ERROR!!xx"))
	err = <-result
	if err == nil || !strings.Contains(err.Error(), "xx!!ERROR!!xx") {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestEndToEnd(t *testing.T) {
	key, err := NewEncryptionKey(CHACHA20IETFPOLY1305, "test secret")
	require.NoError(t, err)

	connReader, connWriter := io.Pipe()
	writer := NewWriter(connWriter, key)
	reader := NewReader(connReader, key)
	expected := "Test"
	wg := sync.WaitGroup{}
	var writeErr error
	go func() {
		defer connWriter.Close()
		wg.Add(1)
		defer wg.Done()
		_, writeErr = writer.Write([]byte(expected))
	}()
	var output bytes.Buffer
	_, readErr := reader.WriteTo(&output)
	wg.Wait()
	if writeErr != nil {
		t.Fatalf("Failed Write: %v", writeErr)
	}
	if readErr != nil {
		t.Fatalf("Failed WriteTo: %v", readErr)
	}
	if output.String() != expected {
		t.Fatalf("Expected output '%v'. Got '%v'", expected, output.String())
	}
}

func TestLazyWriteFlush(t *testing.T) {
	key, err := NewEncryptionKey(CHACHA20IETFPOLY1305, "test secret")
	require.NoError(t, err)
	buf := new(bytes.Buffer)
	writer := NewWriter(buf, key)
	header := []byte{1, 2, 3, 4}
	n, err := writer.LazyWrite(header)
	require.NoError(t, err, "LazyWrite failed: %v", err)
	require.Equal(t, len(header), n, "Wrong write size")
	require.Equal(t, 0, buf.Len(), "LazyWrite isn't lazy")

	err = writer.Flush()
	require.NoError(t, err, "Flush failed: %v", err)

	len1 := buf.Len()
	require.Greater(t, len1, len(header), "Not enough bytes flushed")

	// Check that normal writes now work
	body := []byte{5, 6, 7}
	n, err = writer.Write(body)
	require.NoError(t, err, "Write failed: %v", err)
	require.Equal(t, len(body), n, "Wrong write size")
	require.Greater(t, buf.Len(), len1, "No write observed")

	// Verify content arrives in two blocks
	reader := NewReader(buf, key)
	decrypted := make([]byte, len(header)+len(body))
	n, err = reader.Read(decrypted)
	require.NoError(t, err, "Read failed: %v", err)
	require.Equal(t, len(header), n, "Wrong number of bytes out")
	require.Equal(t, header, decrypted[:n], "Wrong final content")

	n, err = reader.Read(decrypted[n:])
	require.NoError(t, err, "Read failed: %v", err)
	require.Equal(t, len(body), n, "Wrong number of bytes out")
	require.Equal(t, body, decrypted[len(header):], "Wrong final content")
}

func TestLazyWriteConcat(t *testing.T) {
	key, err := NewEncryptionKey(CHACHA20IETFPOLY1305, "test secret")
	require.NoError(t, err)
	buf := new(bytes.Buffer)
	writer := NewWriter(buf, key)
	header := []byte{1, 2, 3, 4}
	n, err := writer.LazyWrite(header)
	if n != len(header) {
		t.Errorf("Wrong write size: %d", n)
	}
	if err != nil {
		t.Errorf("LazyWrite failed: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("LazyWrite isn't lazy: %v", buf.Bytes())
	}

	// Write additional data and flush the header.
	body := []byte{5, 6, 7}
	n, err = writer.Write(body)
	if n != len(body) {
		t.Errorf("Wrong write size: %d", n)
	}
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}
	len1 := buf.Len()
	if len1 <= len(body)+len(header) {
		t.Errorf("Not enough bytes flushed: %d", len1)
	}

	// Flush after write should have no effect
	if err = writer.Flush(); err != nil {
		t.Errorf("Flush failed: %v", err)
	}
	if buf.Len() != len1 {
		t.Errorf("Flush should have no effect")
	}

	// Verify content arrives in one block
	reader := NewReader(buf, key)
	decrypted := make([]byte, len(body)+len(header))
	n, err = reader.Read(decrypted)
	if n != len(decrypted) {
		t.Errorf("Wrong number of bytes out: %d", n)
	}
	if err != nil {
		t.Errorf("Read failed: %v", err)
	}
	if !bytes.Equal(decrypted[:len(header)], header) ||
		!bytes.Equal(decrypted[len(header):], body) {
		t.Errorf("Wrong final content: %v", decrypted)
	}
}

func TestLazyWriteOversize(t *testing.T) {
	key, err := NewEncryptionKey(CHACHA20IETFPOLY1305, "test secret")
	require.NoError(t, err)
	buf := new(bytes.Buffer)
	writer := NewWriter(buf, key)
	N := 25000 // More than one block, less than two.
	data := make([]byte, N)
	for i := range data {
		data[i] = byte(i)
	}
	n, err := writer.LazyWrite(data)
	if n != len(data) {
		t.Errorf("Wrong write size: %d", n)
	}
	if err != nil {
		t.Errorf("LazyWrite failed: %v", err)
	}
	if buf.Len() >= N {
		t.Errorf("Too much data in first block: %d", buf.Len())
	}
	if err = writer.Flush(); err != nil {
		t.Errorf("Flush failed: %v", err)
	}
	if buf.Len() <= N {
		t.Errorf("Not enough data written after flush: %d", buf.Len())
	}

	// Verify content
	reader := NewReader(buf, key)
	decrypted, err := io.ReadAll(reader)
	if len(decrypted) != N {
		t.Errorf("Wrong number of bytes out: %d", len(decrypted))
	}
	if err != nil {
		t.Errorf("Read failed: %v", err)
	}
	if !bytes.Equal(decrypted, data) {
		t.Errorf("Wrong final content: %v", decrypted)
	}
}

func TestLazyWriteConcurrentFlush(t *testing.T) {
	key, err := NewEncryptionKey(CHACHA20IETFPOLY1305, "test secret")
	require.NoError(t, err)
	buf := new(bytes.Buffer)
	writer := NewWriter(buf, key)
	header := []byte{1, 2, 3, 4}
	n, err := writer.LazyWrite(header)
	require.NoError(t, err, "LazyWrite failed: %v", err)
	require.Equalf(t, len(header), n, "Wrong write size: %d", n)
	require.Equal(t, 0, buf.Len(), "LazyWrite isn't lazy: %v", buf.Bytes())

	body := []byte{5, 6, 7}
	r, w := io.Pipe()
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		n, err := writer.ReadFrom(r)
		require.NoError(t, err, "ReadFrom: %v", err)
		require.Equalf(t, int64(len(body)), n, "ReadFrom: Wrong read size %d", n)
		wg.Done()
	}()

	// Wait for ReadFrom to start and get blocked.
	time.Sleep(20 * time.Millisecond)

	// Flush while ReadFrom is blocked.
	require.Nilf(t, writer.Flush(), "Flush error: %v", err)

	len1 := buf.Len()
	require.Greater(t, len1, 0, "No bytes flushed")

	// Check that normal writes now work
	n, err = w.Write(body)
	require.NoError(t, err, "Write failed: %v", err)
	require.Equalf(t, len(body), n, "Wrong write size: %d", n)

	w.Close()
	wg.Wait()
	require.Greater(t, buf.Len(), len1, "No write observed")

	// Verify content arrives in two blocks
	reader := NewReader(buf, key)
	decrypted := make([]byte, len(header)+len(body))
	n, err = reader.Read(decrypted)
	require.Equal(t, len(header), n, "Wrong number of bytes out")
	require.NoError(t, err, "Read failed: %v", err)

	require.Equal(t, header, decrypted[:len(header)], "Wrong final content")

	n, err = reader.Read(decrypted[len(header):])
	require.NoError(t, err, "Read failed: %v", err)
	require.Equal(t, len(body), n, "Wrong number of bytes out")
	require.Equal(t, body, decrypted[len(header):], "Wrong final content")
}

type nullIO struct{}

func (n *nullIO) Write(b []byte) (int, error) {
	return len(b), nil
}

func (r *nullIO) Read(b []byte) (int, error) {
	return len(b), nil
}

// Microbenchmark for the performance of Shadowsocks TCP encryption.
func BenchmarkWriter(b *testing.B) {
	b.StopTimer()
	b.ResetTimer()

	key, err := NewEncryptionKey(CHACHA20IETFPOLY1305, "test secret")
	require.NoError(b, err)
	writer := NewWriter(new(nullIO), key)

	start := time.Now()
	b.StartTimer()
	io.CopyN(writer, new(nullIO), int64(b.N))
	b.StopTimer()
	elapsed := time.Since(start)

	megabits := 8 * float64(b.N) * 1e-6
	b.ReportMetric(megabits/(elapsed.Seconds()), "mbps")
}
