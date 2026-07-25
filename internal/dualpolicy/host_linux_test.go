//go:build linux

package dualpolicy

import (
	"context"
	"errors"
	"testing"
)

func TestHostAdapterMAC(t *testing.T) {
	old := execCombinedOutput
	execCombinedOutput = func(_ context.Context, name string, args ...string) ([]byte, error) {
		if name == "bluetoothctl" && len(args) == 1 && args[0] == "show" {
			return []byte("Controller AA:BB:CC:DD:EE:FF MyPC\n"), nil
		}
		return nil, errors.New("unexpected")
	}
	t.Cleanup(func() { execCombinedOutput = old })

	got, err := HostAdapterMAC()
	if err != nil || got != "AA:BB:CC:DD:EE:FF" {
		t.Fatalf("HostAdapterMAC() = %q, %v", got, err)
	}
}

func TestHostAdapterMACMissing(t *testing.T) {
	old := execCombinedOutput
	execCombinedOutput = func(context.Context, string, ...string) ([]byte, error) {
		return []byte("Alias MyPC\n"), nil
	}
	t.Cleanup(func() { execCombinedOutput = old })

	_, err := HostAdapterMAC()
	if err == nil {
		t.Fatal("expected missing controller error")
	}
}
