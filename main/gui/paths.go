package main

import (
	"fmt"
	"os"
	"path/filepath"
)

const applicationDirectory = "Mixology"

func defaultDataDirectory() (string, error) {
	root, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locate user configuration directory: %w", err)
	}
	return filepath.Join(root, applicationDirectory), nil
}

func prepareDataDirectory(path string) error {
	if path == "" {
		return fmt.Errorf("desktop data directory is required")
	}
	if err := os.MkdirAll(path, 0o700); err != nil {
		return fmt.Errorf("create desktop data directory: %w", err)
	}
	return nil
}
