package session

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tws_manager/internal/bt"
	"tws_manager/internal/security"
)

func TestNormalizeConnectMACInvalid(t *testing.T) {
	_, err := normalizeConnectMAC(bt.Device{MAC: "not-a-mac"})
	if err == nil {
		t.Fatal("expected error for invalid MAC")
	}
}

func TestPrepareConnectDeviceValidatesTransport(t *testing.T) {
	_, _, _, err := prepareConnectDevice(bt.Device{}, "/etc/passwd", 15)
	if err == nil {
		t.Fatal("expected transport validation error")
	}
}

func TestPrepareConnectDeviceValidatesChannel(t *testing.T) {
	_, _, _, err := prepareConnectDevice(bt.Device{MAC: "AA:BB:CC:DD:EE:FF"}, "/dev/rfcomm0", 0)
	if err == nil {
		t.Fatal("expected channel validation error")
	}
}

func TestPrepareConnectDeviceResolvesLabel(t *testing.T) {
	dev, ch, label, err := prepareConnectDevice(bt.Device{MAC: "AA:BB:CC:DD:EE:FF", Name: "Ear"}, "/dev/rfcomm0", 15)
	if err != nil {
		t.Fatal(err)
	}
	if label != "/dev/rfcomm0" {
		t.Fatalf("label = %q, want /dev/rfcomm0", label)
	}
	if ch != 15 {
		t.Fatalf("channel = %d, want 15", ch)
	}
	if dev.Name != "Ear" {
		t.Fatalf("name = %q", dev.Name)
	}
}

func TestPrepareConnectDeviceEnrichesFromConfig(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "devices.json")
	bt.SetConfigPathHook(func() string { return cfgPath })
	t.Cleanup(func() { bt.SetConfigPathHook(nil) })

	if err := bt.RememberDeviceMAC("/dev/rfcomm0", "AA:BB:CC:DD:EE:FF"); err != nil {
		t.Fatal(err)
	}
	bt.SetBluetoothInfoHook(func(string) (string, error) {
		return "Device AA:BB:CC:DD:EE:FF\n\tName: Nothing Ear\n", nil
	})
	t.Cleanup(func() { bt.SetBluetoothInfoHook(nil) })

	dev, _, label, err := prepareConnectDevice(bt.Device{}, "/dev/rfcomm0", 15)
	if err != nil {
		t.Fatal(err)
	}
	if dev.MAC != "AA:BB:CC:DD:EE:FF" || dev.Name != "Nothing Ear" || label != "/dev/rfcomm0" {
		t.Fatalf("dev=%+v label=%q", dev, label)
	}
}

func TestEnrichConnectDeviceNoMAC(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "devices.json")
	bt.SetConfigPathHook(func() string { return cfgPath })
	t.Cleanup(func() { bt.SetConfigPathHook(nil) })

	dev := enrichConnectDevice(bt.Device{Name: "Ear"}, "/dev/rfcomm0")
	if dev.MAC != "" {
		t.Fatalf("dev=%+v", dev)
	}
}

func TestConnectRejectsInvalidMAC(t *testing.T) {
	s := New(nil, false, false)
	err := s.Connect(bt.Device{MAC: "bad-mac"}, "/dev/rfcomm0", 15)
	if err == nil {
		t.Fatal("expected MAC validation error")
	}
}

func TestConnectRejectsInvalidTransport(t *testing.T) {
	s := New(nil, false, false)
	err := s.Connect(bt.Device{MAC: "AA:BB:CC:DD:EE:FF"}, "/tmp/not-rfcomm", 15)
	if err == nil {
		t.Fatal("expected transport validation error")
	}
}

func TestConnectRejectsInvalidChannel(t *testing.T) {
	s := New(nil, false, false)
	err := s.Connect(bt.Device{MAC: "AA:BB:CC:DD:EE:FF"}, "/dev/rfcomm0", 999)
	if err == nil {
		t.Fatal("expected channel validation error")
	}
}

func TestConnectOpenTransportFailure(t *testing.T) {
	s := New(nil, false, false)
	old := hookOpenTransport
	hookOpenTransport = func(string, string, int, bt.RFCOMMProgress) (bt.Transport, int, error) {
		return nil, 0, errors.New("open failed")
	}
	t.Cleanup(func() { hookOpenTransport = old })

	err := s.Connect(bt.Device{MAC: "AA:BB:CC:DD:EE:FF", Name: "Ear"}, "/dev/rfcomm0", 15)
	if err == nil || !strings.Contains(err.Error(), "open failed") {
		t.Fatalf("Connect() = %v, want open failed", err)
	}
}

func TestConnectSuccessViaHookedTransport(t *testing.T) {
	s := New(nil, false, false)
	f, err := os.CreateTemp(t.TempDir(), "rfcomm")
	if err != nil {
		t.Fatal(err)
	}

	old := hookOpenTransport
	hookOpenTransport = func(string, string, int, bt.RFCOMMProgress) (bt.Transport, int, error) {
		return bt.NewTestTransport(f, "AA:BB:CC:DD:EE:FF", 15, "/dev/rfcomm0"), 15, nil
	}
	t.Cleanup(func() { hookOpenTransport = old })

	err = s.Connect(bt.Device{MAC: "aa:bb:cc:dd:ee:ff", Name: "Ear"}, "/dev/rfcomm0", 15)
	if err != nil {
		t.Fatalf("Connect() = %v", err)
	}
	snap := s.Snapshot()
	if !snap.Connected {
		t.Fatal("expected connected snapshot")
	}
	want, _ := security.NormalizeMAC("aa:bb:cc:dd:ee:ff")
	if snap.Device.MAC != want {
		t.Fatalf("device MAC = %q, want %q", snap.Device.MAC, want)
	}
	_ = s.Close()
}
