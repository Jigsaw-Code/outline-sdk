package fullstack_app

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Jigsaw-Code/outline-sdk/transport/shadowsocks"
	"github.com/Jigsaw-Code/outline-sdk/x/mobileproxy"
	"github.com/a-h/templ"
)

var APP_ADDRESS = ":8080"
var PROXY_ADDRESS = ":8181"
var SYSTEM_ADDRESS = ":8282"

var PROXY_DISCONNECT_TIMEOUT = 5
var SYSTEM_TUNNEL_ENDPOINT = "${SYSTEM_ADDRESS}/tunnel/${PROXY_ADDRESS}"

func Start() {
	vpn := VPNController{}

	http.Handle("/", templ.Handler(serverView()))
	http.Handle("/connection/ss://<my-shadowsocks-key>", http.HandlerFunc(vpn.handleConnectionEndpoint))
	http.Handle("/disconnection/ss://<my-shadowsocks-key>", http.HandlerFunc(vpn.handleDisconnectionEndpoint))

	http.ListenAndServe(APP_ADDRESS, nil)
}

type VPNController struct {
	proxy  *mobileproxy.Proxy
	tunnel io.Reader
}

func (vpn *VPNController) handleConnectionEndpoint(responseWriter http.ResponseWriter, request *http.Request) {
	if vpn.proxy != nil {
		http.Error(responseWriter, "already connected", http.StatusConflict)
		return
	}

	endpoint, encryptionKey := parseStaticShadowsocksAccessKey(request.URL.Path)

	proxyDialer, err := shadowsocks.NewStreamDialer(endpoint, encryptionKey)

	if err != nil {
		http.Error(responseWriter, err.Error(), http.StatusInternalServerError)
		return
	}

	proxy, err := mobileproxy.RunProxy(PROXY_ADDRESS, &mobileproxy.StreamDialer{proxyDialer})

	if err != nil {
		http.Error(responseWriter, err.Error(), http.StatusInternalServerError)
		return
	}

	vpn.proxy = proxy

	// TODO: why this code isn't very good at all!
	tunnel, err := os.Open("tunnel.sock")
	vpn.tunnel = tunnel

	// TODO: implement system vpn tunnel service
	// => POST /tunnel/URL forward all non-local traffic to that URL
	// => DELETE /tunnel/URL gracefully shuts down the tunnel

	// => apple box uses VPN API
	// => kotlin box uses SSH tunnel (for now)

	// ! these boxes will be reusable across VPN apps !
	http.NewRequest("POST", SYSTEM_TUNNEL_ENDPOINT, vpn.tunnel)

	writeTemplate(disconnectionButton, responseWriter)
}

func (vpn *VPNController) handleDisconnectionEndpoint(responseWriter http.ResponseWriter, _ *http.Request) {
	http.NewRequest("DELETE", SYSTEM_TUNNEL_ENDPOINT, vpn.tunnel)

	vpn.proxy.Stop(PROXY_DISCONNECT_TIMEOUT)
	vpn.proxy = nil
	vpn.tunnel.Close()
	writeTemplate(connectionButton, responseWriter)
}

func parseStaticShadowsocksAccessKey(staticAccessKey string) (transport.StreamEndpoint, *shadowsocks.EncryptionKey) {
	vpnServerKeyAndHost := strings.Split(strings.TrimPrefix("ss://", staticAccessKey), "@")

	encodedEncryptionKey := vpnServerKeyAndHost[0]
	decodedEncryptionKey, err := base64.StdEncoding.DecodeString(encodedEncryptionKey)

	if err != nil {
		// TODO: do something
	}

	encryptionKeyCipherAndSecret := strings.Split(string(decodedEncryptionKey), ":")
	encryptionKey, err := shadowsocks.NewEncryptionKey(encryptionKeyCipherAndSecret[0], encryptionKeyCipherAndSecret[1])

	if err != nil {
		// TODO: do something
	}

	vpnHostUrlString := vpnServerKeyAndHost[1]
	vpnHostUrl, err := url.Parse(vpnHostUrlString)

	if err != nil {
		// TODO: do something
	}

	vpnHostStreamEndpoint := transport.StreamDialerEndpoint{
		Address: vpnHostUrl.Host,
		Dialer:  &transport.TCPDialer{},
	}

	return vpnHostStreamEndpoint, encryptionKey
}

// func resolveDynamicAccessKey(dynamicAccessKey string) (transport.StreamEndpoint, *shadowsocks.EncryptionKey) {
// 		mobileproxy.NewSmartStreamDialer(dynamicAccessKey, <config>)
// }

// TODO: template arguments
func writeTemplate(template func() templ.Component, writer io.Writer) {
	component := template()
	component.Render(context.Background(), writer)
}
