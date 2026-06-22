//go:build darwin

package app

import (
	"os"
	"path/filepath"
)

// darwinDataDir returns ~/Library/Application Support/tws_manager (or fallback).
func darwinDataDir() string {
	base, err := os.UserConfigDir()
	if err != nil {
		return "tws_manager"
	}
	return filepath.Join(base, "tws_manager")
}

func captureDirDefault(profile Profile) string {
	if profile == ProfileGUI {
		return filepath.Join(darwinDataDir(), "captures")
	}
	return "captures"
}
