package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNormalizeDeviceAndNumberFromDevice(t *testing.T) {
	dev, num, err := normalizeDeviceAndNumber("/dev/rfcomm3", "")
	if err != nil || dev != "/dev/rfcomm3" || num != "3" {
		t.Fatalf("dev=%q num=%q err=%v", dev, num, err)
	}
}

func TestNormalizeDeviceAndNumberFromNumber(t *testing.T) {
	dev, num, err := normalizeDeviceAndNumber("", "5")
	if err != nil || dev != "/dev/rfcomm5" || num != "5" {
		t.Fatalf("dev=%q num=%q err=%v", dev, num, err)
	}
}

func TestParseOwner(t *testing.T) {
	uid, gid, err := parseOwner("1000:1000")
	if err != nil || uid != 1000 || gid != 1000 {
		t.Fatalf("uid=%d gid=%d err=%v", uid, gid, err)
	}
	_, _, err = parseOwner("bad")
	if err == nil {
		t.Fatal("expected owner parse error")
	}
}

func TestEnsureDevicePerms(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rfcomm-test")
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ensureDevicePerms(path, fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid())); err != nil {
		t.Fatalf("ensureDevicePerms() = %v", err)
	}
}

func TestWaitForDevice(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ready")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := waitForDevice(path, 500*time.Millisecond); err != nil {
		t.Fatalf("waitForDevice() = %v", err)
	}
}

func TestRunReleaseNotBound(t *testing.T) {
	old := execCombinedOutput
	execCombinedOutput = func(context.Context, string, ...string) ([]byte, error) {
		return []byte("Can't release device: Not bound"), errors.New("failed")
	}
	t.Cleanup(func() { execCombinedOutput = old })

	if err := run([]string{"release", "--number", "0"}); err != nil {
		t.Fatalf("run(release) = %v", err)
	}
}

func TestRunMissingAction(t *testing.T) {
	if err := run(nil); err == nil {
		t.Fatal("expected usage error")
	}
}

func TestRunBindSuccess(t *testing.T) {
	oldExec := execCombinedOutput
	oldWait := waitForDeviceHook
	oldEnsure := ensureDevicePermsHook
	execCombinedOutput = func(context.Context, string, ...string) ([]byte, error) { return nil, nil }
	waitForDeviceHook = func(string, time.Duration) error { return nil }
	ensureDevicePermsHook = func(string, string) error { return nil }
	t.Cleanup(func() {
		execCombinedOutput = oldExec
		waitForDeviceHook = oldWait
		ensureDevicePermsHook = oldEnsure
	})

	err := run([]string{
		"bind",
		"--number", "0",
		"--addr", "AA:BB:CC:DD:EE:FF",
		"--owner", "1000:1000",
	})
	if err != nil {
		t.Fatalf("run(bind) = %v", err)
	}
}

func TestRunBindValidation(t *testing.T) {
	if err := run([]string{"bind", "--number", "0", "--addr", "bad", "--owner", "1000:1000"}); err == nil {
		t.Fatal("expected invalid MAC error")
	}
}

func TestRunFixPermsValidation(t *testing.T) {
	if err := runFixPerms(nil); err == nil {
		t.Fatal("expected usage error")
	}
	if err := runFixPerms([]string{"--device", "/tmp/x"}); err == nil {
		t.Fatal("expected invalid device error")
	}
	old := ensureDevicePermsHook
	ensureDevicePermsHook = func(string, string) error { return nil }
	t.Cleanup(func() { ensureDevicePermsHook = old })
	if err := runFixPerms([]string{"--device", "/dev/rfcomm0", "--owner", "1000:1000"}); err != nil {
		t.Fatalf("runFixPerms() = %v", err)
	}
}
