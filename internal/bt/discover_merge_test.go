//go:build linux

package bt

import "testing"

func TestMergeBluetoothLists(t *testing.T) {
	paired := []Device{{MAC: "AA:BB:CC:DD:EE:FF", Name: "Ear", Paired: true}}
	connected := []Device{{MAC: "AA:BB:CC:DD:EE:FF", Connected: true}, {MAC: "11:22:33:44:55:66", Name: "Buds", Connected: true}}
	merged := mergeBluetoothLists(paired, connected)
	if len(merged) != 2 {
		t.Fatalf("len = %d, want 2", len(merged))
	}
	aa := merged["AA:BB:CC:DD:EE:FF"]
	if !aa.Paired || !aa.Connected || aa.Name != "Ear" {
		t.Fatalf("AA device = %+v", aa)
	}
}

func TestEnrichDiscoveredDevice(t *testing.T) {
	dev := Device{MAC: "AA:BB:CC:DD:EE:FF"}
	info := "Device AA:BB:CC:DD:EE:FF\n\tName: Nothing Ear\n\tUUID: " + NothingSPPUUID + "\n"
	got := enrichDiscoveredDevice(dev, info)
	if !got.SPP || got.Name != "Nothing Ear" {
		t.Fatalf("device = %+v", got)
	}
	if !isDiscoveredCandidate(got) {
		t.Fatal("expected candidate")
	}
}

func TestValidateOpenRFCOMMParams(t *testing.T) {
	_, _, _, err := validateOpenRFCOMMParams("/etc/passwd", "AA:BB:CC:DD:EE:FF", 15)
	if err == nil {
		t.Fatal("expected invalid device")
	}
	dev, mac, ch, err := validateOpenRFCOMMParams("/dev/rfcomm0", "aa:bb:cc:dd:ee:ff", 15)
	if err != nil || dev != "/dev/rfcomm0" || mac != "AA:BB:CC:DD:EE:FF" || ch != 15 {
		t.Fatalf("got dev=%q mac=%q ch=%d err=%v", dev, mac, ch, err)
	}
}
