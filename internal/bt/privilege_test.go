//go:build linux

package bt

import (
	"errors"
	"fmt"
	"os"
	"testing"
)

func TestParsePrivilegeMode(t *testing.T) {
	tests := []struct {
		in      string
		want    PrivilegeMode
		wantErr bool
	}{
		{in: "", want: PrivilegeModeSudo},
		{in: "sudo", want: PrivilegeModeSudo},
		{in: "polkit", want: PrivilegeModePolkit},
		{in: "auto", want: PrivilegeModeAuto},
		{in: "none", want: PrivilegeModeNone},
		{in: "bad", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := ParsePrivilegeMode(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParsePrivilegeMode(%q) err=%v wantErr=%v", tt.in, err, tt.wantErr)
			}
			if err == nil && got != tt.want {
				t.Fatalf("ParsePrivilegeMode(%q)=%q want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestPolkitCommandArgs(t *testing.T) {
	bind, err := polkitBindArgs("0", "AA:BB:CC:DD:EE:FF", 15)
	if err != nil {
		t.Fatalf("polkitBindArgs() err=%v", err)
	}
	if got, want := bind[0], "bind"; got != want {
		t.Fatalf("bind action=%q want %q", got, want)
	}
	wantOwner := fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid())
	if !containsArgPair(bind, "--owner", wantOwner) {
		t.Fatalf("bind args=%v want --owner %q", bind, wantOwner)
	}
	release, err := polkitReleaseArgs("0")
	if err != nil {
		t.Fatalf("polkitReleaseArgs() err=%v", err)
	}
	if got, want := release[0], "release"; got != want {
		t.Fatalf("release action=%q want %q", got, want)
	}
	fix, err := polkitFixPermsArgs("/dev/rfcomm0", "1000:1000")
	if err != nil {
		t.Fatalf("polkitFixPermsArgs() err=%v", err)
	}
	if got, want := fix[0], "fix-perms"; got != want {
		t.Fatalf("fix action=%q want %q", got, want)
	}
}

func containsArgPair(args []string, key, value string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == key && args[i+1] == value {
			return true
		}
	}
	return false
}

func TestWithPrivilegeFallbackModes(t *testing.T) {
	called := struct{ polkit, sudo int }{}
	polkitFn := func() error {
		called.polkit++
		if called.polkit < 0 {
			return errors.New("unreachable")
		}
		return nil
	}
	sudoFn := func() error {
		called.sudo++
		if called.sudo < 0 {
			return errors.New("unreachable")
		}
		return nil
	}

	t.Run("polkit", func(t *testing.T) {
		if err := ConfigurePrivileges("polkit", ""); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = ConfigurePrivileges("sudo", "") })
		called.polkit, called.sudo = 0, 0
		if err := withPrivilegeFallback(polkitFn, sudoFn); err != nil {
			t.Fatal(err)
		}
		if called.polkit != 1 || called.sudo != 0 {
			t.Fatalf("polkit=%d sudo=%d", called.polkit, called.sudo)
		}
	})

	t.Run("sudo", func(t *testing.T) {
		if err := ConfigurePrivileges("sudo", ""); err != nil {
			t.Fatal(err)
		}
		called.polkit, called.sudo = 0, 0
		if err := withPrivilegeFallback(polkitFn, sudoFn); err != nil {
			t.Fatal(err)
		}
		if called.polkit != 0 || called.sudo != 1 {
			t.Fatalf("polkit=%d sudo=%d", called.polkit, called.sudo)
		}
	})

	t.Run("none", func(t *testing.T) {
		if err := ConfigurePrivileges("none", ""); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = ConfigurePrivileges("sudo", "") })
		if err := withPrivilegeFallback(polkitFn, sudoFn); err == nil {
			t.Fatal("expected error for none mode")
		}
	})

	t.Run("auto polkit ok", func(t *testing.T) {
		if err := ConfigurePrivileges("auto", ""); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = ConfigurePrivileges("sudo", "") })
		called.polkit, called.sudo = 0, 0
		if err := withPrivilegeFallback(polkitFn, sudoFn); err != nil {
			t.Fatal(err)
		}
		if called.polkit != 1 || called.sudo != 0 {
			t.Fatalf("polkit=%d sudo=%d", called.polkit, called.sudo)
		}
	})

	t.Run("auto polkit fail sudo ok", func(t *testing.T) {
		if err := ConfigurePrivileges("auto", ""); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = ConfigurePrivileges("sudo", "") })
		called.polkit, called.sudo = 0, 0
		failPolkit := func() error { called.polkit++; return fmt.Errorf("polkit down") }
		if err := withPrivilegeFallback(failPolkit, sudoFn); err != nil {
			t.Fatal(err)
		}
		if called.polkit != 1 || called.sudo != 1 {
			t.Fatalf("polkit=%d sudo=%d", called.polkit, called.sudo)
		}
	})
}
