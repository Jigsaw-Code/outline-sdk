//go:build !linux && !windows && !darwin

package sysproxy

// SetProxy does nothing on unsupported platforms.
func SetProxy(ip string, port string) error {
	return nil
}

// SetProxy does nothing on unsupported platforms.
func UnsetProxy() error {
	return nil
}
