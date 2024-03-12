package app

import (
	"context"
	"io"
	"net/http"
	"os"

	"github.com/Jigsaw-Code/outline-sdk/x/config"
	"github.com/Jigsaw-Code/outline-sdk/x/mobileproxy"
	"github.com/a-h/templ"
)

var appAddress = ":8080"
var proxyAddress = ":8181"
var systemAddress = ":8282"

var proxyDisconnectTimeoutSecond = 5
var systemTunnelEndpoint = "${systemAddress}/tunnel/${proxyAddress}"

func Start() {
	vpn := VPNController{}

	http.Handle("/", templ.Handler(serverView()))
	http.Handle("/connection/ss://<my-shadowsocks-key>", http.HandlerFunc(vpn.handleConnectionEndpoint))
	http.Handle("/disconnection/ss://<my-shadowsocks-key>", http.HandlerFunc(vpn.handleDisconnectionEndpoint))

	http.ListenAndServe(appAddress, nil)
}

type VPNController struct {
	proxy  *mobileproxy.Proxy
	tunnel io.Reader
}

func (vpn *VPNController) handleConnectionEndpoint(responseWriter http.ResponseWriter, request *http.Request) {
	proxyDialer, err := config.NewStreamDialer(request.URL.Path)

	if err != nil {
		http.Error(responseWriter, err.Error(), http.StatusInternalServerError)
		return
	}

	proxy, err := mobileproxy.RunProxy(proxyAddress, &mobileproxy.StreamDialer{proxyDialer})

	if err != nil {
		http.Error(responseWriter, err.Error(), http.StatusInternalServerError)
		return
	}

	vpn.proxy = proxy

	// TODO: why this code isn't very good at all!
	tunnel, err := os.Open("tunnel.sock")

	if err != nil {
		// do something
	}

	vpn.tunnel = tunnel

	// TODO: implement system vpn tunnel service
	// => POST /tunnel/URL forward all non-local traffic to that URL
	// => DELETE /tunnel/URL gracefully shuts down the tunnel

	// => apple box uses VPN API
	// => kotlin box uses SSH tunnel (for now)

	// ! these boxes will be reusable across VPN apps !
	http.NewRequest("POST", systemTunnelEndpoint, vpn.tunnel)

	writeTemplate(disconnectionButton, responseWriter)
}

func (vpn *VPNController) handleDisconnectionEndpoint(responseWriter http.ResponseWriter, _ *http.Request) {
	http.NewRequest("DELETE", systemTunnelEndpoint, vpn.tunnel)

	vpn.proxy.Stop(proxyDisconnectTimeoutSecond)

	writeTemplate(connectionButton, responseWriter)
}

// TODO: template arguments
func writeTemplate(template func() templ.Component, writer io.Writer) {
	component := template()
	component.Render(context.Background(), writer)
}
