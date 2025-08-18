// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tlsfrag

import (
	"errors"
	"log"

	"golang.org/x/crypto/cryptobyte"
)

// GetSNI accepts the beginning of a TLS connection and returns the
// indicated server name, or an error if the server name was not found.
// Derived from unmarshal() in crypto/tls.
func GetSNI(clienthello []byte) (string, error) {
	plaintext := cryptobyte.String(clienthello)

	log.Printf("plaintext: %v \n", plaintext)

	var s cryptobyte.String
	// Skip uint8 ContentType and uint16 ProtocolVersion
	if !plaintext.Skip(1+2) || !plaintext.ReadUint16LengthPrefixed(&s) {
		return "", errors.New("Bad TLSPlaintext")
	}

	// Skip uint8 message type, uint24 length, uint16 version, and 32 byte random.
	var sessionID cryptobyte.String
	if !s.Skip(1+3+2+32) ||
		!s.ReadUint8LengthPrefixed(&sessionID) {
		return "", errors.New("Bad Handshake message")
	}

	var cipherSuites cryptobyte.String
	if !s.ReadUint16LengthPrefixed(&cipherSuites) {
		return "", errors.New("Bad ciphersuites")
	}

	var compressionMethods cryptobyte.String
	if !s.ReadUint8LengthPrefixed(&compressionMethods) {
		return "", errors.New("Bad compression methods")
	}

	if s.Empty() {
		// ClientHello is optionally followed by extension data
		return "", errors.New("Short hello")
	}

	var extensions cryptobyte.String
	if !s.ReadUint16LengthPrefixed(&extensions) || !s.Empty() {
		return "", errors.New("Bad extensions")
	}

	for !extensions.Empty() {
		var extension uint16
		var extData cryptobyte.String
		if !extensions.ReadUint16(&extension) ||
			!extensions.ReadUint16LengthPrefixed(&extData) {
			return "", errors.New("Bad extension")
		}

		switch extension {
		case 0: // Extension ID 0 is ServerName
			// RFC 6066, Section 3
			var nameList cryptobyte.String
			if !extData.ReadUint16LengthPrefixed(&nameList) || nameList.Empty() {
				return "", errors.New("Bad namelist")
			}
			for !nameList.Empty() {
				var nameType uint8
				var serverName cryptobyte.String
				if !nameList.ReadUint8(&nameType) ||
					!nameList.ReadUint16LengthPrefixed(&serverName) ||
					serverName.Empty() {
					return "", errors.New("Bad SNI")
				}
				if nameType != 0 {
					continue
				}
				return string(serverName), nil
			}
		default:
			// Ignore all other extensions.
			continue
		}
	}
	return "", errors.New("No SNI")
}
