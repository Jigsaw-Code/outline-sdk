package httpproxy

import (
	"encoding/base64"
	"strings"
	"errors"
	"fmt"
	"log"
)

func BasicAuth(usersPasswords map[string]string) func(ctx *Context, authType, authData string) (bool, error) {
	return func(_ *Context, authType, authData string) (bool, error) {
		if authType != "Basic" {
			return false, errors.New("unsupported auth type")
		}

		userPassRaw, err := base64.StdEncoding.DecodeString(authData)
		if err != nil {
			return false, fmt.Errorf("invalid auth data: %w", err)
		}

		userPass := strings.SplitN(string(userPassRaw), ":", 2)
		if len(userPass) < 2 {
			return false, errors.New("invalid auth data: semicolon ':' not found")
		}

		username := userPass[0]
		password := userPass[1]

		if pwd := usersPasswords[username]; pwd != "" && pwd == password {
			return true, nil
		}

		return false, nil
	}
}

func OnError(_ *Context, where string, err, opErr error) {
	if opErr != nil {
		log.Printf("HTTP proxy error: %s: %v, %v\n", where, err, opErr)
	} else {
		log.Printf("HTTP proxy error: %s: %v\n", where, err)
	}
}
