# Outline SDK (Internal, Under Development)

![Build Status](https://github.com/Jigsaw-Code/outline-internal-sdk/actions/workflows/test.yml/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/Jigsaw-Code/outline-internal-sdk)](https://goreportcard.com/report/github.com/Jigsaw-Code/outline-internal-sdk)
[![Go Reference](https://pkg.go.dev/badge/github.com/Jigsaw-Code/outline-internal-sdk.svg)](https://pkg.go.dev/github.com/Jigsaw-Code/outline-internal-sdk)

[![Mattermost](https://badgen.net/badge/Mattermost/Outline%20Community/blue)](https://community.internetfreedomfestival.org/community/channels/outline-community)
[![Reddit](https://badgen.net/badge/Reddit/r%2Foutlinevpn/orange)](https://www.reddit.com/r/outlinevpn/)

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
