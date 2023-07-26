# Outline VPN CLI (PoC)

The CLI interface of Outline VPN client for Linux.

## Usage

#### Standard

```
./OutlineCLI "<svr-ip>" "<svr-port>" "<svr-pass>"
```

#### Advanced (with Golang)

```
go run github.com/Jigsaw-Code/outline-internal-sdk/x/outline-cli@latest "<svr-ip>" "<svr-port>" "<svr-pass>"
```

### CLI Arguments

- `svr-ip`   : the outline server IP (for example, `111.111.111.111`)
- `svr-port` : the outline server port (for example, `21532`)
- `svr-pass` : the outline server password

## Build (for Developers)

We recommend to setup a [go workspace](https://go.dev/blog/get-familiar-with-workspaces) to build the code. Then use the following command to build the CLI (only support Linux):

```
cd outline-internal-sdk/x
go build -o OutlineCLI  -ldflags="-extldflags=-static" ./outline-cli
```
