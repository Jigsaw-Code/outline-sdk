# How to contribute

We'd love to accept your patches and contributions to this project.

## Before you begin

### Sign our Contributor License Agreement

Contributions to this project must be accompanied by a
[Contributor License Agreement](https://cla.developers.google.com/about) (CLA).
You (or your employer) retain the copyright to your contribution; this simply
gives us permission to use and redistribute your contributions as part of the
project.

If you or your current employer have already signed the Google CLA (even if it
was for a different project), you probably don't need to do it again.

Visit <https://cla.developers.google.com/> to see your current agreements or to
sign a new one.

### Review our community guidelines

This project follows
[Google's Open Source Community Guidelines](https://opensource.google/conduct/).

## Contribution process

### Code reviews

All submissions, including submissions by project members, require review. We
use GitHub pull requests for this purpose. Consult
[GitHub Help](https://help.github.com/articles/about-pull-requests/) for more
information on using pull requests.

# Go Documentation

The best way to ensure you got the Go doc formatting right is to visualize it.
To visualize the Go documentation you wrote, run:

```sh
go run golang.org/x/pkgsite/cmd/pkgsite@latest
```

Then open http://localhost:8080 on your browser.

# Cross-platform Development

## Building

In Go you can compile for other target operating system and architecture by specifying the `GOOS` and `GOARCH` environment variables and using the [`go build` command](https://pkg.go.dev/cmd/go#hdr-Compile_packages_and_dependencies). That only works if you are not using [Cgo](https://pkg.go.dev/cmd/cgo) to call C code.

<details>
  <summary>Examples</summary>

MacOS example:

```console
% GOOS=darwin go build -C x -o ./bin/ ./examples/test-connectivity 
% file ./x/bin/test-connectivity 
./x/bin/test-connectivity: Mach-O 64-bit executable x86_64
```

Linux example:

```console
% GOOS=linux go build -C x -o ./bin/ ./examples/test-connectivity 
% file ./x/bin/test-connectivity                      
./x/bin/test-connectivity: ELF 64-bit LSB executable, x86-64, version 1 (SYSV), statically linked, Go BuildID=n0WfUGLum4Y6OpYxZYuz/lbtEdv_kvyUCd3V_qOqb/CC_6GAQqdy_ebeYTdn99/Tk_G3WpBWi8vxqmIlIuU, with debug_info, not stripped
```

Windows example:

```console
% GOOS=windows go build -C x -o ./bin/ ./examples/test-connectivity 
% file ./x/bin/test-connectivity.exe 
./x/bin/test-connectivity.exe: PE32+ executable (console) x86-64 (stripped to external PDB), for MS Windows
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

```console
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
GOOS=linux go run -C x -exec "$(pwd)/run_on_podman.sh" ./examples/test-connectivity
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

```console
% podman run --rm -it -v ./x/bin:/outline gcr.io/distroless/static-debian11 /outline/test-connectivity
Usage of /outline/test-connectivity:
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

```sh
brew tap homebrew/cask-versions
brew install --cask --no-quarantine wine-stable
```

After installation, `wine64` should be on your `PATH`. Check with `wine64 --version`:

```sh
wine64 --version
```

</details>

### Run

You can pass `wine64` as the `-exec` parameter in the `go` calls.

To build:

```sh
GOOS=windows go run -C x -exec "wine64" ./examples/test-connectivity
```

For tests:

```sh
GOOS=windows go test -exec "wine64"  ./...
```

# Tests with external network dependencies

Some tests are implemented talking to external services. That's undesirable, but convenient.
We started tagging them with the `nettest` tag, so they don't run by default. To run them, you need to specify `-tags nettest`, as done in our CI.
For example:

```sh
go test -v -race -bench '.' ./... -benchtime=100ms -tags nettest
```
