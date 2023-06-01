# Outline SDK (Under Development, DO NOT USE)

[![Build Status](https://github.com/Jigsaw-Code/outline-internal-sdk/actions/workflows/test.yml/badge.svg)](https://github.com/Jigsaw-Code/outline-internal-sdk/actions/workflows/test.yml?query=branch%3Amain)
[![Go Report Card](https://goreportcard.com/badge/github.com/Jigsaw-Code/outline-internal-sdk)](https://goreportcard.com/report/github.com/Jigsaw-Code/outline-internal-sdk)
[![Go Reference](https://pkg.go.dev/badge/github.com/Jigsaw-Code/outline-internal-sdk.svg)](https://pkg.go.dev/github.com/Jigsaw-Code/outline-internal-sdk)

[![Mattermost](https://badgen.net/badge/Mattermost/Outline%20Community/blue)](https://community.internetfreedomfestival.org/community/channels/outline-community)
[![Reddit](https://badgen.net/badge/Reddit/r%2Foutlinevpn/orange)](https://www.reddit.com/r/outlinevpn/)

This is the repository to hold the future Outline SDK as we develop it. The goal is to clean up and move reusable code from [outline-ss-server](https://github.com/Jigsaw-Code/outline-ss-server) and [outline-go-tun2socks](https://github.com/Jigsaw-Code/outline-go-tun2socks).

**WARNING: This code is not ready to be used by the public. There's no guarantee of stability.**

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
  - [x] Generic network primitives (TBD, something like a generic TUN device)
  - [x] Implementation based on go-tun2socks

- VPN API
  - [ ] VPN API for desktop (Linux, Windows, macOS)

# Cross-platform Development

## Building

In Go you can compile for other target operating system and architecture by specifying the `GOOS` and `GOARCH` environment variables and using the [`go build` command](https://pkg.go.dev/cmd/go#hdr-Compile_packages_and_dependencies). That only works if you are not using [Cgo](https://pkg.go.dev/cmd/cgo) to call C code.

<details>
  <summary>Examples</summary>

MacOS example:
```
% GOOS=darwin go build -C x -o ./bin/ ./outline-connectivity 
% file ./x/bin/outline-connectivity 
./x/bin/outline-connectivity: Mach-O 64-bit executable x86_64
```

Linux example:
```
% GOOS=linux go build -C x -o ./bin/ ./outline-connectivity 
% file ./x/bin/outline-connectivity                      
./x/bin/outline-connectivity: ELF 64-bit LSB executable, x86-64, version 1 (SYSV), statically linked, Go BuildID=n0WfUGLum4Y6OpYxZYuz/lbtEdv_kvyUCd3V_qOqb/CC_6GAQqdy_ebeYTdn99/Tk_G3WpBWi8vxqmIlIuU, with debug_info, not stripped
```

Windows example:
```
% GOOS=windows go build -C x -o ./bin/ ./outline-connectivity 
% file ./x/bin/outline-connectivity.exe 
./x/bin/outline-connectivity.exe: PE32+ executable (console) x86-64 (stripped to external PDB), for MS Windows
```
</details>

## Running Linux binaries

To run Linux binaries you can use a Linux container via [Podman](https://podman.io/).

### Set up podman
<details>
  <summary>Instructions</summary>

[Install Podman](https://podman.io/docs/installation) (once). On macOS:
```sh
brew install podman
```

Create the podman service VM (once) with the [`podman machine init` command](https://docs.podman.io/en/latest/markdown/podman-machine-init.1.html):
```sh
podman machine init
```

Start the VM with the [`podman machine start` command](https://docs.podman.io/en/latest/markdown/podman-machine-start.1.html), after every time it is stopped:
```sh
podman machine start
``` 

You can see the VM running with the [`podman machine list` command](https://docs.podman.io/en/latest/markdown/podman-machine-list.1.html):
```
% podman machine list
NAME                     VM TYPE     CREATED        LAST UP            CPUS        MEMORY      DISK SIZE
podman-machine-default*  qemu        3 minutes ago  Currently running  1           2.147GB     107.4GB
```

When you are done with development, you can stop the machine with the [`podman machine stop` command](https://docs.podman.io/en/latest/markdown/podman-machine-stop.1.html):
```sh
podman machine stop
```
</details>

### Run

The easiest way is to run a binary is to use the [`go run` command](https://pkg.go.dev/cmd/go#hdr-Compile_and_run_Go_program) directly with the `-exec` flag and our convenience tool `run_on_podman.sh`:
```sh
GOOS=linux go run -C x -exec "$(pwd)/run_on_podman.sh" ./outline-connectivity
```

It also works with the [`go test` command](https://pkg.go.dev/cmd/go#hdr-Test_packages):
```sh
GOOS=linux go test -exec "$(pwd)/run_on_podman.sh"  ./...
```

<details>
  <summary>Details and direct invocation</summary>

The `run_on_podman.sh` script uses the [`podman run` command](https://docs.podman.io/en/latest/markdown/podman-run.1.html) and a minimal ["distroless" container image](https://github.com/GoogleContainerTools/distroless) to run the binary you want:
```sh
podman run --arch $(uname -m) --rm -it -v "${bin}":/outline/bin gcr.io/distroless/static-debian11 /outline/bin "$@"
```

You can also use `podman run` directly to run a pre-built binary:
```
% podman run --rm -it -v ./x/bin:/outline gcr.io/distroless/static-debian11 /outline/outline-connectivity
Usage of /outline/outline-connectivity:
  -domain string
        Domain name to resolve in the test (default "example.com.")
  -key string
        Outline access key
  -proto string
        Comma-separated list of the protocols to test. Muse be "tcp", "udp", or a combination of them (default "tcp,udp")
  -resolver string
        Comma-separated list of addresses of DNS resolver to use for the test (default "8.8.8.8,2001:4860:4860::8888")
  -v    Enable debug output
```

Flags explanation:
- `--rm`: Remove container (and pod if created) after exit
- `-i` (interactive): Keep STDIN open even if not attached
- `-t` (tty): Allocate a pseudo-TTY for container
- `-v` (volume): Bind mount a volume into the container. Volume source will be on the server machine, not the client
</details>

## Running Windows binaries

To run Windows binaries you can use [Wine](https://en.wikipedia.org/wiki/Wine_(software)) to emulate a Windows environment.
This is not the same as a real Windows environment, so make sure you test on actual Windows machines.

### Install Wine

<details>
  <summary>Instructions</summary>

Follow the instructions at https://wiki.winehq.org/Download.

On macOS: 
```
brew tap homebrew/cask-versions
brew install --cask --no-quarantine wine-stable
```

After installation, `wine64` should be on your `PATH`. Check with `wine64 --version`:
```
wine64 --version
```

</details>

### Run

You can pass `wine64` as the `-exec` parameter in the `go` calls.

To build:

```sh
GOOS=windows go run -C x -exec "wine64" ./outline-connectivity
```

For tests:
```sh
GOOS=windows go test -exec "wine64"  ./...
```
