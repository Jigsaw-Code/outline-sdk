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
	"errors"
	"fmt"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

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

// https://learn.microsoft.com/en-us/windows/win32/api/wininet/nf-wininet-internetsetoptionw
// internetSetOption sets an Internet option.
func internetSetOption(hInternet uintptr, dwOption int, lpBuffer uintptr, dwBufferLength uint32) bool {
	ret, _, _ := procInternetSetOption.Call(
		hInternet,
		uintptr(dwOption),
		lpBuffer,
		uintptr(dwBufferLength),
	)
	return ret != 0
}

func resetWininetProxySettings() error {
	result1 := internetSetOption(0, INTERNET_OPTION_SETTINGS_CHANGED, 0, 0)
	result2 := internetSetOption(0, INTERNET_OPTION_REFRESH, 0, 0)

	if result1 && result2 {
		fmt.Println("Operation successful")
		return nil
	} else {
		return errors.New("Wininet setting change operation failed")
	}
}

func SetProxy(ip string, port string) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Internet Settings`, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()

	values := map[string]interface{}{
		"MigrateProxy":  1,
		"ProxyEnable":   1,
		"ProxyHttp1.1":  0,
		"ProxyServer":   fmt.Sprintf("%s:%s", ip, port),
		"ProxyOverride": "*.local;<local>",
	}

	for name, value := range values {
		switch v := value.(type) {
		case int:
			err = key.SetDWordValue(name, uint32(v))
		case string:
			err = key.SetStringValue(name, v)
		default:
			return fmt.Errorf("unsupported value type")
		}
		if err != nil {
			return err
		}
	}

	// Refresh the settings
	err = resetWininetProxySettings()
	if err != nil {
		return err
	}

	return nil
}

func UnsetProxy() error {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Internet Settings`, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()

	// Set ProxyEnable to 0 and ProxyServer to an empty string
	err = key.SetDWordValue("ProxyEnable", 0)
	if err != nil {
		return err
	}
	err = key.SetStringValue("ProxyServer", "")
	if err != nil {
		return err
	}

	// Refresh the settings
	err = resetWininetProxySettings()
	if err != nil {
		return err
	}

	return nil
}
