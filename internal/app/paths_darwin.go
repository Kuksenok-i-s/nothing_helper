//go:build darwin

package app

import (
	"os"
	"path/filepath"
)

// darwinDataDir returns ~/Library/Application Support/nothing_helper (or fallback).
func darwinDataDir() string {
	base, err := os.UserConfigDir()
	if err != nil {
		return "nothing_helper"
	}
	return filepath.Join(base, "nothing_helper")
}

func captureDirDefault(profile Profile) string {
	if profile == ProfileGUI {
		return filepath.Join(darwinDataDir(), "captures")
	}
	return "captures"
}
