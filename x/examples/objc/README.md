# Project Overview

This project is a Go-based example that demonstrates how to use `cgo` to call Objective-C code on Apple platforms (macOS and iOS). The primary purpose is to retrieve detailed process and system information by interfacing with the `Foundation` framework.

The project is structured to build both a command-line tool for macOS and a simple application for iOS.

Example output:

```console
% xcrun simctl launch --console booted org.getoutline.test
org.getoutline.test: 43075
Attempting to get iOS process info using Cgo...

--- Successfully Retrieved Process Info ---
Process Name:           main
Process ID (PID):       43075
User Name:              
Full User Name:         
Globally Unique ID:     5A277847-FAB1-41C-930F-AEBFB8BC145C-43075-00000903D8910887
OS Version:             Version 18.4 (Build 22E238)
Hostname:               fortuna-macbookpro2.roam.internal
Is Mac Catalyst App:    false
Is iOS App on Mac:      false
Physical Memory (B):    17179869184
System Uptime (s):      413005.42
Processor Count:        10
Active Processor Count: 10
-------------------------------------------
```

## Building and Running

### macOS

To build and run the example on macOS, execute the following command from the repository root:

```sh
go run -C x ./examples/objc
```

### iOS

Building for iOS requires specifying the correct target architecture and SDK path.

#### iOS Simulator

To build for the iOS Simulator:

```sh
CC="$(xcrun --sdk iphonesimulator --find cc) -isysroot \"$(xcrun --sdk iphonesimulator --show-sdk-path)\"" GOOS=ios GOARCH=arm64 CGO_ENABLED=1 go -C x build  -v -o examples/objc/ProcessInfo.app ./examples/objc/main.go
```

You can boot a simulator using its name (e.g., "iPhone 16") instead of its UDID, as long as the name is unambiguous:

```sh
xcrun simctl boot "iPhone 16"
```

To run on a booted simulator:

```sh
xcrun simctl install booted ./x/examples/objc/ProcessInfo.app
xcrun simctl launch --console booted org.getoutline.test
```

#### iOS Device

To build for a physical iOS device:

```sh
CC="$(xcrun --sdk iphoneos --find cc) -isysroot \"$(xcrun --sdk iphoneos --show-sdk-path)\"" GOOS=ios GOARCH=arm64 CGO_ENABLED=1 go -C x build -v -o examples/objc/ProcessInfo.app ./examples/objc/main.go
```

Note: Directly installing the resulting `.app` bundle on a physical device from the command line is not straightforward due to Apple's strict code signing requirements. You will typically need to use Xcode to manage the signing and installation process.

## Development Conventions

- **Language:** The project uses Go with `cgo` to interface with Objective-C.
- **Dependencies:** It relies on the `Foundation` framework on Apple platforms.
- **Code Style:** The Go code follows standard Go formatting. The Objective-C code is embedded within `main.go` in a comment block.
- **Memory Management:** The code demonstrates manual memory management for C-allocated memory by using `defer C.free`.
