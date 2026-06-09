package connect

import (
	"testing"

	"tws_manager/internal/bt"
	"tws_manager/internal/security"
)

func TestDeviceFromAddress(t *testing.T) {
	dev, err := DeviceFromAddress("AA:BB:CC:DD:EE:FF", 15)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := security.NormalizeMAC("AA:BB:CC:DD:EE:FF")
	if dev.MAC != want {
		t.Fatalf("MAC=%q want %q", dev.MAC, want)
	}
	if dev.Channel != 15 {
		t.Fatalf("channel=%d want 15", dev.Channel)
	}
}

func TestDeviceFromAddressInvalid(t *testing.T) {
	_, err := DeviceFromAddress("not-a-mac", 15)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBestConnectedCandidate(t *testing.T) {
	t.Run("prefers connected SPP", func(t *testing.T) {
		devs := []bt.Device{
			{MAC: "AA", SPP: true},
			{MAC: "BB", Connected: true},
			{MAC: "CC", SPP: true, Connected: true},
		}
		got, ok := BestConnectedCandidate(devs)
		if !ok || got.MAC != "CC" {
			t.Fatalf("got %+v ok=%v, want CC", got, ok)
		}
	})
	t.Run("ignores disconnected SPP", func(t *testing.T) {
		devs := []bt.Device{
			{MAC: "AA", SPP: true},
			{MAC: "BB", Connected: true},
		}
		got, ok := BestConnectedCandidate(devs)
		if !ok || got.MAC != "BB" {
			t.Fatalf("got %+v ok=%v, want BB", got, ok)
		}
	})
	t.Run("empty when none connected", func(t *testing.T) {
		devs := []bt.Device{
			{MAC: "AA", SPP: true},
			{MAC: "ZZ"},
		}
		if _, ok := BestConnectedCandidate(devs); ok {
			t.Fatal("expected no connected candidate")
		}
	})
}
