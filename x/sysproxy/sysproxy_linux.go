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

//go:build linux && !android

package sysproxy

import (
	"os/exec"
)

type ProxyType string

const (
	proxyTypeHTTP  ProxyType = "http"
	proxyTypeHTTPS ProxyType = "https"
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
	var err error
	// Backup existing settings
	backupHTTPSettings, err = backupProxySettings(proxyTypeHTTP)
	if err != nil {
		return err
	}
	backupHTTPSSettings, err = backupProxySettings(proxyTypeHTTPS)
	if err != nil {
		return err
	}

	if err := setProxySettings(proxyTypeHTTP, host, port); err != nil {
		return err
	}
	if err := setProxySettings(proxyTypeHTTPS, host, port); err != nil {
		return err
	}
	if err := setManualMode(); err != nil {
		return err
	}
	return nil
}

func UnsetWebProxy() error {
	// restore previous settings
	if backupHTTPSettings != nil {
		if err := setProxySettings(proxyTypeHTTP, backupHTTPSettings.host, backupHTTPSettings.port); err != nil {
			return err
		}
	}
	if backupHTTPSSettings != nil {
		if err := setProxySettings(proxyTypeHTTPS, backupHTTPSSettings.host, backupHTTPSSettings.port); err != nil {
			return err
		}
	}
	// Execute Linux specific commands to unset proxy
	return gnomeSettingsSetString("org.gnome.system.proxy", "mode", "none")
}

func SetSocksProxy(host string, port string) error {
	var err error
	// Backup existing settings
	backupSOCKSSettings, err = backupProxySettings(proxyTypeSOCKS)
	if err != nil {
		return err
	}
	if err := setProxySettings(proxyTypeSOCKS, host, port); err != nil {
		return err
	}
	if err := setManualMode(); err != nil {
		return err
	}
	return nil
}

func setManualMode() error {
	return gnomeSettingsSetString("org.gnome.system.proxy", "mode", "manual")
}

func setProxySettings(p ProxyType, host string, port string) error {
	switch p {
	case proxyTypeHTTP:
		if err := gnomeSettingsSetString("org.gnome.system.proxy.http", "host", host); err != nil {
			return err
		}
		if err := gnomeSettingsSetString("org.gnome.system.proxy.http", "port", port); err != nil {
			return err
		}
	case proxyTypeHTTPS:
		if err := gnomeSettingsSetString("org.gnome.system.proxy.https", "host", host); err != nil {
			return err
		}
		if err := gnomeSettingsSetString("org.gnome.system.proxy.https", "port", port); err != nil {
			return err
		}
	case proxyTypeSOCKS:
		if err := gnomeSettingsSetString("org.gnome.system.proxy.socks", "host", host); err != nil {
			return err
		}
		if err := gnomeSettingsSetString("org.gnome.system.proxy.socks", "port", port); err != nil {
			return err
		}
	}
	return nil
}

func backupProxySettings(p ProxyType) (*ProxySettings, error) {
	settings := &ProxySettings{}
	var err error
	switch p {
	case proxyTypeHTTP:
		settings.host, err = gnomeSettingsGetString("org.gnome.system.proxy.http", "host")
		if err != nil {
			return nil, err
		}
		settings.port, err = gnomeSettingsGetString("org.gnome.system.proxy.http", "port")
		if err != nil {
			return nil, err
		}
	case proxyTypeHTTPS:
		settings.host, err = gnomeSettingsGetString("org.gnome.system.proxy.https", "host")
		if err != nil {
			return nil, err
		}
		settings.port, err = gnomeSettingsGetString("org.gnome.system.proxy.https", "port")
		if err != nil {
			return nil, err
		}
	case proxyTypeSOCKS:
		settings.host, err = gnomeSettingsGetString("org.gnome.system.proxy.socks", "host")
		if err != nil {
			return nil, err
		}
		settings.port, err = gnomeSettingsGetString("org.gnome.system.proxy.socks", "port")
		if err != nil {
			return nil, err
		}
	}
	return settings, nil
}

func gnomeSettingsSetString(settings, key, value string) error {
	return exec.Command("gsettings", "set", settings, key, value).Run()
}

func gnomeSettingsGetString(settings, key string) (string, error) {
	out, err := exec.Command("gsettings", "get", settings, key).Output()
	return string(out), err
}
