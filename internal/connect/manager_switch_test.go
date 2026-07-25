//go:build linux

package connect

import (
	"context"
	"errors"
	"os"
	"testing"

	"tws_manager/internal/bt"
	"tws_manager/internal/session"
)

func TestSwitchToIdempotent(t *testing.T) {
	sess := session.New(nil, false, false)
	f, err := os.CreateTemp(t.TempDir(), "rfcomm")
	if err != nil {
		t.Fatal(err)
	}
	dev := bt.Device{MAC: "AA:BB:CC:DD:EE:FF", Name: "Ear", Channel: 15}
	sess.AttachTestLink(bt.NewTestTransport(f, dev.MAC, 15, "/dev/rfcomm97"), dev)
	mgr := New(sess, Options{RFCOMMPath: "/dev/rfcomm97", Channel: 15})

	if err := mgr.SwitchTo(context.Background(), dev); err != nil {
		t.Fatalf("SwitchTo() = %v", err)
	}
}

func TestSwitchToEmptyMAC(t *testing.T) {
	mgr := New(session.New(nil, false, false), Options{RFCOMMPath: "/dev/rfcomm97", Channel: 15})
	err := mgr.SwitchTo(context.Background(), bt.Device{})
	if err == nil || err.Error() != "device MAC is required to connect" {
		t.Fatalf("SwitchTo() = %v", err)
	}
}

func TestSwitchToCancelledContext(t *testing.T) {
	mgr := New(session.New(nil, false, false), Options{RFCOMMPath: "/dev/rfcomm97", Channel: 15})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := mgr.SwitchTo(ctx, bt.Device{MAC: "AA:BB:CC:DD:EE:FF"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("SwitchTo() = %v, want context.Canceled", err)
	}
}
