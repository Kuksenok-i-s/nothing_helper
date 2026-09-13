//go:build linux

package connect

import (
	"context"
	"os"
	"testing"

	"nothing_helper/internal/bt"
	"nothing_helper/internal/session"
)

func TestManagerConnectValidation(t *testing.T) {
	mgr := New(session.New(nil, false, false), Options{RFCOMMPath: "/dev/rfcomm0", Channel: 15})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := mgr.Connect(ctx, bt.Device{MAC: "AA:BB:CC:DD:EE:FF"})
	if err == nil {
		t.Fatal("expected cancelled context")
	}
}

func TestManagerRFCOMMExists(t *testing.T) {
	mgr := New(session.New(nil, false, false), Options{RFCOMMPath: "/dev/rfcomm0", Channel: 15})
	exists, err := mgr.RFCOMMExists()
	if err != nil {
		t.Fatalf("RFCOMMExists() = %v", err)
	}
	_ = exists
}

func TestManagerRFCOMMExistsTempNode(t *testing.T) {
	path := t.TempDir() + "/rfcomm-test"
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	mgr := New(session.New(nil, false, false), Options{RFCOMMPath: path, Channel: 15})
	exists, err := mgr.RFCOMMExists()
	if err != nil || !exists {
		t.Fatalf("exists=%v err=%v", exists, err)
	}
}
