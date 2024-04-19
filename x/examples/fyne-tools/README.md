# DNS System Resolver Lookup Utility

To run:

```console
go run github.com/Jigsaw-Code/outline-sdk/x/examples/fyne-tools@latest
```

## Android

Package for Android and install on emulator or device:
```
go run fyne.io/fyne/v2/cmd/fyne package -os android && adb install Net_Tools.apk
```

Note: the generated APK is around 85MB.


## Desktop

If you are on the target platform (OS and architecture), you can just use the regular `go build` or `go run` command.

Because the app uses cgo, we need to cross-compilation tools to build from other platforms.

The easiest way is to [use zig](https://dev.to/kristoff/zig-makes-go-cross-compilation-just-work-29ho).

[Install zig](https://ziglang.org/learn/getting-started/#installing-zig) and make sure it's in the PATH.

You can download the binary tarball, or [use a package manager](https://github.com/ziglang/zig/wiki/Install-Zig-from-a-Package-Manager), like Homebrew:

```sh
brew install zig 
```

To buuild the Windows app:

```sh
GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC='zig cc -target x86_64-windows' go build .
```

To build the Linux app:

```sh
GOOS=linux GOARCH=amd64 CGO_ENABLED=1 CC='zig cc -target x86_64-linux' go build .
```

The first build will take minutes, since there's a lot of platform code to be built.
Subsequent builds will be incremental and take a few seconds.

## Screenshots

<img width="400" alt="image" src="https://github.com/Jigsaw-Code/outline-sdk/assets/113565/8cead9da-461e-41c8-8ce3-f263d77c6ee8">

<img width="462" alt="image" src="https://github.com/Jigsaw-Code/outline-sdk/assets/113565/9782eab3-d142-4be7-9431-5384c866384d">