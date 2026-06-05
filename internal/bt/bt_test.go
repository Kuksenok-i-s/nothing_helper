package bt

import (
	"errors"
	"fmt"
	"syscall"
	"testing"

	"tws_manager/internal/security"
)

func TestIsRecoverableRFCOMMOpenError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "timeout message", err: fmt.Errorf(`open "/dev/rfcomm0" timed out after 5s`), want: true},
		{name: "EIO", err: syscall.EIO, want: true},
		{name: "ENXIO", err: syscall.ENXIO, want: true},
		{name: "permission", err: syscall.EACCES, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRecoverableRFCOMMOpenError(tt.err); got != tt.want {
				t.Fatalf("isRecoverableRFCOMMOpenError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestIsRFCOMMNotBoundError(t *testing.T) {
	if !isRFCOMMNotBoundOutput("Can't release device: Not bound") {
		t.Fatal("expected not bound output to be recognized")
	}
	if isRFCOMMNotBoundError(errors.New("some other failure")) {
		t.Fatal("unexpected not bound classification")
	}
}

func TestApplyBluetoothInfo(t *testing.T) {
	dev := Device{}
	applyBluetoothInfo(&dev, "Device AA:BB:CC:DD:EE:FF\n\tName: Nothing Ear (a)\n\tAlias: Ear A\n")
	if dev.Name != "Nothing Ear (a)" {
		t.Fatalf("Name = %q", dev.Name)
	}
	if dev.Alias != "Ear A" {
		t.Fatalf("Alias = %q", dev.Alias)
	}
	if !isCandidate(dev) {
		t.Fatal("expected alias/name to classify device as candidate")
	}

	dev = Device{MAC: "AA:BB:CC:DD:EE:FF", Name: "AA:BB:CC:DD:EE:FF"}
	applyBluetoothInfo(&dev, "Device AA:BB:CC:DD:EE:FF\n\tName: Nothing Ear (3)\n")
	if dev.Name != "Nothing Ear (3)" {
		t.Fatalf("Name with MAC fallback = %q", dev.Name)
	}
}

func TestSanitizeConfig(t *testing.T) {
	cfg := sanitizeConfig(Config{Devices: map[string]string{
		"/dev/rfcomm0": "aa:bb:cc:dd:ee:ff",
		"/etc/passwd":  "aa:bb:cc:dd:ee:ff",
		"/dev/rfcomm1": "not-a-mac",
		"/dev/ttyUSB0": "11:22:33:44:55:66",
	}})
	if len(cfg.Devices) != 1 {
		t.Fatalf("len(devices) = %d, want 1", len(cfg.Devices))
	}
	if cfg.Devices["/dev/rfcomm0"] != "AA:BB:CC:DD:EE:FF" {
		t.Fatalf("mac = %q", cfg.Devices["/dev/rfcomm0"])
	}
}

func TestSudoDeviceChownRejectsNonRFCOMM(t *testing.T) {
	if err := sudoDeviceChown("/etc/passwd", "1000:1000"); err == nil {
		t.Fatal("expected error for chown on non-rfcomm path")
	}
	if _, err := security.ValidateRFCOMMDevice("/etc/passwd"); err == nil {
		t.Fatal("expected validation error")
	}
}
