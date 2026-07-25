//go:build linux

package bt

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverPolkitHelperPathConfigured(t *testing.T) {
	helper := filepath.Join(t.TempDir(), "rfcomm-helper")
	if err := os.WriteFile(helper, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := ConfigurePrivileges(string(PrivilegeModePolkit), helper); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ConfigurePrivileges(string(PrivilegeModeAuto), "") })

	if got := discoverPolkitHelperPath(); got != helper {
		t.Fatalf("discoverPolkitHelperPath() = %q, want %q", got, helper)
	}
}

func TestDiscoverPolkitHelperPathMissing(t *testing.T) {
	missing := "/nonexistent/rfcomm-helper"
	if err := ConfigurePrivileges(string(PrivilegeModePolkit), missing); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ConfigurePrivileges(string(PrivilegeModeAuto), "") })

	got := discoverPolkitHelperPath()
	if got == missing {
		t.Fatalf("discoverPolkitHelperPath() = missing configured path %q", got)
	}
}

func TestSudoRFCCOMMBindValidation(t *testing.T) {
	if err := sudoRFCCOMMBind("bad", "AA:BB:CC:DD:EE:FF", 15); err == nil {
		t.Fatal("expected invalid RFCOMM number error")
	}
	if err := sudoRFCCOMMBind("0", "bad-mac", 15); err == nil {
		t.Fatal("expected invalid MAC error")
	}
}

func TestSudoRFCCOMMBindSuccess(t *testing.T) {
	old := execSudoHook
	execSudoHook = func(args ...string) error {
		if len(args) < 4 || args[0] != "rfcomm" || args[1] != "bind" {
			t.Fatalf("args=%v", args)
		}
		return nil
	}
	t.Cleanup(func() { execSudoHook = old })
	if err := sudoRFCCOMMBind("0", "AA:BB:CC:DD:EE:FF", 15); err != nil {
		t.Fatalf("sudoRFCCOMMBind() = %v", err)
	}
}

func TestWarmupSudoAlreadyDone(t *testing.T) {
	sudoMu.Lock()
	oldDone := sudoWarmupDone
	oldValid := sudoTicketValid
	sudoWarmupDone = true
	sudoTicketValid = true
	sudoMu.Unlock()
	t.Cleanup(func() {
		sudoMu.Lock()
		sudoWarmupDone = oldDone
		sudoTicketValid = oldValid
		sudoMu.Unlock()
	})

	ok, err := WarmupSudo()
	if err != nil || !ok {
		t.Fatalf("WarmupSudo() = %v, %v", ok, err)
	}
}

func TestWarmupPrivilegesAuto(t *testing.T) {
	if err := ConfigurePrivileges(string(PrivilegeModeAuto), ""); err != nil {
		t.Fatal(err)
	}
	ok, err := WarmupPrivileges()
	if err != nil || ok {
		t.Fatalf("WarmupPrivileges(auto) = %v, %v", ok, err)
	}
}
