# Outline SDK (Alpha)

[![Build Status](https://github.com/Jigsaw-Code/outline-sdk/actions/workflows/test.yml/badge.svg)](https://github.com/Jigsaw-Code/outline-sdk/actions/workflows/test.yml?query=branch%3Amain)
[![Go Report Card](https://goreportcard.com/badge/github.com/Jigsaw-Code/outline-sdk)](https://goreportcard.com/report/github.com/Jigsaw-Code/outline-sdk)
[![Go Reference](https://pkg.go.dev/badge/github.com/Jigsaw-Code/outline-sdk.svg)](https://pkg.go.dev/github.com/Jigsaw-Code/outline-sdk)

<p align="center">
<img src="https://github.com/Jigsaw-Code/outline-brand/blob/main/assets/powered_by_outline/color/logo.png?raw=true" width=400pt />
</p>

> ⚠️ **Warning**: This code is in early stages and is not guaranteed to be stable. If you are
> interested in integrating with it, we'd love your [feedback](https://github.com/Jigsaw-Code/outline-sdk/issues/new).

The Outline SDK allows you to:

- Create tools to protect against network-level interference
- Add network-level interference protection to existing apps, such as content or communication apps

## Advantages

| Multi-Platform | Proven Technology | Composable |
|:-:|:-:|:-:|
| The Outline SDK can be used to build tools that run on Android, iOS, Windows, macOS and Linux. | The Outline Client and Server have been using the code in the SDK for years, helping millions of users access the internet in even the harshest conditions. | The SDK interfaces were carefully designed to allow for composition and reuse, so you can craft your own transport. |

## Integration

The Outline SDK is written in Go. There are multiple ways to integrate the Outline SDK into your app:

- As a **Go library** ([reference](https://pkg.go.dev/github.com/Jigsaw-Code/outline-sdk)) in a Go application (CLI or graphical app with frameworks like [Fyne.io](https://fyne.io/), [Wails](https://wails.io/), [Qt for Go](https://therecipe.github.io/qt/), or [Go Mobile app](https://pkg.go.dev/golang.org/x/mobile/app)).
- As a **C library**, generated using the appropriate [Go build mode](https://pkg.go.dev/cmd/go#hdr-Build_modes).
- As a native **mobile library**, using [`gomobile bind`](https://pkg.go.dev/golang.org/x/mobile/cmd/gomobile) to generate [Java and Objective-C bindings](https://pkg.go.dev/golang.org/x/mobile/cmd/gobind) for Android, iOS and macOS.
- As a **side service**, built as a standalone Go binary that your main application talks to. Note that this is not possible on iOS, due to the limitation on starting sub-processes.

The Outline Client uses the mobile library approach on Android, iOS and macOS (based on Cordova) and the side service on Windows and Linux (based on Electron).

## Tentative Roadmap

The launch will have two milestones: Alpha and Beta. We are currently in Alpha. Most of the code is not new. It's the same code that is currently being used by the production Outline Client and Server. The SDK repackages code from [outline-ss-server](https://github.com/Jigsaw-Code/outline-ss-server) and [outline-go-tun2socks](https://github.com/Jigsaw-Code/outline-go-tun2socks) in a way that is easier to reuse and extend.

### Alpha

The goal of the Alpha release is to make it available to potential developers early, so they can provide feedback on the SDK and help shape the interfaces, processes and tools.

Alpha features:

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
  - [x] Add DelegatePacketProxy for runtime PacketProxy replacement

### Beta

The goal of the Beta release is to make the SDK available for broader consumption, once we no longer expect major changes to the APIs and we have all supporting resources in place (website, documentation, examples, and so on).

Beta features:

- Network libraries
  - [ ] Use network libraries in the Outline Client
  - [ ] Add extensive testing

- Serverless transport libraries
  - [ ] Encrypted DNS
  - [ ] Packet splitting

- Proxy transport libraries
  - [ ] HTTP Connect
  - [x] SOCKS5 StreamDialer
  - [ ] SOCKS5 PacketDialer

- Add Resources
  - [ ] Website
  - [ ] Bindings
  - [ ] Integration documentation
  - [ ] Example command-line apps
  - [ ] Example graphical apps
