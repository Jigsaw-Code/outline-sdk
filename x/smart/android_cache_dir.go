//go:build android
// +build android

package smart

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// PrivateCacheDirNoContext returns the app-private cache dir path
// (e.g., /data/data/<pkg>/cache), without using any Android Context.
// It validates the directory exists (or creates it).
func PrivateCacheDirNoContext() (string, error) {
	f, err := os.Open("/proc/self/cmdline")
	if err != nil {
		return "", err
	}
	defer f.Close()

	// /proc/self/cmdline is NUL-terminated; read up to the first NUL.
	r := bufio.NewReader(f)
	line, err := r.ReadString('\x00')
	if err != nil && !errors.Is(err, bufio.ErrBufferFull) {
		// On success the NUL will be included; trim regardless.
	}
	pkg := strings.Trim(line, "\x00\r\n\t ")
	if pkg == "" {
		return "", errors.New("could not determine package name from /proc/self/cmdline")
	}

	// Prefer legacy symlink location; it resolves to /data/user/<id>/<pkg>.
	p := filepath.Join("/data/data", pkg, "cache")

	// Ensure it exists (normally already does).
	if st, err := os.Stat(p); err == nil && st.IsDir() {
		return p, nil
	}
	if err := os.MkdirAll(p, 0o700); err != nil {
		return "", err
	}
	return p, nil
}
