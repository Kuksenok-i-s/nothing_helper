//go:build linux

package audio

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

func TestPactlDefaultSink(t *testing.T) {
	oldLook := execLookPath
	oldOut := execCommandOutput
	t.Cleanup(func() {
		execLookPath = oldLook
		execCommandOutput = oldOut
	})
	execLookPath = func(name string) (string, error) {
		if name == "pactl" {
			return "/usr/bin/pactl", nil
		}
		return "", errors.New("missing")
	}
	execCommandOutput = func(_ context.Context, name string, args ...string) ([]byte, error) {
		if name == "pactl" && len(args) == 1 && args[0] == "get-default-sink" {
			return []byte("bluez_output.AA_BB\n"), nil
		}
		return nil, errors.New("unexpected")
	}
	got, err := pactlDefaultSink(context.Background())
	if err != nil || got != "bluez_output.AA_BB" {
		t.Fatalf("pactlDefaultSink() = %q, %v", got, err)
	}
}

func TestWpctlDefaultSink(t *testing.T) {
	oldLook := execLookPath
	oldOut := execCommandOutput
	t.Cleanup(func() {
		execLookPath = oldLook
		execCommandOutput = oldOut
	})
	execLookPath = func(name string) (string, error) {
		if name == "wpctl" {
			return "/usr/bin/wpctl", nil
		}
		return "", errors.New("missing")
	}
	execCommandOutput = func(_ context.Context, name string, args ...string) ([]byte, error) {
		if name == "wpctl" {
			return []byte("node.name=\"bluez_output.2C_BE\"\n"), nil
		}
		return nil, errors.New("unexpected")
	}
	got, err := wpctlDefaultSink(context.Background())
	if err != nil || got != "bluez_output.2C_BE" {
		t.Fatalf("wpctlDefaultSink() = %q, %v", got, err)
	}
}

func TestDefaultPlaybackSinkPrefersPactl(t *testing.T) {
	oldLook := execLookPath
	oldOut := execCommandOutput
	t.Cleanup(func() {
		execLookPath = oldLook
		execCommandOutput = oldOut
	})
	execLookPath = func(name string) (string, error) {
		if name == "pactl" || name == "wpctl" {
			return "/usr/bin/" + name, nil
		}
		return "", errors.New("missing")
	}
	calls := 0
	execCommandOutput = func(_ context.Context, name string, args ...string) ([]byte, error) {
		calls++
		if name == "pactl" {
			return []byte("pactl-sink\n"), nil
		}
		return nil, fmt.Errorf("wpctl should not run")
	}
	if got := defaultPlaybackSink(context.Background()); got != "pactl-sink" {
		t.Fatalf("defaultPlaybackSink() = %q", got)
	}
	if calls != 1 {
		t.Fatalf("calls=%d want 1", calls)
	}
}

func TestDefaultPlaybackSinkFallsBackToWpctl(t *testing.T) {
	oldLook := execLookPath
	oldOut := execCommandOutput
	t.Cleanup(func() {
		execLookPath = oldLook
		execCommandOutput = oldOut
	})
	execLookPath = func(name string) (string, error) {
		if name == "pactl" {
			return "", errors.New("missing")
		}
		if name == "wpctl" {
			return "/usr/bin/wpctl", nil
		}
		return "", errors.New("missing")
	}
	execCommandOutput = func(_ context.Context, name string, _ ...string) ([]byte, error) {
		if name == "wpctl" {
			return []byte("node.name=\"wireplumber\"\n"), nil
		}
		return nil, errors.New("unexpected")
	}
	if got := defaultPlaybackSink(context.Background()); got != "wireplumber" {
		t.Fatalf("defaultPlaybackSink() = %q", got)
	}
}

func TestIsDefaultOutputForMAC(t *testing.T) {
	oldLook := execLookPath
	oldOut := execCommandOutput
	t.Cleanup(func() {
		execLookPath = oldLook
		execCommandOutput = oldOut
	})
	execLookPath = func(name string) (string, error) {
		if name == "pactl" {
			return "/usr/bin/pactl", nil
		}
		return "", errors.New("missing")
	}
	execCommandOutput = func(_ context.Context, name string, _ ...string) ([]byte, error) {
		if name == "pactl" {
			return []byte("bluez_output.2C_BE_EE_4A_EC_9E.1\n"), nil
		}
		return nil, errors.New("unexpected")
	}
	ok, err := IsDefaultOutputForMAC(context.Background(), "2c:be:ee:4a:ec:9e")
	if err != nil || !ok {
		t.Fatalf("IsDefaultOutputForMAC() = %v, %v", ok, err)
	}
}
