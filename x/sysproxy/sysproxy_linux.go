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

//go:build linux

package sysproxy

import (
	"fmt"
	"os/exec"
)

type ProxyType string

const (
	httpProxyType  ProxyType = "http"
	httpsProxyType ProxyType = "https"
	ftpProxyType   ProxyType = "ftp"
)

func SetProxy(ip string, port string) error {
	// Execute Linux specific commands to set proxy
	if err := setManualMode(); err != nil {
		return err
	}
	if err := setWebProxy(httpProxyType, ip, port); err != nil {
		return err
	}
	if err := setWebProxy(httpsProxyType, ip, port); err != nil {
		return err
	}
	if err := setWebProxy(ftpProxyType, ip, port); err != nil {
		return err
	}
	return nil
}

func setManualMode() error {
	return execCommand("gsettings", "set", "org.gnome.system.proxy", "mode", "manual")
}

func setWebProxy(pType ProxyType, ip string, port string) error {
	p := fmt.Sprintf("org.gnome.system.proxy.%s", pType)
	if err := execCommand("gsettings", "set", p, "host", ip); err != nil {
		return err
	}
	if err := execCommand("gsettings", "set", p, "port", port); err != nil {
		return err
	}
	return nil
}

func UnsetProxy() error {
	// Execute Linux specific commands to unset proxy
	return execCommand("gsettings", "set", "org.gnome.system.proxy", "mode", "none")
}

func execCommand(name string, arg ...string) error {
	cmd := exec.Command(name, arg...)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to execute command: %w", err)
	}
	return nil
}
