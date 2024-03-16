// Copyright 2024 Jigsaw Operations LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build darwin && !ios

package sysproxy

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

type ProxyType string

const (
	proxyTypeHTTP  ProxyType = "web"
	proxyTypeHTTPS ProxyType = "secureweb"
	proxyTypeSOCKS ProxyType = "socks"
)

var (
	backupHTTPSettings  *ProxySettings
	backupHTTPSSettings *ProxySettings
	backupSOCKSSettings *ProxySettings
)

type ProxySettings struct {
	host string
	port string
}

func SetWebProxy(host string, port string) error {
	// Get the active network interface
	activeInterface, err := getActiveNetworkInterface()
	if err != nil {
		return err
	}
	backupHTTPSettings, err = getProxySettings(proxyTypeHTTP, activeInterface)
	if err != nil {
		return err
	}

	backupHTTPSSettings, err = getProxySettings(proxyTypeHTTPS, activeInterface)
	if err != nil {
		return err
	}

	// Set the web proxy and secure web proxy
	if err := setProxySettings(proxyTypeHTTP, activeInterface, host, port); err != nil {
		return err
	}
	if err := setProxySettings(proxyTypeHTTPS, activeInterface, host, port); err != nil {
		// revert previous changes
		return err
	}

	return nil
}

func UnsetWebProxy() error {
	// Get the active network interface
	activeInterface, err := getActiveNetworkInterface()
	if err != nil {
		return err
	}
	// Restore the previous settings before disabling the proxy
	if backupHTTPSettings != nil {
		if err := setProxySettings(proxyTypeHTTP, activeInterface, backupHTTPSettings.host, backupHTTPSettings.port); err != nil {
			return err
		}
	}

	if backupHTTPSSettings != nil {
		if err := setProxySettings(proxyTypeHTTPS, activeInterface, backupHTTPSSettings.host, backupHTTPSSettings.port); err != nil {
			return err
		}
	}

	// Unset the web proxy and secure web proxy
	if err := disableProxy(proxyTypeHTTP, activeInterface); err != nil {
		return err
	}
	if err := disableProxy(proxyTypeHTTPS, activeInterface); err != nil {
		return err
	}

	return nil
}

func SetSOCKSProxy(host string, port string) error {
	// Get the active network interface
	activeInterface, err := getActiveNetworkInterface()
	if err != nil {
		return err
	}

	// Backup the previous previous proxy settings before setting the new proxy settings
	backupSOCKSSettings, err = getProxySettings(proxyTypeHTTP, activeInterface)
	if err != nil {
		return err
	}

	// Set the SOCKS proxy
	if err := setProxySettings(proxyTypeSOCKS, activeInterface, host, port); err != nil {
		return err
	}

	return nil
}

func UnsetSOCKSProxy() error {
	// Get the active network interface
	activeInterface, err := getActiveNetworkInterface()
	if err != nil {
		return err
	}

	// Revert to previous the SOCKS proxy
	if err := setProxySettings(proxyTypeSOCKS, activeInterface, backupSOCKSSettings.host, backupSOCKSSettings.port); err != nil {
		return err
	}

	// Unset the SOCKS proxy
	if err := disableProxy(proxyTypeSOCKS, activeInterface); err != nil {
		return err
	}

	return nil
}

// getActiveNetworkInterface finds the active network interface using shell commands.
// https://keith.github.io/xcode-man-pages/networksetup.8.html#listnetworkserviceorder
func getActiveNetworkInterface() (string, error) {
	//cmd := "networksetup -listnetworkserviceorder | grep `route -n get 0.0.0.0 | grep 'interface' | cut -d ':' -f2` -B 1 | head -n 1 | cut -d ' ' -f2"
	out, err := exec.Command("networksetup", "-listnetworkserviceorder").Output()
	if err != nil {
		return "", err
	}
	inet, err := getDefaultRouteInterface()
	if err != nil {
		return "", err
	}
	return getNetworkServiceName(string(out), inet)
}

// getDefaultRouteInterface gets the default route interface using os command.
// Example output of `route get default` on macOS:
//
//	route to: default
//	destination: default
//	mask: default
//	gateway: 192.168.1.1
//	interface: en0
//	flags: <UP,GATEWAY,DONE,STATIC,PRCLONING,GLOBAL>
//	recvpipe  sendpipe  ssthresh  rtt,msec    rttvar  hopcount      mtu     expire
//	0         0         0         0         0         0      1500         0
func getDefaultRouteInterface() (string, error) {
	// Execute a command to get the default route
	out, err := exec.Command("route", "get", "default").Output()
	if err != nil {
		return "", err
	}

	// Extract the interface name from the command output
	output := string(out)
	lines := strings.Split(output, "\n")
	var defaultIface string
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "interface:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				defaultIface = fields[1]
				return defaultIface, nil
			}
		}
	}
	return "", fmt.Errorf("failed to get default route interface")
}

// getNetworkServiceName parses the output of networksetup -listnetworkserviceorder to find
// the network service name for a given hardware port (e.g. Wi-Fi for en0)
func getNetworkServiceName(output, hardwarePort string) (string, error) {
	const pattern = `Hardware Port: ([^,]+),`
	// example line: (Hardware Port: Wi-Fi, Device: en0)
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, hardwarePort) {
			re := regexp.MustCompile(pattern)
			matches := re.FindStringSubmatch(line)
			if len(matches) >= 2 {
				return matches[1], nil
			}
		}
	}
	return "", fmt.Errorf("failed to find network service name for hardware port %s", hardwarePort)
}

// setProxyCommand sets the specified type of proxy on the given network interface.
// https://keith.github.io/xcode-man-pages/networksetup.8.html#getsecurewebproxy
func setProxySettings(p ProxyType, interfaceName string, host string, port string) error {
	switch p {
	case proxyTypeHTTP:
		return exec.Command("networksetup", "-setwebproxy", interfaceName, host, port).Run()
	case proxyTypeHTTPS:
		return exec.Command("networksetup", "-setsecurewebproxy", interfaceName, host, port).Run()
	case proxyTypeSOCKS:
		return exec.Command("networksetup", "-setsocksfirewallproxy", interfaceName, host, port).Run()
	default:
		return fmt.Errorf("unsupported proxy type: %s", p)
	}
}

func getProxySettings(p ProxyType, interfaceName string) (*ProxySettings, error) {
	var output []byte
	var err error
	switch p {
	case proxyTypeHTTP:
		output, err = exec.Command("networksetup", "-getwebproxy", interfaceName).Output()
	case proxyTypeHTTPS:
		output, err = exec.Command("networksetup", "-getsecurewebproxy", interfaceName).Output()
	case proxyTypeSOCKS:
		output, err = exec.Command("networksetup", "-getsocksfirewallproxy", interfaceName).Output()
	default:
		err = fmt.Errorf("unsupported proxy type: %s", p)
	}
	if err != nil {
		return nil, err
	}
	return getHostandPort(string(output))
}
func getHostandPort(commandOutput string) (*ProxySettings, error) {
	var host, port string
	lines := strings.Split(commandOutput, "\n")
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "Server:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				host = fields[1]
			}
		} else if strings.HasPrefix(strings.TrimSpace(line), "Port:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				port = fields[1]
			}
		}
	}
	if host == "" || port == "" {
		return nil, fmt.Errorf("failed to parse host and port from output")
	}
	return &ProxySettings{host: host, port: port}, nil
}

// removeProxyCommand unsets the specified type of proxy from the given network interface.
// https://keith.github.io/xcode-man-pages/networksetup.8.html#setwebproxystate
func disableProxy(p ProxyType, interfaceName string) error {
	switch p {
	case proxyTypeHTTP:
		return exec.Command("networksetup", "-setwebproxystate", interfaceName, "off").Run()
	case proxyTypeHTTPS:
		return exec.Command("networksetup", "-setsecurewebproxystate", interfaceName, "off").Run()
	case proxyTypeSOCKS:
		return exec.Command("networksetup", "-setsocksfirewallproxystate", interfaceName, "off").Run()
	default:
		return fmt.Errorf("unsupported proxy type: %s", p)
	}
}