// Copyright 2024 Jigsaw Operations LLC
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

package parameters

type TransferURL struct {

	// URL is the location of the resource. This string is slightly obfuscated
	// with base64 encoding to mitigate trivial binary executable string scanning.
	URL string

	// SkipVerify indicates whether to verify HTTPS certificates. In some
	// circumvention scenarios, verification is not possible. This must
	// only be set to true when the resource has its own verification mechanism.
	SkipVerify bool

	// OnlyAfterAttempts specifies how to schedule this URL when transferring
	// the same resource (same entity, same ETag) from multiple different
	// candidate locations. For a value of N, this URL is only a candidate
	// after N rounds of attempting the transfer to or from other URLs.
	OnlyAfterAttempts int

	// B64EncodedPublicKey is a base64-encoded RSA public key to be used for
	// encrypting the resource, when uploading, or for verifying a signature of
	// the resource, when downloading. Required by some operations, such as
	// uploading feedback.
	B64EncodedPublicKey string `json:",omitempty"`

	// RequestHeaders are optional HTTP headers to set on any requests made to
	// the destination.
	RequestHeaders map[string]string `json:",omitempty"`
}

type TransferURLs []*TransferURL
