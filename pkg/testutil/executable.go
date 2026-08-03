package testutil

import (
	"path/filepath"
	"runtime"
)

// ExecutablePath returns a platform-correct path for a test binary.
func ExecutablePath(directory, name string) string {
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(directory, name)
}
