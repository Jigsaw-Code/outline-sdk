package httpproxy

import (
	"crypto/tls"
	"net/http"
	"github.com/Jigsaw-Code/outline-sdk/transport"
	"net"
	"context"
	"fmt"
	"time"
)

// NewConnectHandler returns a new ProxyHandler which implements http.Handler interface
func NewConnectHandler(d transport.StreamDialer, addr ...string) *ProxyHandler {
	a := ""
	if len(addr) > 0 {
		a = addr[0]
	}

	return &ProxyHandler{
		Rt: &http.Transport{
			TLSClientConfig: &tls.Config{},
			DialContext:     func(ctx context.Context, network, addr string) (net.Conn, error) { return d.Dial(ctx, addr) },
		},
		address: a,
	}
}

// NewConnectHandlerWCert returns a new ProxyHandler which implements http.Handler interface with custom certificate
func NewConnectHandlerWCert(d transport.StreamDialer, caCert, caKey []byte, addr ...string) (*ProxyHandler, error) {
	a := ""
	if len(addr) > 0 {
		a = addr[0]
	}

	prx := &ProxyHandler{
		Rt: &http.Transport{
			TLSClientConfig: &tls.Config{},
			DialContext:     func(ctx context.Context, network, addr string) (net.Conn, error) { return d.Dial(ctx, addr) },
		},
		Signer:  NewCaSignerCache(1024),
		address: a,
	}

	var err error
	prx.Ca, err = tls.X509KeyPair(caCert, caKey)
	if err != nil {
		return nil, err
	}

	prx.Signer.Ca = &prx.Ca

	return prx, nil
}

// ProxyHandler defines parameters for running an HTTP ProxyHandler. It implements
// http.Handler interface for ListenAndServe function. If you need, you must
// set ProxyHandler struct before handling requests.
type ProxyHandler struct {
	// RoundTripper interface to obtain remote response.
	// By default, it uses &http.Transport{}.
	Rt http.RoundTripper

	// Certificate key pair.
	Ca tls.Certificate

	// Error callback.
	OnError func(ctx *Context, where string, err, opErr error)

	// Auth callback. If you need authentication, set this callback.
	// authType and authData are from Proxy-Authorization header.
	// If it returns true, authentication succeeded.
	// Return error if authentication failed or authType is not supported.
	// Please, don't forget to set AuthType field.
	OnAuth func(ctx *Context, authType, authData string) (bool, error)

	// HTTP Authentication type. If it's not specified (""), uses "Basic".
	// By default, "".
	AuthType string

	Signer *CaSigner

	address string
	l       net.Listener
	errC    chan error
}

// ServeHTTP implements golang http.Handler interface.
func (prx *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := &Context{prx: prx}

	defer func() {
		rec := recover()
		if rec != nil {
			if err, ok := rec.(error); ok && prx.OnError != nil {
				prx.OnError(ctx, "ServeHTTP", ErrPanic, err)
			}

			prx.errC <- fmt.Errorf("panic: %+v", rec)
		}
	}()

	if r.Body != nil {
		defer r.Body.Close()
	}

	if err := ctx.doAccept(w, r); err != nil {
		return
	}

	if err := ctx.doAuth(w, r); err != nil {
		return
	}

	r.Header.Del("Proxy-Connection")
	r.Header.Del("Proxy-Authenticate")
	r.Header.Del("Proxy-Authorization")

	if err := ctx.doConnect(w, r); err != nil {
		return
	}

	if w == nil || r == nil {
		return
	}

	if err := ctx.doRequest(w, r); err != nil {
		return
	}

	_ = ctx.doResponse(w, r)
}

func (prx *ProxyHandler) GetAddr() string {
	return prx.address
}

func (prx *ProxyHandler) StartServer(addr ...string) (err error) {
	// if prx.address is empty listen on random localhost port save it to prx.address and start server
	if len(addr) > 0 {
		_, _, err := net.SplitHostPort(addr[0])
		if err != nil {
			return fmt.Errorf("invalid address: %w", err)
		}

		prx.address = addr[0]
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
