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
Package configurl provides convenience functions to create network objects based on a text config.
This is experimental and mostly for illustrative purposes at this point.

Configurable strategies simplifies the way you create and manage strategies.
With the configurl package, you can use [ProviderContainer.NewPacketDialer], [ProviderContainer.NewStreamDialer] and [ProviderContainer.NewPacketListener] to create objects using a simple text string.

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

It takes a list of count*length pairs meaning splitting the sequence in count segments of the given length. If you omit "[COUNT]*", it's assumed to be 1.

	split:[COUNT1]*[LENGTH1],[COUNT2]*[LENGTH2],...

TLS fragmentation (streams only, package [github.com/Jigsaw-Code/outline-sdk/transport/tlsfrag]).

The Client Hello record payload will be split into two fragments of size LENGTH and len(payload)-LENGTH if LENGTH>0.
If LENGTH<0, the two fragments will be of size len(payload)-LENGTH and LENGTH respectively.
For more details, refer to [github.com/Jigsaw-Code/outline-sdk/transport/tlsfrag].

	tlsfrag:[LENGTH]

Packet reordering (streams only, package [github.com/Jigsaw-Code/outline-sdk/x/disorder])

The disorder strategy sends TCP packets out of order by manipulating the
socket's Time To Live (TTL) or Hop Limit. It temporarily sets the TTL to a low
value, causing specific packets to be dropped early in the network (like at the
first router). These dropped packets are then re-transmitted later by the TCP
stack, resulting in the receiver getting packets out of order. This can help
bypass network filters that rely on inspecting the initial packets of a TCP
connection.

	disorder:[PACKET_NUMBER]

PACKET_NUMBER: The number of writes before the disorder action occurs. The
disorder action triggers when the number of writes equals PACKET_NUMBER. If set
to 0 (default), the disorder happens on the first write. If set to 1, it happens
on the second write, and so on.

# Examples

Packet splitting - To split outgoing streams on bytes 2 and 123, you can use:

	split:2|split:123

Disorder transport - Send some of the packets out of order:

	disorder:0|split:123

Split at position 123, then send packet 0 of 123 bytes (from splitting) out of order. The network filter will first receive packet 1, only then packet 0. This
is done by setting the hop limit for the write to 1, and then restoring it. It will be sent with its original hop limit on retransmission.

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

Defining custom strategies - You can define your custom strategy by implementing and registering [BuildFunc[ObjectType]] functions:

	// Create new config parser.
	// p := configurl.NewProviderContainer()
	// or
	p := configurl.NewDefaultProviders()
	// Register your custom dialer.
	p.StreamDialers.RegisterType("custom", func(ctx context.Context, config *Config) (transport.StreamDialer, error) {
	  // Build logic
	  // ...
	})
	p.PacketDialers.RegisterType("custom", func(ctx context.Context, config *Config) (transport.PacketDialer, error) {
	  // Build logic
	  // ...
	})
	// Then use it
	dialer, err := p.NewStreamDialer(context.Background(), "custom://config")

[Onion Routing]: https://en.wikipedia.org/wiki/Onion_routing
*/
package configurl
