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


## Windows

From macOS, you can build the app for Windows with MinGW.

First install MinGW:

```
brew install mingw-w64
```

Build for 64-bit:

```
GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC="x86_64-w64-mingw32-gcc" go build .
```

32-bit:

```
GOOS=windows GOARCH=386 CGO_ENABLED=1 CC="i686-w64-mingw32-gcc" go build .
```

The first build will take minutes, since there's a lot of platform code to be built.
Subsequent builds will be incremental and take a few seconds.

## Screenshots

<img width="400" alt="image" src="https://github.com/Jigsaw-Code/outline-sdk/assets/113565/8cead9da-461e-41c8-8ce3-f263d77c6ee8">

<img width="462" alt="image" src="https://github.com/Jigsaw-Code/outline-sdk/assets/113565/9782eab3-d142-4be7-9431-5384c866384d">