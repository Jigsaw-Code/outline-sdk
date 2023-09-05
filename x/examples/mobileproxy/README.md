# App Proxy Library

This package illustrates the use Go Mobile to generarate a mobile library to run a local proxy and configure your app networking libraries.

1. Build the Go Mobile binaries with [`go build`](https://pkg.go.dev/cmd/go#hdr-Compile_packages_and_dependencies)

```bash
go build -o ./out/ golang.org/x/mobile/cmd/gomobile golang.org/x/mobile/cmd/gobind
```

2. Build the iOS and Android libraries with [`gomobile bind`](https://pkg.go.dev/golang.org/x/mobile/cmd/gomobile#hdr-Build_a_library_for_Android_and_iOS)

```bash
PATH="$PATH:$(pwd)/out" $(pwd)/out/gomobile bind -target=ios -o "$(pwd)/out/LocalProxy.xcframework" github.com/Jigsaw-Code/outline-sdk/x/appproxy
PATH="$PATH:$(pwd)/out" $(pwd)/out/gomobile bind -target=android -o "$(pwd)/out/LocalProxy.aar" github.com/Jigsaw-Code/outline-sdk/x/appproxy
```

Note: Gomobile expects gobind to be in the PATH, that's why we need to prebuild it, and set up the PATH accordingly.

3. To clean up:

```bash
rm -rf ./out/
```
