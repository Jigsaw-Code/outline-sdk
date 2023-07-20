# Outline SDK (Under Development, DO NOT USE)

[![Build Status](https://github.com/Jigsaw-Code/outline-internal-sdk/actions/workflows/test.yml/badge.svg)](https://github.com/Jigsaw-Code/outline-internal-sdk/actions/workflows/test.yml?query=branch%3Amain)
[![Go Report Card](https://goreportcard.com/badge/github.com/Jigsaw-Code/outline-internal-sdk)](https://goreportcard.com/report/github.com/Jigsaw-Code/outline-internal-sdk)
[![Go Reference](https://pkg.go.dev/badge/github.com/Jigsaw-Code/outline-internal-sdk.svg)](https://pkg.go.dev/github.com/Jigsaw-Code/outline-internal-sdk)

[![Mattermost](https://badgen.net/badge/Mattermost/Outline%20Community/blue)](https://community.internetfreedomfestival.org/community/channels/outline-community)
[![Reddit](https://badgen.net/badge/Reddit/r%2Foutlinevpn/orange)](https://www.reddit.com/r/outlinevpn/)

<p align="center">
<img src="https://github.com/Jigsaw-Code/outline-brand/blob/main/assets/powered_by_outline/color/logo.png?raw=true" width=400pt />
</p>

> **Warning**
> This code is not ready to be used by the public. There's no guarantee of stability.

The Outline SDK helps developers:
- Create tools to protect against network-level interference
- Add network-level interference protection to existing apps, such as content or communication apps

## Advantages

| Multi-Platform | Proven Technology | Composable |
|:-:|:-:|:-:|
| The Outline SDK can be used on Android, iOS, Windows, macOS or Linux. | The Outline Client and Server have been using the code in the SDK for years, helping millions of users in the harshest conditions. | The SDK interfaces were carefully designed to allow for composition and reuse, so you can craft your own transport. |

## Integration

The Outline SDK is written in Go. There are multiple ways to integrate the Outline SDK into your app:

- As a **Go library** ([reference](https://pkg.go.dev/github.com/Jigsaw-Code/outline-internal-sdk)), in a Go application (CLI or graphical app with frameworks like [Fyne.io](https://fyne.io/), [Wails](https://wails.io/), [Qt for Go](https://therecipe.github.io/qt/), or [Go Mobile app](https://pkg.go.dev/golang.org/x/mobile/app))
- As a **C library**, generated using the appropriate [Go build mode](https://pkg.go.dev/cmd/go#hdr-Build_modes).
- As a native **mobile library**, using [`gomobile bind`](https://pkg.go.dev/golang.org/x/mobile/cmd/gomobile) to generate Java and Objective-C bindings for Android, iOS and macOS.
- As a **side service**, built as a standalone Go binary that your main application talks to. Note that this is not possible on iOS, due to the limitation on starting sub-processes.

The Outline Client uses the mobile library approach on Android, iOS and macOS (based on Cordova) and the side service on Windows and Linux (based on Electron).


## Tentative Roadmap

The launch will have two milestones: Alpha and Beta. We are currently in pre-Alpha. Note that most of the code is not new. It's the code being used by the production Outline Client and Server. The SDK work is repackaging code from [outline-ss-server](https://github.com/Jigsaw-Code/outline-ss-server) and [outline-go-tun2socks](https://github.com/Jigsaw-Code/outline-go-tun2socks) in a way that is easier to reuse and extend.

### Alpha
The goal of the Alpha release is to make it available to potential developers early so they can provide feedback on the SDK and help shape the interfaces, processes and tools.

The code in this repository will move to https://github.com/Jigsaw-Code/outline-sdk and versions will be tagged.

Alpha tasks:

- Transport-level libraries
  - [x] Add generic transport client primitives (`StreamDialer`, `PacketListener` and Endpoints)
  - [x] Add TCP and UDP client implementations
  - [x] Add Shadowsocks client implementations
  - [x] Use transport libraries in the Outline Client
  - [x] Use transport libraries in the Outline Server 

- Network-level libraries
  - [x] Add IP Device abstraction
  - [x] Add IP Device implementation based on go-tun2socks (LWIP)
  - [x] Add UDP handler to fallback to DNS-over-TCP
  - [ ] Add DelegatePacketProxy


### Beta

The goal of the Beta release is to communicate that the SDK is ready for broader consumption, after we believe the APIs are stable enough and we have all the supporting resources in place (website, documentation, examples, etc

Beta tasks:

- Network libraries
  - [ ] Use network libraries in the Outline Client
  - [ ] Add extensive testing

- Serverless transport libraries
  - [ ] Encrypted DNS
  - [ ] Packet splitting

- Proxy transport libraries
  - [ ] HTTP Connect
  - [ ] SOCKS5

- Add Resources
  - [ ] Website
  - [ ] Bindings
  - [ ] Integration documentation
  - [ ] Example command-line apps
  - [ ] Example graphical apps
