//go:build linux

package bt

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

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

func TestOpenFileWithTimeoutRegularFile(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "rfcomm")
	if err != nil {
		t.Fatal(err)
	}
	path := tmp.Name()
	tmp.Close()

	f, err := openFileWithTimeout(path, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("openFileWithTimeout(%q) = %v", path, err)
	}
	defer f.Close()
	if _, err := f.Write([]byte("ping")); err != nil {
		t.Fatalf("write after open: %v", err)
	}
}

func TestOpenFileWithTimeoutMissingPath(t *testing.T) {
	_, err := openFileWithTimeout("/nonexistent/rfcomm99", 100*time.Millisecond)
	if err == nil {
		t.Fatal("expected error for missing path")
	}
	if !os.IsNotExist(err) {
		t.Fatalf("expected not-exist error, got %v", err)
	}
}

func TestRunCommandIncludesOutputInError(t *testing.T) {
	_, err := runCommand("sh", "-c", "echo boom >&2; exit 3")
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "boom") || !strings.Contains(msg, "sh -c") {
		t.Fatalf("error should carry command line and output, got %q", msg)
	}
}

func TestRunCommandSuccessReturnsOutput(t *testing.T) {
	out, err := runCommand("sh", "-c", "echo ok")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(out)) != "ok" {
		t.Fatalf("out = %q", out)
	}
}

func TestDeviceConnectedFromInfo(t *testing.T) {
	tests := []struct {
		name string
		info string
		want bool
	}{
		{name: "connected yes", info: "Connected: yes\nAlias: Ear\n", want: true},
		{name: "connected no", info: "Connected: no\n", want: false},
		{name: "missing", info: "Alias: Ear\n", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := deviceConnectedFromInfo(tt.info); got != tt.want {
				t.Fatalf("deviceConnectedFromInfo() = %v, want %v", got, tt.want)
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

func TestShouldProbeNextChannel(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "bind failed", err: wrapRFCOMMBind(fmt.Errorf("rfcomm bind 0")), want: true},
		{name: "open failed", err: wrapRFCOMMOpen(fmt.Errorf(`open "/dev/rfcomm0" timed out`)), want: true},
		{name: "permission", err: wrapRFCOMMPermission(syscall.EACCES), want: false},
		{name: "invalid mac", err: fmt.Errorf("%w: bad mac", ErrInvalidBluetoothMAC), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldProbeNextChannel(tt.err); got != tt.want {
				t.Fatalf("shouldProbeNextChannel(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestParseBluetoothctlOutput(t *testing.T) {
	fields := ParseBluetoothctlOutput("Device AA:BB:CC:DD:EE:FF\n\tName: Nothing Ear (a)\n\tAlias: Ear A\n\tConnected: yes\n")
	if fields["name"] != "Nothing Ear (a)" {
		t.Fatalf("name = %q", fields["name"])
	}
	if fields["alias"] != "Ear A" {
		t.Fatalf("alias = %q", fields["alias"])
	}
	if fields["connected"] != "yes" {
		t.Fatalf("connected = %q", fields["connected"])
	}
}

func TestLoadConfigUsesCache(t *testing.T) {
	path := t.TempDir() + "/devices.json"
	if err := SaveConfig(path, Config{Devices: map[string]string{"/dev/rfcomm0": "AA:BB:CC:DD:EE:FF"}}); err != nil {
		t.Fatal(err)
	}
	cfg1, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	_ = os.Remove(path)
	cfg2, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg1.Devices["/dev/rfcomm0"] != cfg2.Devices["/dev/rfcomm0"] {
		t.Fatalf("cached config mismatch: %+v vs %+v", cfg1, cfg2)
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
