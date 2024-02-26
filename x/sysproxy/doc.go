/*
Package sysproxy provides a simple interface to set/unset system-wide proxy settings.

# Platform Support
Currently this package supports desktop platforms only. The following platforms are supported:
- macOS
- Linux (Gnome is the only supported desktop environment for now.)
- Windows

# Usage

- To set up system-wide proxy settings, use the `SetProxy` function. This function takes two arguments: the IP address and the port of the proxy server.
`SetProxy(ip string, port string) error`

- To unset system-wide proxy settings, use the `UnsetProxy` function.
`UnsetProxy() error`

You need to also clean up the proxy settings when you are done with the proxy server. This is important because the proxy settings will remain in place even after the proxy server is stopped.
The following example demonstrates how to safely unset proxy upon program termination:

```go
func main () {
	// Create channels to receive signals
	done := make(chan bool, 1)
	sigs := make(chan os.Signal, 1)
	sysproxy.SafeCloseProxy(done, sigs)

	// Your code here

	// Send a signal to the program to terminate
	sigs<-syscall.SIGTERM
	// Wait for the program to terminate
	<-done
}

```
*/

package sysproxy
