//go:build psiphon
// +build psiphon

package smart

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func GetAndroidPackageName() (string, error) {
	f, err := os.Open("/proc/self/cmdline")
	if err != nil {
		return "", err
	}
	defer f.Close()

	// /proc/self/cmdline is NULL-terminated; read up to the first NULL.
	r := bufio.NewReader(f)
	line, err := r.ReadString('\x00')
	if err != nil && !errors.Is(err, bufio.ErrBufferFull) {
		// On success the NULL will be included; trim regardless.
	}
	pkg := strings.Trim(line, "\x00\r\n\t ")
	if pkg == "" {
		return "", errors.New("could not determine package name from /proc/self/cmdline")
	}
	return pkg, nil
}

// AndroidPrivateCacheDirNoContext returns the app-private cache dir path
// (e.g., /data/data/<pkg>/cache), without using any Android Context.
// It validates the directory exists (or creates it).
func AndroidPrivateCacheDirNoContext() (string, error) {
	pkg, err := GetAndroidPackageName()
	if err != nil {
		return "", err
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
