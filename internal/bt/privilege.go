package bt

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
)

type PrivilegeMode string

const (
	PrivilegeModeAuto   PrivilegeMode = "auto"
	PrivilegeModePolkit PrivilegeMode = "polkit"
	PrivilegeModeSudo   PrivilegeMode = "sudo"
	PrivilegeModeNone   PrivilegeMode = "none"
)

const (
	defaultPolkitHelperPath = "/usr/libexec/tws_manager_rfcomm_helper"
	altPolkitHelperPath     = "/usr/lib/tws_manager/tws_manager_rfcomm_helper"
)

type privilegeConfig struct {
	mode       PrivilegeMode
	helperPath string
}

var (
	privilegeMu sync.RWMutex
	privilege   = privilegeConfig{mode: PrivilegeModeSudo}
)

func ConfigurePrivileges(mode, helperPath string) error {
	parsed, err := ParsePrivilegeMode(mode)
	if err != nil {
		return err
	}
	privilegeMu.Lock()
	defer privilegeMu.Unlock()
	privilege.mode = parsed
	privilege.helperPath = strings.TrimSpace(helperPath)
	return nil
}

func ParsePrivilegeMode(raw string) (PrivilegeMode, error) {
	switch PrivilegeMode(strings.ToLower(strings.TrimSpace(raw))) {
	case "", PrivilegeModeSudo:
		return PrivilegeModeSudo, nil
	case PrivilegeModeAuto, PrivilegeModePolkit, PrivilegeModeNone:
		return PrivilegeMode(strings.ToLower(strings.TrimSpace(raw))), nil
	default:
		return "", fmt.Errorf("unknown privilege helper mode %q", raw)
	}
}

func CurrentPrivilegeMode() PrivilegeMode {
	privilegeMu.RLock()
	defer privilegeMu.RUnlock()
	return privilege.mode
}

func WarmupPrivileges() (bool, error) {
	switch CurrentPrivilegeMode() {
	case PrivilegeModeSudo:
		return WarmupSudo()
	default:
		return false, nil
	}
}

func privilegedRFCCOMMBind(num, mac string, channel int) error {
	return withPrivilegeFallback(
		func() error { return polkitBind(num, mac, channel) },
		func() error { return sudoRFCCOMMBind(num, mac, channel) },
	)
}

func privilegedRFCOMMRelease(num string) error {
	return withPrivilegeFallback(
		func() error { return polkitRelease(num) },
		func() error { return sudoRFCCOMMRelease(num) },
	)
}

func privilegedEnsureRFCOMMAccess(device, owner string) error {
	return withPrivilegeFallback(
		func() error { return polkitFixPerms(device, owner) },
		func() error {
			if err := sudoDeviceChown(device, owner); err != nil {
				return err
			}
			return sudoDeviceChmod(device)
		},
	)
}

func withPrivilegeFallback(polkitFn, sudoFn func() error) error {
	switch CurrentPrivilegeMode() {
	case PrivilegeModeNone:
		return fmt.Errorf("privileged RFCOMM operations are disabled (--privilege-helper=none)")
	case PrivilegeModePolkit:
		return polkitFn()
	case PrivilegeModeSudo:
		return sudoFn()
	case PrivilegeModeAuto:
		if err := polkitFn(); err != nil {
			if sudoErr := sudoFn(); sudoErr != nil {
				return fmt.Errorf("polkit failed: %w; sudo fallback failed: %w", err, sudoErr)
			}
		}
		return nil
	default:
		return sudoFn()
	}
}

func polkitBind(num, mac string, channel int) error {
	args, err := polkitBindArgs(num, mac, channel)
	if err != nil {
		return err
	}
	return execPolkit(args...)
}

func polkitRelease(num string) error {
	args, err := polkitReleaseArgs(num)
	if err != nil {
		return err
	}
	return execPolkit(args...)
}

func polkitFixPerms(device, owner string) error {
	args, err := polkitFixPermsArgs(device, owner)
	if err != nil {
		return err
	}
	return execPolkit(args...)
}

func polkitBindArgs(num, mac string, channel int) ([]string, error) {
	if num == "" || mac == "" {
		return nil, fmt.Errorf("polkit bind requires rfcomm number and mac")
	}
	owner, err := polkitCallerOwner()
	if err != nil {
		return nil, err
	}
	return []string{
		"bind",
		"--number", num,
		"--addr", mac,
		"--channel", strconv.Itoa(channel),
		"--owner", owner,
	}, nil
}

func polkitCallerOwner() (string, error) {
	uid := os.Getuid()
	gid := os.Getgid()
	if uid < 0 || gid < 0 {
		return "", fmt.Errorf("polkit bind requires a valid caller uid:gid")
	}
	return fmt.Sprintf("%d:%d", uid, gid), nil
}

func polkitReleaseArgs(num string) ([]string, error) {
	if num == "" {
		return nil, fmt.Errorf("polkit release requires rfcomm number")
	}
	return []string{"release", "--number", num}, nil
}

func polkitFixPermsArgs(device, owner string) ([]string, error) {
	if device == "" || owner == "" {
		return nil, fmt.Errorf("polkit fix-perms requires device and owner")
	}
	return []string{"fix-perms", "--device", device, "--owner", owner}, nil
}

func execPolkit(args ...string) error {
	helperPath := discoverPolkitHelperPath()
	if helperPath == "" {
		return fmt.Errorf("polkit helper not found (checked %s and %s)", defaultPolkitHelperPath, altPolkitHelperPath)
	}
	cmdArgs := append([]string{helperPath}, args...)
	_, err := runCommand("pkexec", cmdArgs...)
	return err
}

func discoverPolkitHelperPath() string {
	privilegeMu.RLock()
	configured := privilege.helperPath
	privilegeMu.RUnlock()

	paths := []string{}
	if configured != "" {
		paths = append(paths, configured)
	}
	paths = append(paths, defaultPolkitHelperPath, altPolkitHelperPath)
	for _, p := range paths {
		info, err := os.Stat(p)
		if err == nil && !info.IsDir() {
			return p
		}
	}
	return ""
}
