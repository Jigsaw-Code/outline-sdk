module github.com/Jigsaw-Code/outline-internal-sdk/x/outline-connectivity

go 1.20

retract v0.0.0

require github.com/Jigsaw-Code/outline-internal-sdk v0.0.0

replace github.com/Jigsaw-Code/outline-internal-sdk => ../..

require (
	github.com/shadowsocks/go-shadowsocks2 v0.1.5 // indirect
	golang.org/x/crypto v0.7.0 // indirect
	golang.org/x/sys v0.6.0 // indirect
)
