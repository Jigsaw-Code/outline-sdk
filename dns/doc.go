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

/*
Package dns provides utilities to interact with the Domain Name System (DNS).

The [Domain Name System] (DNS) is responsible for mapping domain names to IP addresses.
Because domain resolution gatekeeps connections and is predominantly done in plaintext, it is commonly used
for network-level filtering.

The main concept in this library is that of a [Resolver], which allows code to query the DNS. Different implementations are provided
to perform DNS resolution over multiple transports:

  - DNS-over-UDP: the standard mechanism of querying resolvers. Communication is done in plaintext, using port 53.
  - [DNS-over-TCP]: alternative to UDP when responses are large. Communication is done in plaintext, using port 53.
  - [DNS-over-TLS] (DoT): uses the TCP protocol, but over a connection encrypted with TLS. Is uses port 853, which
    makes it very easy to block using the port number, as no other protocol is assigned to that port.
  - [DNS-over-HTTPS] (DoH): uses HTTP exchanges for querying the resolver and communicates over a connection encrypted with TLS. It uses
    port 443. That makes the DoH traffic undistinguishable from web traffic, making it harder to block.

[Domain Name System]: https://datatracker.ietf.org/doc/html/rfc1034
[DNS-over-TCP]: https://datatracker.ietf.org/doc/html/rfc7766
[DNS-over-TLS]: https://datatracker.ietf.org/doc/html/rfc7858
[DNS-over-HTTPS]: https://datatracker.ietf.org/doc/html/rfc8484
*/
package dns
