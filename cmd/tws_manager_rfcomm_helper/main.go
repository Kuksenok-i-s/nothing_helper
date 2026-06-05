package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"tws_manager/internal/security"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return usageError("missing action (bind|release|fix-perms)")
	}
	switch args[0] {
	case "bind":
		return runBind(args[1:])
	case "release":
		return runRelease(args[1:])
	case "fix-perms":
		return runFixPerms(args[1:])
	default:
		return usageError("unknown action %q", args[0])
	}
}

func runBind(args []string) error {
	fs := flag.NewFlagSet("bind", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		number  = fs.String("number", "", "RFCOMM numeric index")
		device  = fs.String("device", "", "RFCOMM device path")
		addr    = fs.String("addr", "", "Bluetooth MAC address")
		channel = fs.Int("channel", 15, "RFCOMM channel")
		owner   = fs.String("owner", "", "owner in uid:gid format")
	)
	if err := fs.Parse(args); err != nil {
		return err
	}
	devPath, num, err := normalizeDeviceAndNumber(*device, *number)
	if err != nil {
		return err
	}
	mac, err := security.NormalizeMAC(*addr)
	if err != nil {
		return err
	}
	if err := security.ValidateChannel(*channel); err != nil {
		return err
	}
	if _, _, err := parseOwner(*owner); err != nil {
		return err
	}
	out, err := exec.Command("rfcomm", "bind", num, mac, strconv.Itoa(*channel)).CombinedOutput()
	if err != nil {
		return fmt.Errorf("rfcomm bind %s %s %d: %w: %s", num, mac, *channel, err, strings.TrimSpace(string(out)))
	}
	if err := waitForDevice(devPath, 3*time.Second); err != nil {
		return fmt.Errorf("rfcomm bind succeeded but %s not ready: %w", devPath, err)
	}
	return ensureDevicePerms(devPath, *owner)
}

func runRelease(args []string) error {
	fs := flag.NewFlagSet("release", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		number = fs.String("number", "", "RFCOMM numeric index")
		device = fs.String("device", "", "RFCOMM device path")
	)
	if err := fs.Parse(args); err != nil {
		return err
	}
	_, num, err := normalizeDeviceAndNumber(*device, *number)
	if err != nil {
		return err
	}
	out, err := exec.Command("rfcomm", "release", num).CombinedOutput()
	if err != nil {
		msg := strings.ToLower(string(out))
		if strings.Contains(msg, "not bound") || strings.Contains(msg, "no such device") || strings.Contains(msg, "can't release") {
			return nil
		}
		return fmt.Errorf("rfcomm release %s: %w: %s", num, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func runFixPerms(args []string) error {
	fs := flag.NewFlagSet("fix-perms", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	device := fs.String("device", "", "RFCOMM device path")
	owner := fs.String("owner", "", "owner in uid:gid format")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *device == "" {
		return usageError("fix-perms requires --device")
	}
	devPath, err := security.ValidateRFCOMMDevice(*device)
	if err != nil {
		return err
	}
	return ensureDevicePerms(devPath, *owner)
}

func ensureDevicePerms(device, owner string) error {
	uid, gid, err := parseOwner(owner)
	if err != nil {
		return err
	}
	if err := os.Chown(device, uid, gid); err != nil {
		return fmt.Errorf("chown %s to %s: %w", device, owner, err)
	}
	info, err := os.Stat(device)
	if err != nil {
		return err
	}
	mode := info.Mode().Perm() | 0o600
	if err := os.Chmod(device, mode); err != nil {
		return fmt.Errorf("chmod %s: %w", device, err)
	}
	return nil
}

func parseOwner(owner string) (int, int, error) {
	parts := strings.Split(owner, ":")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return 0, 0, usageError("owner must be uid:gid")
	}
	uid, err := strconv.Atoi(parts[0])
	if err != nil || uid < 0 {
		return 0, 0, usageError("invalid owner uid %q", parts[0])
	}
	gid, err := strconv.Atoi(parts[1])
	if err != nil || gid < 0 {
		return 0, 0, usageError("invalid owner gid %q", parts[1])
	}
	return uid, gid, nil
}

func normalizeDeviceAndNumber(device, number string) (string, string, error) {
	if strings.TrimSpace(device) != "" {
		devPath, err := security.ValidateRFCOMMDevice(device)
		if err != nil {
			return "", "", err
		}
		num, err := security.RFCOMMNumber(devPath)
		if err != nil {
			return "", "", err
		}
		if number != "" && number != num {
			return "", "", usageError("--number %s does not match --device %s", number, devPath)
		}
		return devPath, num, nil
	}
	if strings.TrimSpace(number) == "" {
		return "", "", usageError("either --device or --number is required")
	}
	if err := security.ValidateRFCOMMNumber(number); err != nil {
		return "", "", err
	}
	return "/dev/rfcomm" + number, number, nil
}

func waitForDevice(device string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var last error
	for time.Now().Before(deadline) {
		if _, err := os.Stat(device); err == nil {
			return nil
		} else {
			last = err
		}
		time.Sleep(100 * time.Millisecond)
	}
	if last == nil {
		last = errors.New("device did not appear")
	}
	return last
}

func usageError(format string, args ...any) error {
	return fmt.Errorf("usage: "+format, args...)
}
