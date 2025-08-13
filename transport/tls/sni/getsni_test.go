package tls

import (
	"encoding/hex"
	"testing"
)

// ClientHello for www.wikipedia.org, as a hex string.
const HEX = "1603010200010001fc0303168cafd33e2cde2db2c48f3e3ec1d32567c362e7c42f3f865768e2602e6bdeb020457210ccbdbe991fd206ff8481bfab5e7f2099038b48ec1a5220f03d2d574d7100222a2a130113021303c02bc02fc02cc030cca9cca8c013c014009c009d002f0035000a010001914a4a00000000001600140000117777772e77696b6970656469612e6f726700170000ff01000100000a000a0008caca001d00170018000b00020100002300000010000e000c02683208687474702f312e31000500050100000000000d00140012040308040401050308050501080606010201001200000033002b0029caca000100001d00202a9dfacdd81fa3a4c7300bdb6ee5d98e9774eb75c3fe7878d8a2b1802e092f6e002d00020101002b000b0a1a1a0304030303020301001b00030200020a0a000100001500c700000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"

func TestGetSNI(t *testing.T) {
	clienthello, err := hex.DecodeString(HEX)
	if err != nil {
		t.Error(err)
		return
	}
	sni, err := GetSNI(clienthello)
	if err != nil {
		t.Error(err)
		return
	}
	if sni != "www.wikipedia.org" {
		t.Errorf("Wrong SNI: %s", sni)
	}
}

func BenchmarkGetSNI(b *testing.B) {
	clienthello, _ := hex.DecodeString(HEX)
	for i := 0; i < b.N; i++ {
		GetSNI(clienthello)
	}
}

func TestGetSNIShort(t *testing.T) {
	clienthello, err := hex.DecodeString(HEX)
	if err != nil {
		t.Error(err)
		return
	}
	// Only provide the first 64 bytes.  This segment doesn't include the SNI,
	// so GetSNI should return an error.
	sni, err := GetSNI(clienthello[:64])
	if err == nil {
		t.Error("Expected failure")
	}
	if sni != "" {
		t.Errorf("Expected empty SNI: %s", sni)
	}
}

func TestGetSNILong(t *testing.T) {
	clienthello, err := hex.DecodeString(HEX)
	if err != nil {
		t.Error(err)
		return
	}
	// Append 100 bytes containing arbitrary data.
	// GetSNI should ignore the extra data.
	extra := [100]byte{17}
	clienthello = append(clienthello, extra[:]...)
	sni, err := GetSNI(clienthello)
	if err != nil {
		t.Error(err)
		return
	}
	if sni != "www.wikipedia.org" {
		t.Errorf("Wrong SNI: %s", sni)
	}
}
