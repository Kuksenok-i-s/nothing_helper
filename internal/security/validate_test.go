package security

import (
	"strings"
	"testing"
)

func TestValidateMAC(t *testing.T) {
	if err := ValidateMAC("aa:bb:cc:dd:ee:ff"); err != nil {
		t.Fatal(err)
	}
	mac, err := NormalizeMAC("aa:bb:cc:dd:ee:ff")
	if err != nil || mac != "AA:BB:CC:DD:EE:FF" {
		t.Fatalf("NormalizeMAC() = %q, %v", mac, err)
	}
	if err := ValidateMAC("not-a-mac"); err == nil {
		t.Fatal("expected error for invalid MAC")
	}
}

func TestValidateRFCOMMDevice(t *testing.T) {
	tests := []struct {
		path    string
		want    string
		wantErr bool
	}{
		{path: "/dev/rfcomm0", want: "/dev/rfcomm0"},
		{path: "/dev/rfcomm12", want: "/dev/rfcomm12"},
		{path: "../../../etc/passwd", wantErr: true},
		{path: "/dev/ttyUSB0", wantErr: true},
		{path: "/dev/rfcomm", wantErr: true},
	}
	for _, tt := range tests {
		got, err := ValidateRFCOMMDevice(tt.path)
		if tt.wantErr {
			if err == nil {
				t.Fatalf("ValidateRFCOMMDevice(%q) expected error", tt.path)
			}
			continue
		}
		if err != nil || got != tt.want {
			t.Fatalf("ValidateRFCOMMDevice(%q) = %q, %v; want %q", tt.path, got, err, tt.want)
		}
	}
}

func TestValidateChannel(t *testing.T) {
	if err := ValidateChannel(15); err != nil {
		t.Fatal(err)
	}
	if err := ValidateChannel(0); err == nil {
		t.Fatal("expected error for channel 0")
	}
	if err := ValidateChannel(63); err != nil {
		t.Fatalf("63: %v", err)
	}
	if err := ValidateChannel(64); err == nil {
		t.Fatal("expected error for channel 64")
	}
}

func TestValidateWritablePath(t *testing.T) {
	dir := t.TempDir()
	abs, err := ValidateWritablePath(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(abs, "/") {
		t.Fatalf("expected absolute path, got %q", abs)
	}
	if _, err := ValidateWritablePath("/tmp/../etc/passwd"); err == nil {
		t.Fatal("expected error for traversal path")
	}
}
