//go:build darwin

package bt

import (
	"fmt"
	"strings"
)

type PrivilegeMode string

const (
	PrivilegeModeAuto   PrivilegeMode = "auto"
	PrivilegeModePolkit PrivilegeMode = "polkit"
	PrivilegeModeSudo   PrivilegeMode = "sudo"
	PrivilegeModeNone   PrivilegeMode = "none"
)

func ConfigurePrivileges(mode, helperPath string) error {
	parsed, err := ParsePrivilegeMode(mode)
	if err != nil {
		return err
	}
	if parsed != PrivilegeModeNone && strings.TrimSpace(mode) != "" && strings.ToLower(strings.TrimSpace(mode)) != "none" {
		return fmt.Errorf("darwin only supports --privilege-helper=none, got %q", mode)
	}
	return nil
}

func ParsePrivilegeMode(raw string) (PrivilegeMode, error) {
	switch PrivilegeMode(strings.ToLower(strings.TrimSpace(raw))) {
	case "", PrivilegeModeNone:
		return PrivilegeModeNone, nil
	case PrivilegeModeAuto, PrivilegeModePolkit, PrivilegeModeSudo:
		return PrivilegeModeNone, nil
	default:
		return "", fmt.Errorf("unknown privilege helper mode %q", raw)
	}
}

func CurrentPrivilegeMode() PrivilegeMode { return PrivilegeModeNone }

func WarmupPrivileges() (bool, error) { return false, nil }
