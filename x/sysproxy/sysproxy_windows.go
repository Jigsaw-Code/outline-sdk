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

//go:build windows

package sysproxy

import (
	"fmt"
	"net"
	"strings"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

type ProxySettings struct {
	ProxyServer   string
	ProxyOverride string
}

var (
	modwininet            = windows.NewLazySystemDLL("wininet.dll")
	procInternetSetOption = modwininet.NewProc("InternetSetOptionW")
)

// https://learn.microsoft.com/en-us/windows/win32/wininet/option-flags
// INTERNET_OPTION_SETTINGS_CHANGED: 39
// Notifies the system that the registry settings have been changed so that it verifies the settings on the next call to InternetConnect.
// INTERNET_OPTION_REFRESH: 37
// Causes the proxy data to be reread from the registry for a handle. No buffer is required.
// This option can be used on the HINTERNET handle returned by InternetOpen.
// This is used by InternetSetOption.
const (
	INTERNET_OPTION_SETTINGS_CHANGED = 39
	INTERNET_OPTION_REFRESH          = 37
)

func SetWebProxy(host string, port string) error {

	settings := &ProxySettings{
		ProxyServer:   net.JoinHostPort(host, port),
		ProxyOverride: "*.local;<local>",
	}

	if err := setProxySettings(settings); err != nil {
		return err
	}

	return nil
}

func ClearWebProxy() error {
	// clear proxy settings
	return clearProxy()
}

// SetProxy does nothing on windows platforms.
func SetSOCKSProxy(host string, port string) error {
	endpoint := fmt.Sprintf("socks=%s", net.JoinHostPort(host, port))
	settings := &ProxySettings{
		ProxyServer:   endpoint,
		ProxyOverride: "*.local;<local>",
	}

	if err := setProxySettings(settings); err != nil {
		return err
	}

	return nil
}

// SetProxy does nothing on windows platforms.
func ClearSOCKSProxy() error {
	return clearProxy()
}

func clearProxy() error {
	settings := &ProxySettings{
		ProxyServer:   "127.0.0.1:0",
		ProxyOverride: "*.local;<local>",
	}

	if err := setProxySettings(settings); err != nil {
		return err
	}

	err := disableProxy()
	if err != nil {
		return err
	}
	return nil
}

func setProxySettings(settings *ProxySettings) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Internet Settings`, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()

	if err = key.SetStringValue("ProxyServer", settings.ProxyServer); err != nil {
		return err
	}
	if err = key.SetStringValue("ProxyOverride", settings.ProxyOverride); err != nil {
		return err
	}
	// Finally, enable the proxy
	if err = key.SetDWordValue("ProxyEnable", uint32(1)); err != nil {
		return err
	}

	// Refresh the settings
	return notifyWinInetProxySettingsChanged()
}

func disableProxy() error {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Internet Settings`, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()

	// Set ProxyEnable to 0
	err = key.SetDWordValue("ProxyEnable", 0)
	if err != nil {
		return err
	}

	// Refresh the settings
	return notifyWinInetProxySettingsChanged()
}

// https://learn.microsoft.com/en-us/windows/win32/api/wininet/nf-wininet-internetsetoptionw
// internetSetOption sets an Internet option.
func internetSetOption(hInternet uintptr, dwOption int, lpBuffer uintptr, dwBufferLength uint32) error {
	ret, _, lastErr := procInternetSetOption.Call(
		hInternet,
		uintptr(dwOption),
		lpBuffer,
		uintptr(dwBufferLength),
	)
	if ret == 0 {
		return lastErr
	}
	return nil
}

func notifyWinInetProxySettingsChanged() error {
	if err := internetSetOption(0, INTERNET_OPTION_SETTINGS_CHANGED, 0, 0); err != nil {
		return fmt.Errorf("failed to notify the system that the registry settings have been changed: %w", err)
	}

	if err := internetSetOption(0, INTERNET_OPTION_REFRESH, 0, 0); err != nil {
		return fmt.Errorf("failed to refresh the proxy data from the registry: %w", err)
	}

	return nil
}
func getWebProxy() (host string, port string, err error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Internet Settings`, registry.QUERY_VALUE)
	if err != nil {
		return "", "", err
	}
	defer key.Close()

	address, _, err := key.GetStringValue("ProxyServer")
	fmt.Printf("socks address: %s\n", address)
	if err != nil {
		return "", "", err
	}

	host, port, err = net.SplitHostPort(address)
	if err != nil {
		return "", "", err
	}

	return host, port, nil
}

func getSOCKSProxy() (host string, port string, err error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Internet Settings`, registry.QUERY_VALUE)
	if err != nil {
		return "", "", err
	}
	defer key.Close()

	address, _, err := key.GetStringValue("ProxyServer")
	h := strings.TrimPrefix(address, "socks=")

	host, port, err = net.SplitHostPort(h)
	if err != nil {
		return "", "", err
	}

	return host, port, nil
}
