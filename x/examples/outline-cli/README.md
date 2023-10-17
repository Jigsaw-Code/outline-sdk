# OutlineVPN CLI

The CLI interface of OutlineVPN client for Linux.

## Usage

#### Standard

```
./outline-cli -transport "ss://<outline-server-access-key>"
```

#### Advanced (with Golang)

```
go run github.com/Jigsaw-Code/outline-sdk/x/examples/outline-cli@latest -transport "ss://<outline-server-access-key>"
```

### Arguments

- `-transport` : the Outline server access key from the service provider, it should start with "ss://"

## Build (for Developers)

We recommend to setup a [go workspace](https://go.dev/blog/get-familiar-with-workspaces) to build the code. Then use the following command to build the CLI (only support Linux):

```
cd outline-sdk/x/examples/
go build -o outline-cli  -ldflags="-extldflags=-static" ./outline-cli
```
