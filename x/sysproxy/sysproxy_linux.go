//go:build linux

package sysproxy

import (
	"fmt"
	"os/exec"
)

func SetProxy(ip string, port string) error {
	// Execute Linux specific commands to set proxy
	if err := setManualMode(); err != nil {
		return err
	}
	if err := setHttpProxy(ip, port); err != nil {
		return err
	}
	if err := setHttpsProxy(ip, port); err != nil {
		return err
	}
	if err := setFtpProxy(ip, port); err != nil {
		return err
	}
	return nil
}

func setManualMode() error {
	return execCommand("gsettings", "set", "org.gnome.system.proxy", "mode", "manual")
}

func setHttpProxy(ip string, port string) error {
	if err := execCommand("gsettings", "set", "org.gnome.system.proxy.http", "host", ip); err != nil {
		return err
	}
	if err := execCommand("gsettings", "set", "org.gnome.system.proxy.http", "port", port); err != nil {
		return err
	}
	return nil
}

func setHttpsProxy(ip string, port string) error {
	if err := execCommand("gsettings", "set", "org.gnome.system.proxy.https", "host", ip); err != nil {
		return err
	}
	if err := execCommand("gsettings", "set", "org.gnome.system.proxy.https", "port", port); err != nil {
		return err
	}
	return nil
}

func setFtpProxy(ip string, port string) error {
	if err := execCommand("gsettings", "set", "org.gnome.system.proxy.ftp", "host", ip); err != nil {
		return err
	}
	if err := execCommand("gsettings", "set", "org.gnome.system.proxy.ftp", "port", port); err != nil {
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
