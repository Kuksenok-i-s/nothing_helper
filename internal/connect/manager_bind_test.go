//go:build linux

package connect

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"tws_manager/internal/bt"
	"tws_manager/internal/session"
)

func TestManagerBindValidation(t *testing.T) {
	mgr := New(session.New(nil, false, false), Options{RFCOMMPath: "/dev/rfcomm97", Channel: 15})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := mgr.Bind(ctx, bt.Device{MAC: "AA:BB:CC:DD:EE:FF"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Bind() = %v, want context.Canceled", err)
	}

	err = mgr.Bind(context.Background(), bt.Device{})
	if err == nil || err.Error() != "device MAC is required to bind /dev/rfcomm97" {
		t.Fatalf("Bind() = %v", err)
	}
}

func TestManagerBindSuccess(t *testing.T) {
	mgr := New(session.New(nil, false, false), Options{RFCOMMPath: "/dev/rfcomm97", Channel: 15})
	var bound bool
	managerBindHook = func(m *Manager, ctx context.Context, dev bt.Device) error {
		if dev.MAC != "AA:BB:CC:DD:EE:FF" {
			t.Fatalf("dev=%+v", dev)
		}
		bound = true
		return nil
	}
	t.Cleanup(func() { managerBindHook = nil })

	if err := mgr.Bind(context.Background(), bt.Device{MAC: "AA:BB:CC:DD:EE:FF", Channel: 15}); err != nil {
		t.Fatalf("Bind() = %v", err)
	}
	if !bound {
		t.Fatal("bind hook was not called")
	}
}

func TestManagerBindRealPath(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "rfcomm")
	if err != nil {
		t.Fatal(err)
	}
	path := tmp.Name()
	tmp.Close()
	device := "/dev/rfcomm97"
	cfgPath := filepath.Join(t.TempDir(), "devices.json")

	bt.SetConfigPathHook(func() string { return cfgPath })
	t.Cleanup(func() { bt.SetConfigPathHook(nil) })

	restore := bt.SetRFCOMMHooks(
		func(dev string, d time.Duration) (*os.File, error) {
			if dev == device {
				return os.OpenFile(path, os.O_RDWR, 0)
			}
			return nil, errors.New("unexpected device")
		},
		func(string, string, int) error { return nil },
	)
	t.Cleanup(restore)

	mgr := New(session.New(nil, false, false), Options{RFCOMMPath: device, Channel: 15})
	if err := mgr.Bind(context.Background(), bt.Device{MAC: "AA:BB:CC:DD:EE:FF", Channel: 15}); err != nil {
		t.Fatalf("Bind() = %v", err)
	}
	mac, ok := bt.LookupDeviceMAC(device)
	if !ok || mac != "AA:BB:CC:DD:EE:FF" {
		t.Fatalf("LookupDeviceMAC() = %q, %v", mac, ok)
	}
}
