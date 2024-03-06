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

//go:build darwin

package sysproxy

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

type ProxyType string

const (
	httpProxyType  ProxyType = "web"
	httpsProxyType ProxyType = "secureweb"
)

func SetProxy(host string, port string) error {
	// Get the active network interface
	activeInterface, err := getActiveNetworkInterface()
	if err != nil {
		return err
	}

	// Set the web proxy and secure web proxy
	if err := setProxyCommand(httpProxyType, activeInterface, ip, port); err != nil {
		return err
	}
	if err := setProxyCommand(httpsProxyType, activeInterface, ip, port); err != nil {
		return err
	}

	return nil
}

func UnsetProxy() error {
	// Get the active network interface
	activeInterface, err := getActiveNetworkInterface()
	if err != nil {
		return err
	}

	// Unset the web proxy and secure web proxy
	if err := removeProxyCommand(httpProxyType, activeInterface); err != nil {
		return err
	}
	if err := removeProxyCommand(httpsProxyType, activeInterface); err != nil {
		return err
	}

	return nil
}

// getActiveNetworkInterface finds the active network interface using shell commands.
// https://keith.github.io/xcode-man-pages/networksetup.8.html#listnetworkserviceorder
func getActiveNetworkInterface() (string, error) {
	cmd := "networksetup -listnetworkserviceorder | grep `route -n get 0.0.0.0 | grep 'interface' | cut -d ':' -f2` -B 1 | head -n 1 | cut -d ' ' -f2"
	out, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// setProxyCommand sets the specified type of proxy on the given network interface.
// https://keith.github.io/xcode-man-pages/networksetup.8.html#getsecurewebproxy
func setProxyCommand(ptype ProxyType, interfaceName string, ip string, port string) error {
	cmdStr := fmt.Sprintf("networksetup -set%sproxy \"%s\" %s %s", ptype, interfaceName, ip, port)

	return runCommand(cmdStr)
}

// removeProxyCommand unsets the specified type of proxy from the given network interface.
// https://keith.github.io/xcode-man-pages/networksetup.8.html#setwebproxystate
func removeProxyCommand(ptype ProxyType, interfaceName string) error {
	cmdStr := fmt.Sprintf("networksetup -set%sproxystate \"%s\" off", ptype, interfaceName)

	return runCommand(cmdStr)
}

func runCommand(cmdStr string) error {
	cmd := exec.Command("bash", "-c", cmdStr)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("%v: %s", err, stderr.String())
	}
	return nil
}
