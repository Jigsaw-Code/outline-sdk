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

/*
Package tlsfrag provides tools to split the [TLS handshake record] containing
the [Client Hello message] into multiple [TLS records]. This technique,
known as TLS record fragmentation, forces censors to maintain state and
allocate memory for potential reassembly, making censorship more difficult and
resource-intensive. For detailed explanation on how this technique works, refer
to [Circumventing the GFW with TLS Record Fragmentation].

This package offers convenient helper functions to create a TLS
[transport.StreamDialer] that fragments the [TLS handshake record]:
  - [NewFixedLenStreamDialer] creates a [transport.StreamDialer] that splits
    the [Client Hello message] into two records. One of the records will have
    the specified length of splitLen bytes.
  - [NewStreamDialerFunc] offers a more flexible way to fragment [Client Hello
    message]. It accepts a callback function that determines the split point,
    enabling advanced splitting logic such as splitting based on the SNI
    extension.

[Circumventing the GFW with TLS Record Fragmentation]: https://upb-syssec.github.io/blog/2023/record-fragmentation/#tls-record-fragmentation
[TLS records]: https://datatracker.ietf.org/doc/html/rfc8446#section-5.1
[TLS handshake record]: https://datatracker.ietf.org/doc/html/rfc8446#appendix-B.3
[Client Hello message]: https://datatracker.ietf.org/doc/html/rfc8446#section-4.1.2
*/
package tlsfrag
