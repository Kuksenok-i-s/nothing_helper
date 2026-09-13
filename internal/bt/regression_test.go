//go:build linux

package bt

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestReviveClosesVerificationFD(t *testing.T) {
	f, e := os.CreateTemp(t.TempDir(), "verification")
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	oldExec, oldWait, oldEnsure, oldOpen := execCombinedOutput, rfcommWaitForDevice, rfcommEnsureAccess, rfcommOpenFile
	defer func() {
		execCombinedOutput = oldExec
		rfcommWaitForDevice = oldWait
		rfcommEnsureAccess = oldEnsure
		rfcommOpenFile = oldOpen
	}()
	execCombinedOutput = func(context.Context, string, ...string) ([]byte, error) { return nil, nil }
	rfcommWaitForDevice = func(string, time.Duration) error { return nil }
	rfcommEnsureAccess = func(string) error { return nil }
	rfcommOpenFile = func(string, time.Duration) (*os.File, error) { return f, nil }
	if e := ReviveRFCOMMDevice("/dev/rfcomm0", "AA:BB:CC:DD:EE:FF", 15, nil); e != nil {
		t.Fatal(e)
	}
	if _, e := f.Stat(); e == nil {
		t.Fatal("verification FD is still open after successful revive")
	}
}
