package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"nothing_helper/internal/security"
)

var execCombinedOutput = func(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

var (
	waitForDeviceHook     func(string, time.Duration) error
	ensureDevicePermsHook func(string, string) error
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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	switch args[0] {
	case "bind":
		return runBind(ctx, args[1:])
	case "release":
		return runRelease(ctx, args[1:])
	case "fix-perms":
		return runFixPerms(args[1:])
	default:
		return usageError("unknown action %q", args[0])
	}
}

func runBind(ctx context.Context, args []string) error {
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
	devPath, num, mac, err := validateBindInputs(*device, *number, *addr, *channel, *owner)
	if err != nil {
		return err
	}
	out, err := execCombinedOutput(ctx, "rfcomm", "bind", num, mac, strconv.Itoa(*channel))
	if err != nil {
		return fmt.Errorf("rfcomm bind %s %s %d: %w: %s", num, mac, *channel, err, strings.TrimSpace(string(out)))
	}
	if err := waitForDeviceFn(devPath, 3*time.Second); err != nil {
		return fmt.Errorf("rfcomm bind succeeded but %s not ready: %w", devPath, err)
	}
	return ensureDevicePermsFn(devPath, *owner)
}

func validateBindInputs(device, number, addr string, channel int, owner string) (devPath, num, mac string, err error) {
	devPath, num, err = normalizeDeviceAndNumber(device, number)
	if err != nil {
		return "", "", "", err
	}
	mac, err = security.NormalizeMAC(addr)
	if err != nil {
		return "", "", "", err
	}
	if err := security.ValidateChannel(channel); err != nil {
		return "", "", "", err
	}
	if _, _, err := parseOwner(owner); err != nil {
		return "", "", "", err
	}
	return devPath, num, mac, nil
}

func runRelease(ctx context.Context, args []string) error {
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
	out, err := execCombinedOutput(ctx, "rfcomm", "release", num)
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
	if ensureDevicePermsHook != nil {
		return ensureDevicePermsHook(device, owner)
	}
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
	uid, err := parseOwnerID(parts[0], "uid")
	if err != nil {
		return 0, 0, err
	}
	gid, err := parseOwnerID(parts[1], "gid")
	if err != nil {
		return 0, 0, err
	}
	return uid, gid, nil
}

func parseOwnerID(raw, label string) (int, error) {
	id, err := strconv.Atoi(raw)
	if err != nil || id < 0 {
		return 0, usageError("invalid owner %s %q", label, raw)
	}
	return id, nil
}

func normalizeDeviceAndNumber(device, number string) (string, string, error) {
	if strings.TrimSpace(device) != "" {
		return normalizeDevicePath(device, number)
	}
	return normalizeRFCOMMNumber(number)
}

func normalizeDevicePath(device, number string) (string, string, error) {
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

func normalizeRFCOMMNumber(number string) (string, string, error) {
	if strings.TrimSpace(number) == "" {
		return "", "", usageError("either --device or --number is required")
	}
	if err := security.ValidateRFCOMMNumber(number); err != nil {
		return "", "", err
	}
	return "/dev/rfcomm" + number, number, nil
}

func waitForDevice(device string, timeout time.Duration) error {
	if waitForDeviceHook != nil {
		return waitForDeviceHook(device, timeout)
	}
	deadline := time.Now().Add(timeout)
	var last error
	for time.Now().Before(deadline) {
		_, err := os.Stat(device)
		if err == nil {
			return nil
		}
		last = err
		time.Sleep(100 * time.Millisecond)
	}
	if last == nil {
		last = errors.New("device did not appear")
	}
	return last
}

func waitForDeviceFn(device string, timeout time.Duration) error {
	if waitForDeviceHook != nil {
		return waitForDeviceHook(device, timeout)
	}
	return waitForDevice(device, timeout)
}

func ensureDevicePermsFn(device, owner string) error {
	if ensureDevicePermsHook != nil {
		return ensureDevicePermsHook(device, owner)
	}
	return ensureDevicePerms(device, owner)
}

func usageError(format string, args ...any) error {
	return fmt.Errorf("usage: "+format, args...)
}
