//go:build darwin

package sysproxy

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

func SetProxy(ip string, port string) error {
	// Execute macOS specific commands to set proxy
	// Get the active network interface
	activeInterface, err := getActiveNetworkInterfaceMacOS()
	if err != nil {
		return err
	}

	// Set the web proxy and secure web proxy
	if err := setProxyMacOS("web", activeInterface, ip, port); err != nil {
		return err
	}
	if err := setProxyMacOS("secureweb", activeInterface, ip, port); err != nil {
		return err
	}

	return nil
}

func UnsetProxy() error {
	// Execute macOS specific commands to unset proxy
	// Get the active network interface
	activeInterface, err := getActiveNetworkInterfaceMacOS()
	if err != nil {
		return err
	}

	// Set the web proxy and secure web proxy
	if err := removeProxyMacOS("web", activeInterface); err != nil {
		return err
	}
	if err := removeProxyMacOS("secureweb", activeInterface); err != nil {
		return err
	}

	return nil
}

// getActiveNetworkInterface finds the active network interface using shell commands.
func getActiveNetworkInterfaceMacOS() (string, error) {
	cmd := "sh -c \"networksetup -listnetworkserviceorder | grep `route -n get 0.0.0.0 | grep 'interface' | cut -d ':' -f2` -B 1 | head -n 1 | cut -d ' ' -f2\""
	out, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// setProxyMacOS sets the specified type of proxy on the given network interface.
func setProxyMacOS(proxyType string, interfaceName string, ip string, port string) error {
	var cmdStr string
	if proxyType == "web" {
		cmdStr = fmt.Sprintf("networksetup -setwebproxy \"%s\" %s %s", interfaceName, ip, port)
	} else if proxyType == "secureweb" {
		cmdStr = fmt.Sprintf("networksetup -setsecurewebproxy \"%s\" %s %s", interfaceName, ip, port)
	} else {
		return fmt.Errorf("unknown proxy type: %s", proxyType)
	}

	cmd := exec.Command("bash", "-c", cmdStr)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("%v: %s", err, stderr.String())
	}
	return nil
}

func removeProxyMacOS(proxyType string, interfaceName string) error {
	var cmdStr string
	if proxyType == "web" {
		cmdStr = fmt.Sprintf("networksetup -setwebproxystate \"%s\" off", interfaceName)
	} else if proxyType == "secureweb" {
		cmdStr = fmt.Sprintf("networksetup -setsecurewebproxystate \"%s\" off", interfaceName)
	} else {
		return fmt.Errorf("unknown proxy type: %s", proxyType)
	}

	cmd := exec.Command("bash", "-c", cmdStr)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("%v: %s", err, stderr.String())
	}
	return nil
}
