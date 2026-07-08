# AGENTS.md

This file provides guidance to AI coding agents working with code in this repository.

## Module structure

Two decoupled Go modules, no go.work:

- Root `golang.getoutline.org/sdk` — stable core (`transport/`, `dns/`, `network/`). The import path is a vanity path; the repo lives on GitHub but code must import `golang.getoutline.org/sdk`, not `github.com/Jigsaw-Code/outline-sdk`.
- `x/` (`golang.getoutline.org/sdk/x`) — higher-level code, extensions, and experimental libraries, with no stability guarantees. It is a separate module so its dependencies don't pollute the root module, and because code that builds on the SDK cannot live in the root without creating circular dependencies. New libraries start here; only validated low-level libraries graduate to root.

`x/go.mod` pins a **released** version of the root module. Never add a local `replace`: a change spanning both modules requires changing root, releasing it, then bumping the pin in `x/`. Plan work accordingly — cross-module changes cannot land atomically.

## Commands

Run `x/`-module commands from the repo root with `-C x`.

- Test: `go test -race ./...` and `go test -C x -race ./...`
- Network-dependent tests are behind a build tag: add `-tags nettest` (CI runs `go test -tags nettest -race -bench '.' ./... -benchtime=100ms` on both modules)
- Lint (both modules): `go tool staticcheck ./...` and `go -C x tool staticcheck ./...`
- License check (both modules): `go tool go-licenses check --ignore=golang.org/x --allowed_licenses=Apache-2.0,Apache-3,BSD-3-Clause,BSD-4-Clause,CC0-1.0,MIT ./...`. Dependencies must use permissive ("notice"-type) licenses compatible with the project's Apache-2.0 license — no copyleft or restricted licenses.
- Godoc preview: `go tool pkgsite -dev` (serves on :8080)

Tools are declared via `tool` directives in go.mod — no separate installs.

## Gotchas

- The `psiphon` build tag (`x/psiphon/`, `x/mobileproxy/psiphon/`) is currently broken on Go 1.25 (psiphon-tls incompatibility, issue #513) and disabled in CI. Don't try to fix psiphon build failures in unrelated PRs.
- gomobile: `gomobile bind` must target a wrapper package written for binding (like `x/mobileproxy`), never SDK packages directly. Mobile builds: `task build:mobileproxy` in `x/` (see `x/Taskfile.yml`).
- If the vanity import path fails to resolve, set `GOPROXY=https://proxy.golang.org`.

## Workflow

- New features and API changes should go through a [GitHub "API Proposals" discussion](https://github.com/OutlineFoundation/outline-sdk/discussions/categories/api-proposals-and-ideas) before implementation; bug fixes can be direct PRs.
- All new code needs tests in `_test.go` files alongside the code.
- Commit messages use conventional commits with the package as scope: `fix(packetrelay): …`, `test(x): …`, `docs(cli): …`.
- See [docs/cross-platform.md](./docs/cross-platform.md) for building and running on other platforms (`run_on_android.sh`, `run_on_podman.sh`, wine), and [CONTRIBUTING.md](./CONTRIBUTING.md) for the contribution process and the Google CLA requirement.
