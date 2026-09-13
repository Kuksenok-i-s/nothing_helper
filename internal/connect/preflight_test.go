package connect

import (
	"testing"

	"nothing_helper/internal/bt"
)

func TestParseDeviceSelection(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		count   int
		want    int
		wantErr bool
	}{
		{name: "empty skip", line: "", count: 3, want: 0},
		{name: "valid first", line: "1", count: 3, want: 1},
		{name: "valid last", line: "3", count: 3, want: 3},
		{name: "invalid text", line: "abc", count: 3, wantErr: true},
		{name: "out of range", line: "4", count: 3, wantErr: true},
		{name: "zero", line: "0", count: 3, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDeviceSelection(tt.line, tt.count)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseDeviceSelection() err=%v wantErr=%v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("index = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestDeviceAtSelection(t *testing.T) {
	devs := []bt.Device{
		{MAC: "AA:BB:CC:DD:EE:FF", Name: "Ear A"},
		{MAC: "11:22:33:44:55:66", Name: "Ear B", Channel: 7},
	}
	got, err := DeviceAtSelection(1, devs, 15)
	if err != nil {
		t.Fatal(err)
	}
	if got.MAC != "AA:BB:CC:DD:EE:FF" || got.Channel != 15 {
		t.Fatalf("device = %+v", got)
	}
	got, err = DeviceAtSelection(2, devs, 15)
	if err != nil {
		t.Fatal(err)
	}
	if got.Channel != 7 {
		t.Fatalf("channel = %d, want preserved 7", got.Channel)
	}
	if _, err := DeviceAtSelection(9, devs, 15); err == nil {
		t.Fatal("expected error for out-of-range index")
	}
}

func TestResolvePreflightAddressExplicit(t *testing.T) {
	if got := ResolvePreflightAddress("AA:BB:CC:DD:EE:FF", "/dev/rfcomm0"); got != "AA:BB:CC:DD:EE:FF" {
		t.Fatalf("explicit address = %q", got)
	}
	if got := ResolvePreflightAddress("", ""); got != "" {
		t.Fatalf("empty = %q", got)
	}
}

func TestDeviceFromPreflightAddress(t *testing.T) {
	dev, ok, err := DeviceFromPreflightAddress("AA:BB:CC:DD:EE:FF", 15)
	if err != nil || !ok || dev.MAC != "AA:BB:CC:DD:EE:FF" {
		t.Fatalf("dev=%+v ok=%v err=%v", dev, ok, err)
	}
	_, ok, err = DeviceFromPreflightAddress("", 15)
	if err != nil || ok {
		t.Fatalf("empty address ok=%v err=%v", ok, err)
	}
}
