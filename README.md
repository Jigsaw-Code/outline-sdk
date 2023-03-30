# Outline SDK (Internal, Under Development)

This is the repository to hold the future Outline SDK as we develop it. the goal is to clean up and move reusable code from [outline-ss-server](https://github.com/Jigsaw-Code/outline-ss-server) and [outline-go-tun2socks](https://github.com/Jigsaw-Code/outline-go-tun2socks).

Tentative roadmap:

- Transport libraries
  - [x] Generic transport client primitives (`StreamDialer`, `PacketListener` and Endpoints)
  - [x] TCP and UDP client implementations
  - [x] Shadowsocks client implementations
  - [ ] Generic transport server primitives (TBD)
  - [ ] TCP and UDP server implementations
  - [ ] Shadowsocks server implementations
  - [ ] Utility implementations (`ReplaceablePacketListener`, `TruncateDNSPacketListener`)

- Network libraries
  - [ ] Generic network primitives (TBD, something like a generic TUN device)
  - [ ] Implementation based on go-tun2socks

- VPN API
  - [ ] VPN API for desktop (Linux, Windows, macOS)
