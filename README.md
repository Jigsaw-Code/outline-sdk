# Outline SDK (Beta)

[![Build Status](https://github.com/Jigsaw-Code/outline-sdk/actions/workflows/test.yml/badge.svg)](https://github.com/Jigsaw-Code/outline-sdk/actions/workflows/test.yml?query=branch%3Amain)
[![Go Report Card](https://goreportcard.com/badge/github.com/Jigsaw-Code/outline-sdk)](https://goreportcard.com/report/github.com/Jigsaw-Code/outline-sdk)
[![Go Reference](https://pkg.go.dev/badge/github.com/Jigsaw-Code/outline-sdk.svg)](https://pkg.go.dev/github.com/Jigsaw-Code/outline-sdk)

<p align="center">
<img src="https://github.com/Jigsaw-Code/outline-brand/blob/main/assets/powered_by_outline/color/logo.png?raw=true" width=400pt />
</p>

> ⚠️ **Warning**: This code is in early stages and is not guaranteed to be stable. If you are
> interested in integrating with it, we'd love your [feedback](https://github.com/Jigsaw-Code/outline-sdk/issues/new).

The Outline SDK allows you to:

- Create tools to protect against network-level interference.
- Add network-level interference protection to existing apps, such as content or communication apps.

## Advantages

| Multi-Platform | Proven Technology | Composable |
|:-:|:-:|:-:|
| Supports Android, iOS, Windows, macOS and Linux. | Field-tested in the Outline Client and Server, helping millions to access the internet under harsh conditions. | Designed for modularity and reuse, allowing you to craft custom transports. |


## Integration Methods

Choose from one of the following methods to integrate the Outline SDK into your project:

- **Generated Mobile Library**: For Android, iOS, and macOS apps. Uses [`gomobile bind`](https://pkg.go.dev/golang.org/x/mobile/cmd/gomobile) to generate Java and Objective-C bindings.
- **Side Service**: For desktop and Android apps. Runs a standalone Go binary that your application communicates with (not available on iOS due to subprocess limitations).
- **Go Library**: Directly import the SDK into your Go application. [API Reference](https://pkg.go.dev/github.com/Jigsaw-Code/outline-sdk).
- **Generated C Library**: Generate C bindings using [`go build`](https://pkg.go.dev/cmd/go#hdr-Build_modes).

The Outline Client uses a **generated mobile library** on Android, iOS and macOS (based on Cordova) and a **side service** on Windows and Linux (based on Electron).

Below we provide more details on each integration approach. For more details about setting up and using Outline SDK features, see the [Discussions tab](https://github.com/Jigsaw-Code/outline-sdk/discussions).

### Generated Mobile Library

To integrate the SDK into a mobile app, follow these steps:

1. **Create a Go library**: Create a Go package that wraps the SDK functionalities you need.
1. **Generate mobile library**: Use [`gomobile bind`](https://pkg.go.dev/golang.org/x/mobile/cmd/gomobile) to generate Android Archives (AAR) and Apple Frameworks with Java and Objective-C bindings.
    - Android examples: [Outline Android Archive](https://github.com/Jigsaw-Code/outline-go-tun2socks/blob/dada2652ae2c6205f2daa3f88c805bbd6b28a713/Makefile#L27), [Intra Android Archive](https://github.com/Jigsaw-Code/outline-go-tun2socks/blob/dada2652ae2c6205f2daa3f88c805bbd6b28a713/Makefile#L21).
    - Apple examples: [Outline iOS Framework](https://github.com/Jigsaw-Code/outline-go-tun2socks/blob/dada2652ae2c6205f2daa3f88c805bbd6b28a713/Makefile#L30), [Outline macOS Framework](https://github.com/Jigsaw-Code/outline-go-tun2socks/blob/dada2652ae2c6205f2daa3f88c805bbd6b28a713/Makefile#L36).
1. **Integrate into your app**: Add the generated library to your app. For more details, see Go Mobile's [SDK applications and generating bindings](https://github.com/golang/go/wiki/Mobile#sdk-applications-and-generating-bindings).

> **Note**: You must use `gomobile bind` on the package you create, not directly on the SDK packages.

An easy way to integrate with the SDK in a mobile app is by using the [`x/mobileproxy` library](./x/mobileproxy/) 
to run a local web proxy that you can use to configure your app's networking libraries.

### Side Service

To integrate the SDK as a side service, follow these steps:

1. **Define IPC mechanism**: Choose an inter-process communication (IPC) mechanism (for example, sockets, standard I/O).
1. **Build the service**: Create a Go binary that includes the server-side of the IPC and used the SDK.
    - Examples: [Outline Electron backend code](https://github.com/Jigsaw-Code/outline-go-tun2socks/blob/master/outline/electron/main.go), [Outline Windows Client backend build](https://github.com/Jigsaw-Code/outline-go-tun2socks/blob/dada2652ae2c6205f2daa3f88c805bbd6b28a713/Makefile#L67), [Outline Linux Client backend build](https://github.com/Jigsaw-Code/outline-go-tun2socks/blob/dada2652ae2c6205f2daa3f88c805bbd6b28a713/Makefile#L56).
3. **Bundle the service**: Include the Go binary in your application bundle.
    - Examples: [Outline Windows Client](https://github.com/Jigsaw-Code/outline-client/blob/b06819922037230ee3ba9471097c40793af819e8/src/electron/electron-builder.json#L21), [Outline Linux Client](https://github.com/Jigsaw-Code/outline-client/blob/b06819922037230ee3ba9471097c40793af819e8/src/electron/electron-builder.json#L10)
4. **Start the service**: Launch the Go binary as a subprocess from your application.
    - Example: [Outline Electron Clients](https://github.com/Jigsaw-Code/outline-client/blob/b06819922037230ee3ba9471097c40793af819e8/src/electron/go_vpn_tunnel.ts#L227)
5. **Service Calls**: Add code to your app for communication with the service.


### Go Library

To integrate the Outline SDK as a Go library, you can directly import it into your Go application. See the [API Reference](https://pkg.go.dev/github.com/Jigsaw-Code/outline-sdk) for what's available.


This approach is suitable for both command-line and GUI-based applications. You can build GUI-based applications in Go with frameworks like [Wails](https://wails.io/), [Fyne.io](https://fyne.io/), [Qt for Go](https://therecipe.github.io/qt/), or [Go Mobile app](https://pkg.go.dev/golang.org/x/mobile/app).

For examples, see [x/examples](./x/examples/).

### Generated C Library

This approach is suited for applications that require C bindings. It is similar to the Generated Mobile Library approach, where you need to first create a Go library to generate bindings for.

Steps:

1. **Create a Go library**: Create a Go package that wraps the SDK functionalities you need. Functions to be exported must be marked with `//export`, as described in the [cgo documentation](https://pkg.go.dev/cmd/cgo#hdr-C_references_to_Go).
1. **Generate C library**: Use `go build` with the [appropriate `-buildmode` flag](https://pkg.go.dev/cmd/go#hdr-Build_modes). Anternatively, you can use [SWIG](https://swig.org/Doc4.1/Go.html#Go).
1. **Integrate into your app**: Add the generated C library to your application, according to your build system.


## Tentative Roadmap

This launch is currently in Beta. Most of the code is not new. It's the same code that is currently being used by the production Outline Client and Server. The SDK repackages code from [outline-ss-server](https://github.com/Jigsaw-Code/outline-ss-server) and [outline-go-tun2socks](https://github.com/Jigsaw-Code/outline-go-tun2socks) in a way that is easier to reuse and extend.

### Features

- Network-level libraries
  - [x] IP Device abstraction (v0.0.2)
  - [x] IP Device implementation based on go-tun2socks (LWIP) (v0.0.2)
  - [x] UDP handler to fallback to DNS-over-TCP (v0.0.2)
  - [x] DelegatePacketProxy for runtime PacketProxy replacement (v0.0.2)

- Proxy protocols
  - [x] TCP and UDP Dialers (v0.0.2)
  - [x] Shadowsocks wrappers and Dialers ([reference](https://pkg.go.dev/github.com/Jigsaw-Code/outline-sdk/transport/shadowsocks)) (v0.0.2)
  - [x] SOCKS5 StreamDialer ([reference](https://pkg.go.dev/github.com/Jigsaw-Code/outline-sdk/transport/socks5)) (v0.0.6)
  - [ ] SOCKS5 PacketDialer (coming soon)
  - [ ] HTTP Connect (coming soon)

- Transport protocols
  - [x] Stream (TCP) split ([reference](https://pkg.go.dev/github.com/Jigsaw-Code/outline-sdk/transport/split)) (v0.0.6)
  - [x] TLS connection wrapper and StreamDialer ([reference](https://pkg.go.dev/github.com/Jigsaw-Code/outline-sdk/transport/tls))

- Name resolution
  - [ ] Resilient DNS (coming soon)
  - [ ] Encrypted DNS (coming soon)

- Integration resources
  - For Mobile apps
    - [x] Library to run a local SOCKS5 or HTTP-Connect proxy ([source](./x/mobileproxy/mobileproxy.go), [example Go usage](./x/examples/fetch-proxy/main.go), [example mobile usage](./x/examples/mobileproxy)).
    - [x] Documentation on how to integrate the SDK into mobile apps
    - [x] Connectivity Test mobile app (iOS and Android) using [Capacitor](https://capacitorjs.com/)
  - For Go apps
    - [x] Connectivity Test example [Wails](https://wails.io/) graphical app
    - [x] Connectivity Test example command-line app ([source](./x/examples/outline-connectivity/))
    - [x] Outline Client example command-line app ([source](./x/examples/outline-cli/))
    - [x] Page fetch example command-line app ([source](./x/examples/outline-fetch/))
    - [x] Local proxy example command-line app ([source](./x/examples/http2transport/))
