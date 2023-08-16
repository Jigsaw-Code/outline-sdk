package proxy

type Proxy interface {
	// GetAddr returns the address of the proxy server if it is configured to listen on a specific address.
	// If address is not configured and the server is not started, it returns an empty string.
	GetAddr() string
	// StartServer starts the proxy server. If no address is specified, it will listen on a random localhost port
	StartServer(addr ...string) error
	// StopServer stops the proxy server
	StopServer() error
	// GetError returns the error that caused the proxy server to stop or nil if the server is still running or not started
	GetError() error
}
