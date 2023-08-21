package httpproxy

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Jigsaw-Code/outline-sdk/x/local-proxy/proxy"
	"github.com/go-httpproxy/httpproxy"
)

// NewConnectHandler returns a new ProxyHandler which implements http.Handler interface
func NewConnectHandler(d transport.StreamDialer, addr ...string) *ProxyHandler {
	a := ""
	if len(addr) > 0 {
		a = addr[0]
	}

	return &ProxyHandler{
		Proxy: httpproxy.Proxy{
			Rt: &http.Transport{
				TLSClientConfig: &tls.Config{},
				DialContext:     func(ctx context.Context, network, addr string) (net.Conn, error) { return d.Dial(ctx, addr) },
			},
		},
		address: a,
	}
}

// ProxyHandler defines parameters for running an HTTP ProxyHandler. It implements
// http.Handler interface for ListenAndServe function. If you need, you must
// set ProxyHandler struct before handling requests.
type ProxyHandler struct {
	httpproxy.Proxy

	address string
	l       net.Listener
	errC    chan error
}

var (
	_ http.Handler = (*ProxyHandler)(nil)
	_ proxy.Proxy  = (*ProxyHandler)(nil)
)

func (prx *ProxyHandler) GetAddr() string {
	return prx.address
}

func (prx *ProxyHandler) StartServer(addr string) (err error) {
	// if prx.address is empty listen on random localhost port save it to prx.address and start server
	if addr != "" {
		_, _, err := net.SplitHostPort(addr)
		if err != nil {
			return fmt.Errorf("invalid address: %w", err)
		}

		prx.address = addr
	}

	switch prx.address {
	case "":
		prx.l, err = net.Listen("tcp", "localhost:0")
		if err != nil {
			return fmt.Errorf("failed to listen on random localhost port: %w", err)
		}

		prx.address = prx.l.Addr().String()
	default:
		prx.l, err = net.Listen("tcp", prx.address)
		if err != nil {
			return fmt.Errorf("failed to listen on %s: %w", prx.address, err)
		}
	}

	go func() {
		if err = http.Serve(prx.l, prx); err != nil {
			prx.errC <- err
		}
	}()

	// sleep some time to check if server failed to start
	time.Sleep(100 * time.Millisecond)

	select {
	case err = <-prx.errC:
		return fmt.Errorf("failed to start server: %w", err)
	default:
		return nil
	}
}

func (prx *ProxyHandler) StopServer() error {
	return prx.l.Close()
}

func (prx *ProxyHandler) GetError() error {
	select {
	case err := <-prx.errC:
		return err
	default:
		return nil
	}
}
