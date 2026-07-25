//go:build linux

package bt

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRememberAndLookupDeviceMAC(t *testing.T) {
	path := filepath.Join(t.TempDir(), "devices.json")
	old := configPathOverride
	configPathOverride = func() string { return path }
	t.Cleanup(func() { configPathOverride = old })

	if err := RememberDeviceMAC("/dev/rfcomm0", "aa:bb:cc:dd:ee:ff"); err != nil {
		t.Fatalf("RememberDeviceMAC() = %v", err)
	}
	mac, ok := LookupDeviceMAC("/dev/rfcomm0")
	if !ok || mac != "AA:BB:CC:DD:EE:FF" {
		t.Fatalf("LookupDeviceMAC() = %q, %v", mac, ok)
	}
}

func TestLookupDeviceMACMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "devices.json")
	old := configPathOverride
	configPathOverride = func() string { return path }
	t.Cleanup(func() { configPathOverride = old })

	if _, ok := LookupDeviceMAC("/dev/rfcomm0"); ok {
		t.Fatal("expected missing MAC")
	}
}

func TestEnrichDeviceInfoUsesBluetoothInfo(t *testing.T) {
	old := bluetoothInfoFn
	bluetoothInfoFn = func(string) (string, error) {
		return "Device AA:BB:CC:DD:EE:FF\n\tName: Nothing Ear\n", nil
	}
	t.Cleanup(func() { bluetoothInfoFn = old })

	dev := EnrichDeviceInfo(Device{MAC: "AA:BB:CC:DD:EE:FF", Name: "AA:BB:CC:DD:EE:FF"})
	if dev.Name != "Nothing Ear" {
		t.Fatalf("name=%q", dev.Name)
	}
}

func TestWaitForDeviceFindsExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rfcomm-ready")
	if err := writeTestFile(path); err != nil {
		t.Fatal(err)
	}
	if err := waitForDevice(path, 500); err != nil {
		t.Fatalf("waitForDevice() = %v", err)
	}
}

func writeTestFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	return f.Close()
}
