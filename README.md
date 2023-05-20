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
  - [ ] Generic network primitives (TBD, something like a generic TUN device)
  - [ ] Implementation based on go-tun2socks

- VPN API
  - [ ] VPN API for desktop (Linux, Windows, macOS)

# Cross-platform Development

## Building

In Go, you can compile for other target operating system and architecture by specifying the `GOOS` and `GOARCH` environment variables.

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

To run Linux binaries we use a Linux container via [Podman](https://podman.io/).

## Set up podman
<details>
  <summary>Instructions</summary>

[Install Podman](https://podman.io/docs/installation)
On macOS (once):
```sh
brew install podman
```

Create the podman service VM (once):
```sh
podman machine init
```

Start the VM (after every time it is stopped):
```sh
podman machine start
``` 

You can see it running with `podman machine list`:
```
% podman machine list
NAME                     VM TYPE     CREATED        LAST UP            CPUS        MEMORY      DISK SIZE
podman-machine-default*  qemu        3 minutes ago  Currently running  1           2.147GB     107.4GB
```

When you are done with development, you can stop the machine:
```sh
podman machine stop
```
</details>

## Run

The easiest way is to run using `go run` directly with the `--exec` flag and our convenience tool `run_on_podman.sh`:
```sh
GOOS=linux go run -C x -exec "$(pwd)/run_on_podman.sh" ./outline-connectivity
```

It also works for tests:
```sh
GOOS=linux go test -exec "$(pwd)/run_on_podman.sh"  ./...
```

<details>
  <summary>Details and direct invocation</summary>

The `run_on_podman.sh` script uses `podman run` and the minimal [Alpine Linux](https://en.wikipedia.org/wiki/Alpine_Linux) to run the binary you want:
```sh
podman run --rm -it -v "${bin}":/outline/bin alpine /outline/bin "$@"
```

You can also use `podman` directly to run a pre-built binary:
```
% podman run --rm -it -v ./x/bin:/outline alpine /outline/outline-connectivity
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