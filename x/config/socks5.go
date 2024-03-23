package config

import (
	"net/url"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Jigsaw-Code/outline-sdk/transport/socks5"
)

func newSOCKS5StreamDialerFromURL(innerDialer transport.StreamDialer, configURL *url.URL) (transport.StreamDialer, error) {
	endpoint := transport.StreamDialerEndpoint{Dialer: innerDialer, Address: configURL.Host}
	dialer, err := socks5.NewStreamDialer(&endpoint)
	if err != nil {
		return nil, err
	}
	userInfo := configURL.User
	if userInfo != nil {
		username := userInfo.Username()
		password, _ := userInfo.Password()
		err := dialer.SetCredentials([]byte(username), []byte(password))
		if err != nil {
			return nil, err
		}
	}
	return dialer, nil
}
