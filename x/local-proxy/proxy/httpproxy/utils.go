package httpproxy

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// InMemoryResponse creates new HTTP response given arguments.
func InMemoryResponse(code int, header http.Header, body []byte) *http.Response {
	if header == nil {
		header = make(http.Header)
	}

	st := http.StatusText(code)
	if st != "" {
		st = " " + st
	}

	var bodyReadCloser io.ReadCloser
	var bodyContentLength = int64(0)
	if body != nil {
		bodyReadCloser = io.NopCloser(bytes.NewBuffer(body))
		bodyContentLength = int64(len(body))

	}

	return &http.Response{
		Status:        fmt.Sprintf("%d%s", code, st),
		StatusCode:    code,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        header,
		Body:          bodyReadCloser,
		ContentLength: bodyContentLength,
	}
}

// ServeResponse serves HTTP response to http.ResponseWriter.
func ServeResponse(w http.ResponseWriter, resp *http.Response) error {
	if resp.Body != nil {
		defer resp.Body.Close()
	}

	h := w.Header()
	for k, v := range resp.Header {
		for _, v1 := range v {
			h.Add(k, v1)
		}
	}

	if h.Get("Date") == "" {
		h.Set("Date", time.Now().UTC().Format("Mon, 2 Jan 2006 15:04:05")+" GMT")
	}

	if h.Get("Content-Type") == "" && resp.ContentLength != 0 {
		h.Set("Content-Type", "text/plain; charset=utf-8")
	}

	if resp.ContentLength >= 0 {
		h.Set("Content-Length", strconv.FormatInt(resp.ContentLength, 10))
	} else {
		h.Del("Content-Length")
	}

	h.Del("Transfer-Encoding")
	te := ""

	if len(resp.TransferEncoding) > 0 {
		if len(resp.TransferEncoding) > 1 {
			return ErrUnsupportedTransferEncoding
		}
		te = resp.TransferEncoding[0]
	}
	h.Del("Connection")

	clientConnection := ""
	if resp.Request != nil {
		clientConnection = resp.Request.Header.Get("Connection")
	}

	switch clientConnection {
	case "close":
		h.Set("Connection", "close")
	case "keep-alive":
		if h.Get("Content-Length") != "" || te == "chunked" {
			h.Set("Connection", "keep-alive")
		} else {
			h.Set("Connection", "close")
		}
	default:
		if te == "chunked" {
			h.Set("Connection", "close")
		}
	}

	switch te {
	case "":
		w.WriteHeader(resp.StatusCode)
		if resp.Body != nil {
			if _, err := io.Copy(w, resp.Body); err != nil {
				return err
			}
		}

	case "chunked":
		h.Set("Transfer-Encoding", "chunked")
		w.WriteHeader(resp.StatusCode)
		w2 := httputil.NewChunkedWriter(w)
		if resp.Body != nil {
			if _, err := io.Copy(w2, resp.Body); err != nil {
				return err
			}
		}

		if err := w2.Close(); err != nil {
			return err
		}

		if _, err := w.Write([]byte("\r\n")); err != nil {
			return err
		}

	default:
		return ErrUnsupportedTransferEncoding
	}

	return nil
}

// ServeInMemory serves HTTP response given arguments to http.ResponseWriter.
func ServeInMemory(w http.ResponseWriter, code int, header http.Header, body []byte) error {
	return ServeResponse(w, InMemoryResponse(code, header, body))
}

var hasPort = regexp.MustCompile(`:\d+$`)

func addDefaultPortIfEmpty(h string) string {
	if !hasPort.MatchString(h) {
		return h + ":80"
	}

	return h
}

func stripPort(s string) string {
	ix := strings.IndexRune(s, ':')
	if ix == -1 {
		return s
	}
	return s[:ix]
}
