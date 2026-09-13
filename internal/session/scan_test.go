package session

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"nothing_helper/internal/bt"
	"nothing_helper/internal/spp"
)

func TestRunQueryScanRejectsInvalidRange(t *testing.T) {
	s := New(nil, false, false)
	err := s.RunQueryScan(context.Background(), 0xC100, 0xC000, 200*time.Millisecond)
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestRunQueryScanCancelled(t *testing.T) {
	s := New(nil, false, false)
	f, err := os.CreateTemp(t.TempDir(), "rfcomm")
	if err != nil {
		t.Fatal(err)
	}
	dev := bt.Device{MAC: "AA:BB:CC:DD:EE:FF", Name: "Ear"}
	s.AttachTestLink(bt.NewTestTransport(f, dev.MAC, 15, "/dev/rfcomm0"), dev)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = s.RunQueryScan(ctx, 0xC001, 0xC002, 200*time.Millisecond)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("RunQueryScan() = %v, want context.Canceled", err)
	}
}

func TestRunQueryScanDisconnectsDuringScan(t *testing.T) {
	s := New(nil, false, false)
	f, err := os.CreateTemp(t.TempDir(), "rfcomm")
	if err != nil {
		t.Fatal(err)
	}
	dev := bt.Device{MAC: "AA:BB:CC:DD:EE:FF", Name: "Ear"}
	s.AttachTestLink(bt.NewTestTransport(f, dev.MAC, 15, "/dev/rfcomm0"), dev)
	s.mu.Lock()
	s.transport = nil
	s.mu.Unlock()

	err = s.RunQueryScan(context.Background(), 0xC001, 0xC001, 200*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "disconnected during scan") {
		t.Fatalf("RunQueryScan() = %v, want disconnect error", err)
	}
}

func TestRunQueryScanSendsOneCommand(t *testing.T) {
	spp.ResetFSN()
	s := New(nil, false, false)
	f, err := os.CreateTemp(t.TempDir(), "rfcomm")
	if err != nil {
		t.Fatal(err)
	}
	dev := bt.Device{MAC: "AA:BB:CC:DD:EE:FF", Name: "Ear"}
	s.AttachTestLink(bt.NewTestTransport(f, dev.MAC, 15, "/dev/rfcomm0"), dev)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := s.RunQueryScan(ctx, 0xC001, 0xC001, 200*time.Millisecond); err != nil {
		t.Fatalf("RunQueryScan() = %v", err)
	}
	s.mu.Lock()
	_, ok := s.pending[1]
	s.mu.Unlock()
	if !ok {
		t.Fatal("expected pending TX after scan command")
	}
}

func TestRunQueryScanTwoCommands(t *testing.T) {
	spp.ResetFSN()
	s := New(nil, false, false)
	f, err := os.CreateTemp(t.TempDir(), "rfcomm")
	if err != nil {
		t.Fatal(err)
	}
	dev := bt.Device{MAC: "AA:BB:CC:DD:EE:FF", Name: "Ear"}
	s.AttachTestLink(bt.NewTestTransport(f, dev.MAC, 15, "/dev/rfcomm0"), dev)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := s.RunQueryScan(ctx, 0xC001, 0xC002, 200*time.Millisecond); err != nil {
		t.Fatalf("RunQueryScan() = %v", err)
	}
}

func TestRunQueryScanSendFailure(t *testing.T) {
	s := New(nil, false, false)
	f, err := os.CreateTemp(t.TempDir(), "rfcomm")
	if err != nil {
		t.Fatal(err)
	}
	dev := bt.Device{MAC: "AA:BB:CC:DD:EE:FF", Name: "Ear"}
	s.AttachTestLink(bt.NewTestTransport(f, dev.MAC, 15, "/dev/rfcomm0"), dev)
	f.Close()

	err = s.RunQueryScan(context.Background(), 0xC001, 0xC001, 200*time.Millisecond)
	if err == nil {
		t.Fatal("expected send failure")
	}
}

func TestNormalizeBatteryPollInterval(t *testing.T) {
	if got := normalizeBatteryPollInterval(0); got != 0 {
		t.Fatalf("zero = %v", got)
	}
	if got := normalizeBatteryPollInterval(5 * time.Second); got != 30*time.Second {
		t.Fatalf("clamp = %v", got)
	}
	if got := normalizeBatteryPollInterval(time.Minute); got != time.Minute {
		t.Fatalf("minute = %v", got)
	}
}

func TestTryStartBatteryPollingIdempotent(t *testing.T) {
	s := New(nil, false, false)
	if !s.tryStartBatteryPolling() {
		t.Fatal("first acquire should succeed")
	}
	if s.tryStartBatteryPolling() {
		t.Fatal("second acquire should fail")
	}
	s.mu.Lock()
	s.batteryPolling = false
	s.mu.Unlock()
}

func TestStartBatteryPollingNoOpWhenDisabled(t *testing.T) {
	s := New(nil, false, false)
	s.StartBatteryPolling(context.Background(), 0)
	s.mu.Lock()
	polling := s.batteryPolling
	s.mu.Unlock()
	if polling {
		t.Fatal("battery polling should stay off")
	}
}

func TestRunBatteryPollLoopExitsOnCancel(t *testing.T) {
	s := New(nil, false, false)
	s.batteryPolling = true
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s.runBatteryPollLoop(ctx, 30*time.Second)
	s.mu.Lock()
	polling := s.batteryPolling
	s.mu.Unlock()
	if polling {
		t.Fatal("expected polling flag cleared")
	}
}
