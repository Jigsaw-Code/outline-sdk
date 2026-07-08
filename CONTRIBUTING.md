# How to contribute

We'd love to accept your patches and contributions to this project.

## Contribution process

If you don't know what to contribute, a good start is to go over the [issue tracker](https://github.com/OutlineFoundation/outline-sdk/issues).

For new features, it's best to share your idea first before going too deep into the implementation,
so we can align on the design.

* If there's a feature request open, share your proposal there.
* Otherwise, start with a discussion on [API Proposals](https://github.com/OutlineFoundation/outline-sdk/discussions/categories/api-proposals-and-ideas).

For bug fixes, you can send a PR directly.

### Code reviews

All submissions, including submissions by project members, require review. We
use GitHub pull requests for this purpose. Consult
[GitHub Help](https://help.github.com/articles/about-pull-requests/) for more
information on using pull requests.

## Repository structure

The repository has two Go modules:

* The root (`/`) module, where all the basic definitions and stable libraries live
* The `/x` module, where higher-level code, extensions and experimental libraries live.

New libraries should start in the `x/` module. We encourage you to create an example in `x/examples` or extend
one of the tools in `x/tools/` to demonstrate your feature.

Only low-level libraries that have been validated should move to the root module.

You cannot make atomic changes across module boundaries. If you need to change both the root and `x` modules
You need to first change root, merge, then you can refer to it in `x`.
Module `x` has a pinned version of the root module in its [`go.mod`](./x/go.mod) file.

To build and run code in the `x` module, you have to enter the `x` folder, or use `go -C x` flag from the repository root.
For example:

```sh
go run -C x ./tools/fetch https://ipinfo.io
```

Or

```sh
go -C x mod tidy
```

## Write Go Documentation

Writing and improving existing documentation is a good way to start with contributions.

The best way to ensure you got the Go doc formatting right is to visualize it.
To visualize the Go documentation you wrote, run:

```sh
go tool pkgsite -dev
```

Then open http://localhost:8080 on your browser. The `-dev` flag is optional and enables developer mode, reloading content on changes.

## Style

We use the standard Go style. Use `gofmt -w <path>` tool to make sure the style is correct.

## Cross-platform Development

See [docs/cross-platform.md](./docs/cross-platform.md) for how to build binaries for other platforms and run them during development (Android via adb, Linux via Podman, Windows via Wine).

## Testing

All new code must be accompanied by tests. Tests should be placed in `_test.go` files alongside the code they are testing.

### Running Tests

To run all tests in the repository, run the following commands from the root of the repository:

```sh
go test -race ./...
go test -C x -race ./...
```

This will run all tests except those that have external network dependencies.

### Network Dependant Tests

Some tests have external network dependencies. These tests are tagged with the `nettest` build tag and are not run by default. To run these tests, you must include the `-tags nettest` flag. Our CI runs these tests.

For example:

```sh
go test -v -race -tags nettest
```

### Benchmarks

To run benchmarks:

```sh
go test -race -bench '.' ./... -benchtime=100ms
go -C x test -race -bench '.' ./... -benchtime=100ms
```

### Continuous Integration (CI)

All pull requests are tested on our CI system. The CI runs all tests, including `nettest`s, on Linux, macOS, and Windows. It also runs tests on an Android emulator. You can see the CI configuration in [`.github/workflows/test.yml`](.github/workflows/test.yml).
