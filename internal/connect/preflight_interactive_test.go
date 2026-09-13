package connect

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"nothing_helper/internal/bt"
	"nothing_helper/internal/session"
)

func TestReportAutoConnectErr(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "bluetooth", err: errWaitingForBluetooth, want: "Bluetooth disconnected"},
		{name: "audio", err: errWaitingForAudioOutput, want: "A2DP"},
		{name: "none", err: errNoCandidate, want: "no compatible TWS device found"},
		{name: "other", err: errors.New("other"), want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var msg string
			reportAutoConnectErr(func(s string) { msg = s }, tt.err)
			if tt.want == "" {
				if msg != "" {
					t.Fatalf("msg = %q, want empty", msg)
				}
				return
			}
			if msg == "" || !strings.Contains(msg, tt.want) {
				t.Fatalf("msg = %q, want substring %q", msg, tt.want)
			}
		})
	}
}

func TestSelectDiscoveredDevice(t *testing.T) {
	devices := []bt.Device{
		{MAC: "AA:BB:CC:DD:EE:01", Name: "Ear One"},
		{MAC: "AA:BB:CC:DD:EE:02", Name: "Ear Two"},
	}
	io := PreflightIO{
		Discover: func() ([]bt.Device, error) { return devices, nil },
		ReadLine: func() (string, error) { return "2", nil },
		Printf:   func(string, ...any) {},
	}
	dev, ok, err := selectDiscoveredDevice(15, io)
	if err != nil || !ok || dev.MAC != "AA:BB:CC:DD:EE:02" {
		t.Fatalf("dev=%+v ok=%v err=%v", dev, ok, err)
	}

	_, _, err = selectDiscoveredDevice(15, PreflightIO{})
	if err == nil {
		t.Fatal("expected discover required error")
	}

	ioEmpty := PreflightIO{
		Discover: func() ([]bt.Device, error) { return nil, nil },
		Printf:   func(string, ...any) {},
	}
	_, ok, err = selectDiscoveredDevice(15, ioEmpty)
	if err != nil || ok {
		t.Fatalf("empty discover = ok=%v err=%v", ok, err)
	}

	ioSkip := PreflightIO{
		Discover: func() ([]bt.Device, error) { return devices, nil },
		ReadLine: func() (string, error) { return "", nil },
		Printf:   func(string, ...any) {},
	}
	_, ok, err = selectDiscoveredDevice(15, ioSkip)
	if err != nil || ok {
		t.Fatalf("skip selection = ok=%v err=%v", ok, err)
	}

	ioDiscoverErr := PreflightIO{
		Discover: func() ([]bt.Device, error) { return nil, errors.New("discover failed") },
		Printf:   func(string, ...any) {},
	}
	_, ok, err = selectDiscoveredDevice(15, ioDiscoverErr)
	if err != nil || ok {
		t.Fatalf("discover err = ok=%v err=%v", ok, err)
	}
}

func TestSelectDeviceForRFCOMMFromAddress(t *testing.T) {
	dev, ok, err := SelectDeviceForRFCOMM("AA:BB:CC:DD:EE:FF", 15, PreflightIO{})
	if err != nil || !ok || dev.MAC != "AA:BB:CC:DD:EE:FF" {
		t.Fatalf("dev=%+v ok=%v err=%v", dev, ok, err)
	}
}

func TestSelectDeviceForRFCOMMInteractive(t *testing.T) {
	io := PreflightIO{
		Discover: func() ([]bt.Device, error) {
			return []bt.Device{{MAC: "AA:BB:CC:DD:EE:FF", Name: "Ear"}}, nil
		},
		ReadLine: func() (string, error) { return "1\n", nil },
		Printf:   func(string, ...any) {},
	}
	dev, ok, err := SelectDeviceForRFCOMM("", 15, io)
	if err != nil || !ok || dev.MAC != "AA:BB:CC:DD:EE:FF" {
		t.Fatalf("dev=%+v ok=%v err=%v", dev, ok, err)
	}
}

func TestSelectDeviceForRFCOMMSkipEmptyLine(t *testing.T) {
	io := PreflightIO{
		Discover: func() ([]bt.Device, error) {
			return []bt.Device{{MAC: "AA:BB:CC:DD:EE:FF", Name: "Ear"}}, nil
		},
		ReadLine: func() (string, error) { return "\n", nil },
		Printf:   func(string, ...any) {},
	}
	_, ok, err := SelectDeviceForRFCOMM("", 15, io)
	if err != nil || ok {
		t.Fatalf("ok=%v err=%v, want skip", ok, err)
	}
}

func TestSelectDeviceForRFCOMMDiscoverError(t *testing.T) {
	io := PreflightIO{
		Discover: func() ([]bt.Device, error) { return nil, errors.New("discover failed") },
		Printf:   func(string, ...any) {},
	}
	_, ok, err := SelectDeviceForRFCOMM("", 15, io)
	if err != nil || ok {
		t.Fatalf("ok=%v err=%v, want soft skip", ok, err)
	}
}

func TestPreflightRFCOMMExistingDevice(t *testing.T) {
	mgr := New(session.New(nil, false, false), Options{RFCOMMPath: "/dev/rfcomm97", Channel: 15})
	preflightTestHooksVar = &preflightTestHooks{
		rfcommExists: func(m *Manager) (bool, error) { return true, nil },
	}
	t.Cleanup(func() { preflightTestHooksVar = nil })

	dev, ok, err := PreflightRFCOMM(context.Background(), mgr, "AA:BB:CC:DD:EE:FF", PreflightIO{Printf: func(string, ...any) {}})
	if err != nil || !ok || dev.MAC != "AA:BB:CC:DD:EE:FF" {
		t.Fatalf("dev=%+v ok=%v err=%v", dev, ok, err)
	}
}

func TestPreflightRFCOMMBindAfterConfirm(t *testing.T) {
	mgr := New(session.New(nil, false, false), Options{RFCOMMPath: "/dev/rfcomm97", Channel: 15})
	var bound bool
	preflightTestHooksVar = &preflightTestHooks{
		rfcommExists: func(m *Manager) (bool, error) { return false, nil },
		bind: func(m *Manager, ctx context.Context, dev bt.Device) error {
			bound = true
			return nil
		},
	}
	t.Cleanup(func() { preflightTestHooksVar = nil })

	io := PreflightIO{
		Confirm: func(string) bool { return true },
		Printf:  func(string, ...any) {},
	}
	dev, ok, err := PreflightRFCOMM(context.Background(), mgr, "AA:BB:CC:DD:EE:FF", io)
	if err != nil || !ok || !bound || dev.MAC != "AA:BB:CC:DD:EE:FF" {
		t.Fatalf("dev=%+v ok=%v bound=%v err=%v", dev, ok, bound, err)
	}
}

func TestPreflightRFCOMMSkipOnDeclinedConfirm(t *testing.T) {
	mgr := New(session.New(nil, false, false), Options{RFCOMMPath: "/dev/rfcomm97", Channel: 15})
	preflightTestHooksVar = &preflightTestHooks{
		rfcommExists: func(m *Manager) (bool, error) { return false, nil },
	}
	t.Cleanup(func() { preflightTestHooksVar = nil })

	io := PreflightIO{
		Confirm: func(string) bool { return false },
		Printf:  func(string, ...any) {},
	}
	_, ok, err := PreflightRFCOMM(context.Background(), mgr, "AA:BB:CC:DD:EE:FF", io)
	if err != nil || ok {
		t.Fatalf("ok=%v err=%v, want declined skip", ok, err)
	}
}

func TestRunPreflightConnectAutoDiscoverSkip(t *testing.T) {
	mgr := New(session.New(nil, false, false), Options{RFCOMMPath: "/dev/rfcomm0", Channel: 15})
	called := false
	err := RunPreflightConnect(context.Background(), mgr, true, "", PreflightIO{}, func(context.Context, bt.Device) error {
		called = true
		return nil
	})
	if err != nil || called {
		t.Fatalf("err=%v called=%v, want skip", err, called)
	}
}

func TestRunPreflightConnectInvokesConnect(t *testing.T) {
	mgr := New(session.New(nil, false, false), Options{RFCOMMPath: "/dev/rfcomm97", Channel: 15})
	preflightTestHooksVar = &preflightTestHooks{
		rfcommExists: func(m *Manager) (bool, error) { return true, nil },
	}
	t.Cleanup(func() { preflightTestHooksVar = nil })

	done := make(chan bt.Device, 1)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	err := RunPreflightConnect(ctx, mgr, false, "AA:BB:CC:DD:EE:FF", PreflightIO{}, func(_ context.Context, dev bt.Device) error {
		done <- dev
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case dev := <-done:
		if dev.MAC != "AA:BB:CC:DD:EE:FF" {
			t.Fatalf("dev=%+v", dev)
		}
	case <-time.After(time.Second):
		t.Fatal("connect was not invoked")
	}
}

func TestPreflightRFCOMMExistsError(t *testing.T) {
	mgr := New(session.New(nil, false, false), Options{RFCOMMPath: "/dev/rfcomm97", Channel: 15})
	preflightTestHooksVar = &preflightTestHooks{
		rfcommExists: func(m *Manager) (bool, error) { return false, errors.New("stat failed") },
	}
	t.Cleanup(func() { preflightTestHooksVar = nil })

	_, _, err := PreflightRFCOMM(context.Background(), mgr, "", PreflightIO{Printf: func(string, ...any) {}})
	if err == nil {
		t.Fatal("expected rfcomm exists error")
	}
}

func TestPreflightRFCOMMBindError(t *testing.T) {
	mgr := New(session.New(nil, false, false), Options{RFCOMMPath: "/dev/rfcomm97", Channel: 15})
	preflightTestHooksVar = &preflightTestHooks{
		rfcommExists: func(m *Manager) (bool, error) { return false, nil },
		bind: func(m *Manager, ctx context.Context, dev bt.Device) error {
			return errors.New("bind failed")
		},
	}
	t.Cleanup(func() { preflightTestHooksVar = nil })

	io := PreflightIO{
		Confirm: func(string) bool { return true },
		Printf:  func(string, ...any) {},
	}
	_, ok, err := PreflightRFCOMM(context.Background(), mgr, "AA:BB:CC:DD:EE:FF", io)
	if err == nil || ok || !strings.Contains(err.Error(), "bind failed") {
		t.Fatalf("ok=%v err=%v, want bind error", ok, err)
	}
}

func TestPreflightRFCOMMDiscoverAndBind(t *testing.T) {
	mgr := New(session.New(nil, false, false), Options{RFCOMMPath: "/dev/rfcomm97", Channel: 15})
	var bound bool
	preflightTestHooksVar = &preflightTestHooks{
		rfcommExists: func(m *Manager) (bool, error) { return false, nil },
		bind: func(m *Manager, ctx context.Context, dev bt.Device) error {
			bound = true
			return nil
		},
	}
	t.Cleanup(func() { preflightTestHooksVar = nil })

	io := PreflightIO{
		Discover: func() ([]bt.Device, error) {
			return []bt.Device{{MAC: "AA:BB:CC:DD:EE:FF", Name: "Ear"}}, nil
		},
		ReadLine: func() (string, error) { return "1\n", nil },
		Confirm:  func(string) bool { return true },
		Printf:   func(string, ...any) {},
	}
	dev, ok, err := PreflightRFCOMM(context.Background(), mgr, "", io)
	if err != nil || !ok || !bound || dev.MAC != "AA:BB:CC:DD:EE:FF" {
		t.Fatalf("dev=%+v ok=%v bound=%v err=%v", dev, ok, bound, err)
	}
}
