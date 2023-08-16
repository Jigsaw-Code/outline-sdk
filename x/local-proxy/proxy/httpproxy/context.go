package httpproxy

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
)

// Context keeps context of each proxy request.
type Context struct {
	// Context to be used by user callbacks
	context.Context

	// Pointer of ProxyHandler struct handled this context.
	prx *ProxyHandler

	// Original ProxyHandler request.
	req *http.Request

	// Original ProxyHandler request, if proxy request method is CONNECT.
	connectReq *http.Request

	// Remote host, if proxy request method is CONNECT.
	connectHost string
}

func (ctx *Context) onAuth(authType, authData string) (bool, error) {
	defer func() {
		if err, ok := recover().(error); ok {
			ctx.doError("Auth", ErrPanic, err)
		}
	}()

	return ctx.prx.OnAuth(ctx, authType, authData)
}

func (ctx *Context) doError(where string, err, opErr error) {
	if ctx.prx.OnError == nil {
		return
	}
	ctx.prx.OnError(ctx, where, err, opErr)
}

func (ctx *Context) doAccept(w http.ResponseWriter, r *http.Request) error {
	ctx.req = r
	if !r.ProtoAtLeast(1, 0) || r.ProtoAtLeast(2, 0) {
		ctx.doError("Accept", ErrNotSupportHTTPVer, nil)

		return ErrNotSupportHTTPVer
	}

	return nil
}

func (ctx *Context) doAuth(w http.ResponseWriter, r *http.Request) error {
	if r.Method != "CONNECT" && !r.URL.IsAbs() {
		return nil
	}

	if ctx.prx.OnAuth == nil {
		return nil
	}

	prxAuthType := ctx.prx.AuthType
	if prxAuthType == "" {
		prxAuthType = "Basic"
	}

	var respBody = http.StatusText(http.StatusProxyAuthRequired)

	authParts := strings.SplitN(r.Header.Get("Proxy-Authorization"), " ", 2)
	if len(authParts) >= 2 {
		authType := authParts[0]
		authData := authParts[1]

		if authorized, err := ctx.onAuth(authType, authData); err == nil {
			if authorized {
				return nil
			} else {
				respBody += " [Unauthorized]"
			}
		}
	}

	err := ServeInMemory(w, http.StatusProxyAuthRequired,
		map[string][]string{"Proxy-Authenticate": {prxAuthType}},
		[]byte(respBody),
	)
	if err != nil && !isConnectionClosed(err) {
		ctx.doError("Auth", ErrResponseWrite, err)

		return err
	}

	return nil
}

func (ctx *Context) doConnect(w http.ResponseWriter, r *http.Request) error {
	if r.Method != "CONNECT" {
		return nil
	}

	hij, ok := w.(http.Hijacker)
	if !ok {
		ctx.doError("Connect", ErrNotSupportHijacking, nil)

		return ErrNotSupportHijacking
	}

	conn, _, err := hij.Hijack()
	if err != nil {
		ctx.doError("Connect", ErrNotSupportHijacking, err)

		return ErrNotSupportHijacking
	}

	hijConn := conn
	ctx.connectReq = r
	ctx.connectHost = addDefaultPortIfEmpty(r.URL.Host)

	conn, err = net.Dial("tcp", ctx.connectHost)
	if err != nil {
		hijConn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
		hijConn.Close()
		ctx.doError("Connect", ErrRemoteConnect, err)

		return ErrRemoteConnect
	}

	remoteConn := conn.(*net.TCPConn)
	if _, err := hijConn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n")); err != nil {
		hijConn.Close()
		remoteConn.Close()

		if !isConnectionClosed(err) {
			ctx.doError("Connect", ErrResponseWrite, err)
		}

		return ErrResponseWrite
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		defer func() {
			e := recover()
			err, ok := e.(error)
			if !ok {
				return
			}

			hijConn.Close()
			remoteConn.Close()

			if !isConnectionClosed(err) {
				ctx.doError("Connect", ErrRequestRead, err)
			}
		}()

		_, err := io.Copy(remoteConn, hijConn)
		if err != nil {
			panic(err)
		}

		remoteConn.CloseWrite()
		if c, ok := hijConn.(*net.TCPConn); ok {
			c.CloseRead()
		}
	}()

	go func() {
		defer wg.Done()

		defer func() {
			e := recover()
			err, ok := e.(error)
			if !ok {
				return
			}
			hijConn.Close()
			remoteConn.Close()
			if !isConnectionClosed(err) {
				ctx.doError("Connect", ErrResponseWrite, err)
			}
		}()

		_, err := io.Copy(hijConn, remoteConn)
		if err != nil {
			panic(err)
		}

		remoteConn.CloseRead()
		if c, ok := hijConn.(*net.TCPConn); ok {
			c.CloseWrite()
		}
	}()

	wg.Wait()
	hijConn.Close()
	remoteConn.Close()

	return ErrProxyConnectionClosed
}

func (ctx *Context) doRequest(w http.ResponseWriter, r *http.Request) error {
	if r.URL.IsAbs() {
		r.RequestURI = r.URL.String()

		return nil
	}

	err := ServeInMemory(w, 500, nil, []byte("This is a proxy server. Does not respond to non-proxy requests."))
	if err != nil && !isConnectionClosed(err) {
		ctx.doError("Request", ErrResponseWrite, err)
	}

	return ErrNonProxyRequest
}

func (ctx *Context) doResponse(w http.ResponseWriter, r *http.Request) error {
	resp, err := ctx.prx.Rt.RoundTrip(r)
	if err != nil {
		if err != context.Canceled && !isConnectionClosed(err) {
			ctx.doError("Response", ErrRoundTrip, err)
		}

		err := ServeInMemory(w, 404, nil, nil)
		if err != nil && !isConnectionClosed(err) {
			ctx.doError("Response", ErrResponseWrite, err)
		}

		return err
	}

	resp.Request = r
	resp.TransferEncoding = nil

	err = ServeResponse(w, resp)
	if err != nil && !isConnectionClosed(err) {
		ctx.doError("Response", ErrResponseWrite, err)
	}

	return err
}
