//go:build linux

package bt

import (
	"context"
	"errors"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"os/exec"
)

func TestOpenRFCOMMAfterPermissionFix(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "rfcomm")
	if err != nil {
		t.Fatal(err)
	}
	path := tmp.Name()
	tmp.Close()

	oldOpen := rfcommOpenFile
	oldEnsure := rfcommEnsureAccess
	rfcommEnsureAccess = func(string) error { return nil }
	rfcommOpenFile = func(string, time.Duration) (*os.File, error) {
		return openFileWithTimeout(path, 100*time.Millisecond)
	}
	t.Cleanup(func() {
		rfcommOpenFile = oldOpen
		rfcommEnsureAccess = oldEnsure
	})

	f, err := openRFCOMMAfterPermissionFix("/dev/rfcomm0", nil)
	if err != nil {
		t.Fatalf("openRFCOMMAfterPermissionFix() = %v", err)
	}
	f.Close()
}

func TestOpenRFCOMMAfterRevive(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "rfcomm")
	if err != nil {
		t.Fatal(err)
	}
	path := tmp.Name()
	tmp.Close()

	oldOpen := rfcommOpenFile
	oldRevive := rfcommReviveDevice
	rfcommReviveDevice = func(string, string, int, RFCOMMProgress) error { return nil }
	rfcommOpenFile = func(string, time.Duration) (*os.File, error) {
		return openFileWithTimeout(path, 100*time.Millisecond)
	}
	t.Cleanup(func() {
		rfcommOpenFile = oldOpen
		rfcommReviveDevice = oldRevive
	})

	f, err := openRFCOMMAfterRevive("/dev/rfcomm0", "AA:BB:CC:DD:EE:FF", 15, nil, syscall.EIO)
	if err != nil {
		t.Fatalf("openRFCOMMAfterRevive() = %v", err)
	}
	f.Close()
}

func TestCreateBindAndOpenRFCOMM(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "rfcomm")
	if err != nil {
		t.Fatal(err)
	}
	path := tmp.Name()
	tmp.Close()

	oldOpen := rfcommOpenFile
	oldWait := rfcommWaitForDevice
	oldBind := rfcommBindDevice
	rfcommBindDevice = func(string, string, int) error { return nil }
	rfcommWaitForDevice = func(string, time.Duration) error { return nil }
	rfcommOpenFile = func(string, time.Duration) (*os.File, error) {
		return openFileWithTimeout(path, 100*time.Millisecond)
	}
	t.Cleanup(func() {
		rfcommOpenFile = oldOpen
		rfcommWaitForDevice = oldWait
		rfcommBindDevice = oldBind
	})

	f, err := createBindAndOpenRFCOMM("/dev/rfcomm0", "AA:BB:CC:DD:EE:FF", 15, nil)
	if err != nil {
		t.Fatalf("createBindAndOpenRFCOMM() = %v", err)
	}
	f.Close()
}

func TestOpenRFCOMMOnChannelOpenExisting(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "rfcomm")
	if err != nil {
		t.Fatal(err)
	}
	path := tmp.Name()
	tmp.Close()

	oldOpen := rfcommOpenFile
	rfcommOpenFile = func(string, time.Duration) (*os.File, error) {
		return openFileWithTimeout(path, 100*time.Millisecond)
	}
	t.Cleanup(func() { rfcommOpenFile = oldOpen })

	f, err := openRFCOMMOnChannel("/dev/rfcomm0", "AA:BB:CC:DD:EE:FF", 15, nil)
	if err != nil {
		t.Fatalf("openRFCOMMOnChannel() = %v", err)
	}
	f.Close()
}

func TestBindRFCOMMDevicePlainSuccess(t *testing.T) {
	oldExec := execCombinedOutput
	oldWait := rfcommWaitForDevice
	oldEnsure := rfcommEnsureAccess
	execCombinedOutput = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		if name == "rfcomm" && args[0] == "bind" {
			return nil, nil
		}
		return nil, errors.New("unexpected")
	}
	rfcommWaitForDevice = func(string, time.Duration) error { return nil }
	rfcommEnsureAccess = func(string) error { return nil }
	t.Cleanup(func() {
		execCombinedOutput = oldExec
		rfcommWaitForDevice = oldWait
		rfcommEnsureAccess = oldEnsure
	})

	if err := BindRFCOMMDevice("/dev/rfcomm0", "AA:BB:CC:DD:EE:FF", 15); err != nil {
		t.Fatalf("BindRFCOMMDevice() = %v", err)
	}
}

func TestBindRFCOMMDevicePrivilegedFallback(t *testing.T) {
	oldExec := execCombinedOutput
	oldWait := rfcommWaitForDevice
	oldEnsure := rfcommEnsureAccess
	oldPriv := rfcommPrivilegedBind
	execCombinedOutput = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return nil, errors.New("bind failed")
	}
	rfcommPrivilegedBind = func(string, string, int) error { return nil }
	rfcommWaitForDevice = func(string, time.Duration) error { return nil }
	rfcommEnsureAccess = func(string) error { return nil }
	t.Cleanup(func() {
		execCombinedOutput = oldExec
		rfcommWaitForDevice = oldWait
		rfcommEnsureAccess = oldEnsure
		rfcommPrivilegedBind = oldPriv
	})

	if err := BindRFCOMMDevice("/dev/rfcomm0", "AA:BB:CC:DD:EE:FF", 15); err != nil {
		t.Fatalf("BindRFCOMMDevice() = %v", err)
	}
}

func TestOpenRFCOMMOnChannelPermissionPath(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "rfcomm")
	if err != nil {
		t.Fatal(err)
	}
	path := tmp.Name()
	tmp.Close()

	oldOpen := rfcommOpenFile
	oldEnsure := rfcommEnsureAccess
	calls := 0
	rfcommOpenFile = func(device string, timeout time.Duration) (*os.File, error) {
		calls++
		if calls == 1 {
			return nil, syscall.EACCES
		}
		return openFileWithTimeout(path, timeout)
	}
	rfcommEnsureAccess = func(string) error { return nil }
	t.Cleanup(func() {
		rfcommOpenFile = oldOpen
		rfcommEnsureAccess = oldEnsure
	})

	f, err := openRFCOMMOnChannel("/dev/rfcomm0", "AA:BB:CC:DD:EE:FF", 15, nil)
	if err != nil {
		t.Fatalf("openRFCOMMOnChannel() = %v", err)
	}
	f.Close()
}

func TestReleaseRFCOMMDeviceNotBound(t *testing.T) {
	oldExec := execCombinedOutput
	execCombinedOutput = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return []byte("Can't release device: Not bound"), errors.New("failed")
	}
	t.Cleanup(func() { execCombinedOutput = oldExec })

	if err := ReleaseRFCOMMDevice("/dev/rfcomm0"); err != nil {
		t.Fatalf("ReleaseRFCOMMDevice() = %v", err)
	}
}

func TestReleaseRFCOMMDeviceSuccess(t *testing.T) {
	oldExec := execCombinedOutput
	execCombinedOutput = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return nil, nil
	}
	t.Cleanup(func() { execCombinedOutput = oldExec })

	if err := ReleaseRFCOMMDevice("/dev/rfcomm0"); err != nil {
		t.Fatalf("ReleaseRFCOMMDevice() = %v", err)
	}
}

func TestReleaseRFCOMMDevicePrivilegedFallback(t *testing.T) {
	oldExec := execCombinedOutput
	oldPriv := rfcommPrivilegedRelease
	execCombinedOutput = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return []byte("release failed"), errors.New("permission denied")
	}
	rfcommPrivilegedRelease = func(string) error { return nil }
	t.Cleanup(func() {
		execCombinedOutput = oldExec
		rfcommPrivilegedRelease = oldPriv
	})

	if err := ReleaseRFCOMMDevice("/dev/rfcomm0"); err != nil {
		t.Fatalf("ReleaseRFCOMMDevice() = %v", err)
	}
}

func TestReleaseRFCOMMDevicePrivilegedNotBound(t *testing.T) {
	oldExec := execCombinedOutput
	oldPriv := rfcommPrivilegedRelease
	execCombinedOutput = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return []byte("release failed"), errors.New("permission denied")
	}
	rfcommPrivilegedRelease = func(string) error {
		return errors.New("can't release device: device not configured")
	}
	t.Cleanup(func() {
		execCombinedOutput = oldExec
		rfcommPrivilegedRelease = oldPriv
	})

	if err := ReleaseRFCOMMDevice("/dev/rfcomm0"); err != nil {
		t.Fatalf("ReleaseRFCOMMDevice() = %v, want success on not-bound priv", err)
	}
}

func TestReviveRFCOMMDeviceReleaseFailure(t *testing.T) {
	oldExec := execCombinedOutput
	execCombinedOutput = func(context.Context, string, ...string) ([]byte, error) {
		return []byte("denied"), errors.New("permission denied")
	}
	oldPriv := rfcommPrivilegedRelease
	rfcommPrivilegedRelease = func(string) error { return errors.New("priv failed") }
	t.Cleanup(func() {
		execCombinedOutput = oldExec
		rfcommPrivilegedRelease = oldPriv
	})

	err := ReviveRFCOMMDevice("/dev/rfcomm0", "AA:BB:CC:DD:EE:FF", 15, nil)
	if err == nil || !strings.Contains(err.Error(), "release") {
		t.Fatalf("ReviveRFCOMMDevice() = %v, want release error", err)
	}
}

func TestExecSudoTicketSuccess(t *testing.T) {
	oldOut := execCmdCombinedOutput
	sudoMu.Lock()
	oldValid := sudoTicketValid
	oldDone := sudoWarmupDone
	sudoTicketValid = true
	sudoWarmupDone = true
	sudoMu.Unlock()
	execCmdCombinedOutput = func(cmd *exec.Cmd) ([]byte, error) { return nil, nil }
	t.Cleanup(func() {
		execCmdCombinedOutput = oldOut
		sudoMu.Lock()
		sudoTicketValid = oldValid
		sudoWarmupDone = oldDone
		sudoMu.Unlock()
	})

	if err := execSudo("true"); err != nil {
		t.Fatalf("execSudo() = %v", err)
	}
}

func TestExecSudoTicketFailureIncludesOutput(t *testing.T) {
	oldOut := execCmdCombinedOutput
	sudoMu.Lock()
	oldValid := sudoTicketValid
	oldDone := sudoWarmupDone
	sudoTicketValid = true
	sudoWarmupDone = true
	sudoMu.Unlock()
	execCmdCombinedOutput = func(cmd *exec.Cmd) ([]byte, error) {
		return []byte("denied"), errors.New("exit 1")
	}
	t.Cleanup(func() {
		execCmdCombinedOutput = oldOut
		sudoMu.Lock()
		sudoTicketValid = oldValid
		sudoWarmupDone = oldDone
		sudoMu.Unlock()
	})

	err := execSudo("true")
	if err == nil || !strings.Contains(err.Error(), "denied") {
		t.Fatalf("execSudo() = %v, want denied output", err)
	}
}

func TestExecSudoPasswordProvider(t *testing.T) {
	oldOut := execCmdCombinedOutput
	sudoMu.Lock()
	oldValid := sudoTicketValid
	oldDone := sudoWarmupDone
	oldFn := sudoPasswordFn
	sudoTicketValid = false
	sudoWarmupDone = false
	sudoPasswordFn = func(string) (string, error) { return "secret", nil }
	sudoMu.Unlock()
	execCmdCombinedOutput = func(cmd *exec.Cmd) ([]byte, error) { return nil, nil }
	t.Cleanup(func() {
		execCmdCombinedOutput = oldOut
		sudoMu.Lock()
		sudoTicketValid = oldValid
		sudoWarmupDone = oldDone
		sudoPasswordFn = oldFn
		sudoMu.Unlock()
	})

	if err := execSudo("true"); err != nil {
		t.Fatalf("execSudo() = %v", err)
	}
}

func TestExecSudoPasswordProviderError(t *testing.T) {
	sudoMu.Lock()
	oldFn := sudoPasswordFn
	sudoTicketValid = false
	sudoPasswordFn = func(string) (string, error) { return "", errors.New("cancelled") }
	sudoMu.Unlock()
	t.Cleanup(func() {
		sudoMu.Lock()
		sudoPasswordFn = oldFn
		sudoMu.Unlock()
	})

	if err := execSudo("true"); err == nil {
		t.Fatal("expected password provider error")
	}
}

func TestExecSudoInteractive(t *testing.T) {
	oldRun := execCmdRun
	sudoMu.Lock()
	oldValid := sudoTicketValid
	oldDone := sudoWarmupDone
	oldFn := sudoPasswordFn
	sudoTicketValid = false
	sudoWarmupDone = false
	sudoPasswordFn = nil
	sudoMu.Unlock()
	execCmdRun = func(cmd *exec.Cmd) error { return nil }
	t.Cleanup(func() {
		execCmdRun = oldRun
		sudoMu.Lock()
		sudoTicketValid = oldValid
		sudoWarmupDone = oldDone
		sudoPasswordFn = oldFn
		sudoMu.Unlock()
	})

	if err := execSudo("true"); err != nil {
		t.Fatalf("execSudo() = %v", err)
	}
}

func TestExecSudoInteractiveFailure(t *testing.T) {
	oldRun := execCmdRun
	sudoMu.Lock()
	oldValid := sudoTicketValid
	oldDone := sudoWarmupDone
	oldFn := sudoPasswordFn
	sudoTicketValid = false
	sudoWarmupDone = false
	sudoPasswordFn = nil
	sudoMu.Unlock()
	execCmdRun = func(cmd *exec.Cmd) error { return errors.New("cancelled") }
	t.Cleanup(func() {
		execCmdRun = oldRun
		sudoMu.Lock()
		sudoTicketValid = oldValid
		sudoWarmupDone = oldDone
		sudoPasswordFn = oldFn
		sudoMu.Unlock()
	})

	if err := execSudo("true"); err == nil {
		t.Fatal("expected interactive failure")
	}
}

func TestReleaseRFCOMMDevicePrivilegedFailure(t *testing.T) {
	oldExec := execCombinedOutput
	oldPriv := rfcommPrivilegedRelease
	execCombinedOutput = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return []byte("release failed"), errors.New("permission denied")
	}
	rfcommPrivilegedRelease = func(string) error { return errors.New("priv failed") }
	t.Cleanup(func() {
		execCombinedOutput = oldExec
		rfcommPrivilegedRelease = oldPriv
	})

	err := ReleaseRFCOMMDevice("/dev/rfcomm0")
	if err == nil || !strings.Contains(err.Error(), "privileged fallback") {
		t.Fatalf("ReleaseRFCOMMDevice() = %v, want privileged fallback error", err)
	}
}

func TestReviveRFCOMMDeviceSuccess(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "rfcomm")
	if err != nil {
		t.Fatal(err)
	}
	path := tmp.Name()
	tmp.Close()
	device := "/dev/rfcomm0"

	oldExec := execCombinedOutput
	oldBind := rfcommBindDevice
	oldWait := rfcommWaitForDevice
	oldEnsure := rfcommEnsureAccess
	oldOpen := rfcommOpenFile
	execCombinedOutput = func(context.Context, string, ...string) ([]byte, error) { return nil, nil }
	rfcommBindDevice = func(string, string, int) error { return nil }
	rfcommWaitForDevice = func(string, time.Duration) error { return nil }
	rfcommEnsureAccess = func(string) error { return nil }
	rfcommOpenFile = func(dev string, d time.Duration) (*os.File, error) {
		if dev == device {
			return openFileWithTimeout(path, d)
		}
		return nil, errors.New("unexpected device")
	}
	t.Cleanup(func() {
		execCombinedOutput = oldExec
		rfcommBindDevice = oldBind
		rfcommWaitForDevice = oldWait
		rfcommEnsureAccess = oldEnsure
		rfcommOpenFile = oldOpen
	})

	if err := ReviveRFCOMMDevice(device, "AA:BB:CC:DD:EE:FF", 15, nil); err != nil {
		t.Fatalf("ReviveRFCOMMDevice() = %v", err)
	}
}

func TestEnsureRFCOMMDeviceAccessRejectsInvalidPath(t *testing.T) {
	if err := EnsureRFCOMMDeviceAccess("/tmp/not-rfcomm"); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestOpenRFCOMMOnChannelMissingAddress(t *testing.T) {
	oldOpen := rfcommOpenFile
	rfcommOpenFile = func(string, time.Duration) (*os.File, error) {
		return nil, &os.PathError{Op: "open", Path: "/dev/rfcomm0", Err: os.ErrNotExist}
	}
	t.Cleanup(func() { rfcommOpenFile = oldOpen })

	_, err := openRFCOMMOnChannel("/dev/rfcomm0", "", 15, nil)
	if err == nil || !strings.Contains(err.Error(), "pass --addr") {
		t.Fatalf("openRFCOMMOnChannel() = %v, want missing address error", err)
	}
}

func TestOpenRFCOMMOnChannelNonRecoverableError(t *testing.T) {
	oldOpen := rfcommOpenFile
	rfcommOpenFile = func(string, time.Duration) (*os.File, error) {
		return nil, errors.New("device busy")
	}
	t.Cleanup(func() { rfcommOpenFile = oldOpen })

	_, err := openRFCOMMOnChannel("/dev/rfcomm0", "AA:BB:CC:DD:EE:FF", 15, nil)
	if err == nil || !strings.Contains(err.Error(), "device busy") {
		t.Fatalf("openRFCOMMOnChannel() = %v, want busy error", err)
	}
}

func TestBindRFCOMMWithProbeSuccess(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "rfcomm")
	if err != nil {
		t.Fatal(err)
	}
	path := tmp.Name()
	tmp.Close()
	device := "/dev/rfcomm0"

	oldBind := rfcommBindDevice
	oldOpen := rfcommOpenFile
	oldExec := execCombinedOutput
	rfcommBindDevice = func(string, string, int) error { return nil }
	rfcommOpenFile = func(dev string, d time.Duration) (*os.File, error) {
		if dev == device {
			return openFileWithTimeout(path, d)
		}
		return nil, errors.New("unexpected device")
	}
	execCombinedOutput = func(context.Context, string, ...string) ([]byte, error) { return nil, nil }
	t.Cleanup(func() {
		rfcommBindDevice = oldBind
		rfcommOpenFile = oldOpen
		execCombinedOutput = oldExec
	})

	ch, err := BindRFCOMMWithProbe(device, "AA:BB:CC:DD:EE:FF", 15, nil)
	if err != nil || ch != 15 {
		t.Fatalf("BindRFCOMMWithProbe() = %d, %v", ch, err)
	}
}
