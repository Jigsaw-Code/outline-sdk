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

## Integration Methods

The Outline SDK is written in Go. Choose from one of the following methods to integrate the Outline SDK:

- **Generated Mobile Library**: Ideal for Android, iOS, and macOS apps. Uses [`gomobile bind`](https://pkg.go.dev/golang.org/x/mobile/cmd/gomobile) for generating Java and Objective-C bindings.
- **Side Service**: Suitable for desktop and Android applications to run a standalone Go binary that your application talks to. Not available on iOS due to subprocess limitations.
- **Go Library**: Directly import the SDK into your Go application. [API Reference](https://pkg.go.dev/github.com/Jigsaw-Code/outline-sdk).
- **Generated C Library**: Generate C bindings using [`go build`](https://pkg.go.dev/cmd/go#hdr-Build_modes).

The Outline Client uses a **generated mobile library** on Android, iOS and macOS (based on Cordova) and a **side service** on Windows and Linux (based on Electron).

Below we provide more details on the integration approaches.

### Generated Mobile Library

To integrate the Outline SDK into a mobile application, you'll need to generate Java and Objective-C bindings. Direct dependency on Go libraries is not supported in mobile apps.

Steps:
1. **Create a Go library**: Create a Go package that wraps the SDK functionalities you need.
1. **Generate mobile library**: Use [`gomobile bind`](https://pkg.go.dev/golang.org/x/mobile/cmd/gomobile) to generate Android Archives (AAR) and Apple Frameworks with Java and Objective-C bindings.
    - Android examples: [Outline Android Archive](https://github.com/Jigsaw-Code/outline-go-tun2socks/blob/dada2652ae2c6205f2daa3f88c805bbd6b28a713/Makefile#L27), [Intra Android Archive](https://github.com/Jigsaw-Code/outline-go-tun2socks/blob/dada2652ae2c6205f2daa3f88c805bbd6b28a713/Makefile#L21).
    - Apple examples: [Outline iOS Framework](https://github.com/Jigsaw-Code/outline-go-tun2socks/blob/dada2652ae2c6205f2daa3f88c805bbd6b28a713/Makefile#L30), [Outline macOS Framework](https://github.com/Jigsaw-Code/outline-go-tun2socks/blob/dada2652ae2c6205f2daa3f88c805bbd6b28a713/Makefile#L36).
1. **Integrate into your app**: Add the generated library to your app. For more details, refer to Go Mobile's [SDK applications and generating bindings](https://github.com/golang/go/wiki/Mobile#sdk-applications-and-generating-bindings).

> **Note**: You must call Go Mobile on a package you create. Calling it on the SDK packages will not work.


### Side Service

You can build a Go binary that can easily integrate with the SDK as a regular Go library dependency. Your application can then run the binary as a subprocess. You can communicate with the binary using a predefined protocol via an inter-process communication mechanism (IPC) (for example, sockets, standard I/O, command-line flags).

Steps:
1. **Define the IPC mechanism**
1. **Build the service** in Go, using the SDK. Include the server-side of the IPC.
    - Examples: [Outline Electron backend code](https://github.com/Jigsaw-Code/outline-go-tun2socks/blob/master/outline/electron/main.go), [Outline Windows Client backend build](https://github.com/Jigsaw-Code/outline-go-tun2socks/blob/dada2652ae2c6205f2daa3f88c805bbd6b28a713/Makefile#L67), [Outline Linux Client backend build](https://github.com/Jigsaw-Code/outline-go-tun2socks/blob/dada2652ae2c6205f2daa3f88c805bbd6b28a713/Makefile#L56).
1. **Bundle the service** into your application bundle.
    - Examples: [Outline Windows Client](https://github.com/Jigsaw-Code/outline-client/blob/b06819922037230ee3ba9471097c40793af819e8/src/electron/electron-builder.json#L21), [Outline Linux Client](https://github.com/Jigsaw-Code/outline-client/blob/b06819922037230ee3ba9471097c40793af819e8/src/electron/electron-builder.json#L10)
1. **Add code to start the service** in your app, by launching it as a sub-process.
    - Example: [Outline Electron Clients](https://github.com/Jigsaw-Code/outline-client/blob/b06819922037230ee3ba9471097c40793af819e8/src/electron/go_vpn_tunnel.ts#L227)
1. **Add the service calls** for your app to communicate with the service.


### Go Library

To integrate the Outline SDK as a Go library, you can directly import it into your Go application. See the [API Reference](https://pkg.go.dev/github.com/Jigsaw-Code/outline-sdk) for what's available.


This approach is suitable for both command-line and GUI-based applications. You can build GUI-based applications in Go with frameworks like like [Wails](https://wails.io/), [Fyne.io](https://fyne.io/), [Qt for Go](https://therecipe.github.io/qt/), or [Go Mobile app](https://pkg.go.dev/golang.org/x/mobile/app).

You can find multiple examples of Go applications using the SDK in [x/examples](./x/examples/).

### Generated C Library

For applications that require C bindings. This approach is similar to the Generated Mobile Library approach, where you need to first create a Go library to generate bindings for.

Steps:

1. **Create a Go library**: Create a Go package that wraps the SDK functionalities you need. Functions to be exported must be marked with `//export`, as per the [cgo documentation](https://pkg.go.dev/cmd/cgo#hdr-C_references_to_Go).
1. **Generate C library**: Use `go build` with the [appropriate `-buildmode` flag](https://pkg.go.dev/cmd/go#hdr-Build_modes). Anternatively, you can use [SWIG](https://swig.org/Doc4.1/Go.html#Go).
1. **Integrate into your app**: Add the generated C library to your application, according to your build system.


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

- Transport client strategies
  - Proxyless strategies
    - [ ] Encrypted DNS
    - [ ] Packet splitting
  - Proxy-based strategies
    - [ ] HTTP Connect
    - [x] SOCKS5 StreamDialer
    - [ ] SOCKS5 PacketDialer

- Integration resources
  - For Mobile apps
    - [ ] Library to run a local SOCKS5 or HTTP-Connect proxy
    - [ ] Documentation on how to integrate the SDK into mobile apps
    - [ ] Connectivity Test mobile app using [Capacitor](https://capacitorjs.com/)
  - For Go apps
    - [ ] Connectivity Test example [Wails](https://wails.io/) graphical app
    - [ ] Connectivity Test example command-line app
    - [ ] Outline Client example command-line app
    - [ ] Page fetch example command-line app
    - [ ] Local proxy example command-line app

- Server-side libraries
  - [ ] To be defined

- Other Resources
  - [ ] Website
  - [ ] Bindings
