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
Package config provides convenience functions to create dialer objects based on a text config.
This is experimental and mostly for illustrative purposes at this point.

Configurable transports simplifies the way you create and manage transports.
With the config package, you can use [NewPacketDialer] and [NewStreamDialer] to create dialers using a simple text string.

Key Benefits:

  - Ease of Use: Create transports effortlessly by providing a textual configuration, reducing boilerplate code.
  - Serialization: Easily share configurations with users or between different parts of your application, including your Go backend.
  - Dynamic Configuration: Set your app's transport settings at runtime.
  - DPI Evasion: Advanced nesting and configuration options help you evade Deep Packet Inspection (DPI).

# Config Format

The configuration string is composed of parts separated by the `|` symbol, which define nested dialers.
For example, `A|B` means dialer `B` takes dialer `A` as its input.
An empty string represents the direct TCP/UDP dialer, and is used as the input to the first cofigured dialer.

Each dialer configuration follows a URL format, where the scheme defines the type of Dialer. Supported formats are described below.

# Proxy Protocols

Shadowsocks proxy (compatible with Outline's access keys, package [github.com/Jigsaw-Code/outline-sdk/transport/shadowsocks])

	ss://[USERINFO]@[HOST]:[PORT]?prefix=[PREFIX]

SOCKS5 proxy (works with both stream and packet dialers, package [github.com/Jigsaw-Code/outline-sdk/transport/socks5])

	socks5://[USERINFO]@[HOST]:[PORT]

USERINFO field is optional and only required if username and password authentication is used. It is in the format of username:password.

# Transports

TLS transport (currently streams only, package [github.com/Jigsaw-Code/outline-sdk/transport/tls])

The sni parameter defines the name to be sent in the TLS SNI. It can be empty.
The certname parameter defines what name to validate against the server certificate.

	tls:sni=[SNI]&certname=[CERT_NAME]

WebSockets

	ws:tcp_path=[PATH]&udp_path=[PATH]

# DNS Protection

DNS resolution (streams only, package [github.com/Jigsaw-Code/outline-sdk/dns])

It takes a host:port address. If the port is missing, it will use 53. The resulting dialer will use the input dialer with
Happy Eyeballs to connect to the destination.

	do53:address=[ADDRESS]

DNS-over-HTTPS resolution (streams only, package [github.com/Jigsaw-Code/outline-sdk/dns])

It takes a host name and a host:port address. The name will be used in the SNI and Host header, while the address is used to connect
to the DoH server. The address is optional, and will default to "[NAME]:443". The resulting dialer will use the input dialer with
Happy Eyeballs to connect to the destination.

	doh:name=[NAME]&address=[ADDRESS]

Address override.

This dialer configuration is helpful for testing and development or if you need to fix the domain
resolution.
The host parameter, if not empty, specifies the host to dial instead of the original host.
The port parameter, if not empty, specifies the port to dial instead of the original port.

	override:host=[HOST]&port=[PORT]

# Packet manipulation

These strategies manipulate packets to bypass SNI-based blocking.

Stream split transport (streams only, package [github.com/Jigsaw-Code/outline-sdk/transport/split])

It takes the length of the prefix. The stream will be split when PREFIX_LENGTH bytes are first written.

	split:[PREFIX_LENGTH]

TLS fragmentation (streams only, package [github.com/Jigsaw-Code/outline-sdk/transport/tlsfrag]).

The Client Hello record payload will be split into two fragments of size LENGTH and len(payload)-LENGTH if LENGTH>0.
If LENGTH<0, the two fragments will be of size len(payload)-LENGTH and LENGTH respectively.
For more details, refer to [github.com/Jigsaw-Code/outline-sdk/transport/tlsfrag].

	tlsfrag:[LENGTH]

# Examples

Packet splitting - To split outgoing streams on bytes 2 and 123, you can use:

	split:2|split:123

Evading DNS and SNI blocking - You can use Cloudflare's DNS-over-HTTPS to protect against DNS disruption.
The DoH resolver cloudflare-dns.com is accessible from any cloudflare.net IP, so you can specify the address to avoid blocking
of the resolver itself. This can be combines with a TCP split or TLS Record Fragmentation to bypass SNI-based blocking:

	doh:name=cloudflare-dns.com.&address=cloudflare.net.:443|split:2

SOCKS5-over-TLS, with domain-fronting - To tunnel SOCKS5 over TLS, and set the SNI to decoy.example.com, while still validating against your host name, use:

	tls:sni=decoy.example.com&certname=[HOST]|socks5:[HOST]:[PORT]

Onion Routing with Shadowsocks - To route your traffic through three Shadowsocks servers, similar to [Onion Routing], use:

	ss://[USERINFO1]@[HOST1]:[PORT1]|ss://[USERINFO2]@[HOST2]:[PORT2]|ss://[USERINFO3]@[HOST3]:[PORT3]

In that case, HOST1 will be your entry node, and HOST3 will be your exit node.

DPI Evasion - To add packet splitting to a Shadowsocks server for enhanced DPI evasion, use:

	split:2|ss://[USERINFO]@[HOST]:[PORT]

Defining custom transport - You can define your custom transport by implementing and registering the [NewStreamDialerFunc] and [NewPacketDialerFunc] functions:

	// create new config parser
	// p := new(ConfigToDialer)
	// or
	p := NewDefaultConfigToDialer()
	// register your custom dialer
	p.RegisterPacketDialerWrapper("custom", wrapStreamDialerWithCustom)
	p.RegisterStreamDialerWrapper("custom", wrapPacketDialerWithCustom)
	// then use it
	dialer, err := p.NewStreamDialer(innerDialer, "custom://config")

where wrapStreamDialerWithCustom and wrapPacketDialerWithCustom implement [NewPacketDialerFunc] and [NewStreamDialerFunc].

[Onion Routing]: https://en.wikipedia.org/wiki/Onion_routing
*/
package configurl
