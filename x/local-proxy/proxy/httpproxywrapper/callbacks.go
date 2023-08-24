package httpproxy

import (
	"log"

	"github.com/go-httpproxy/httpproxy"
)

func BasicAuth(usersPasswords map[string]string) func(ctx *httpproxy.Context, authType, user, pass string) bool {
	return func(_ *httpproxy.Context, authType, user, pass string) bool {
		if authType != "Basic" {
			return false
		}

		if pwd := usersPasswords[user]; pwd != "" && pwd == pass {
			return true
		}

		return false
	}
}

func OnError(_ *httpproxy.Context, where string, err *httpproxy.Error, opErr error) {
	if opErr != nil {
		log.Printf("HTTP proxy error: %s: %v, %v\n", where, err, opErr)
	} else {
		log.Printf("HTTP proxy error: %s: %v\n", where, err)
	}
}
