//go:build darwin

package app

import (
	"strings"
	"testing"
)

func TestValidateFlags(t *testing.T) {
	guiCapture := captureDirDefault(ProfileGUI)
	tests := []struct {
		name    string
		device  string
		addr    string
		channel int
		capture string
		trace   string
		wantErr bool
	}{
		{name: "empty device", device: "", channel: 15, capture: guiCapture},
		{name: "mac ref", device: "aa:bb:cc:dd:ee:ff", channel: 15, capture: guiCapture},
		{name: "rfcomm ref", device: "rfcomm:aa:bb:cc:dd:ee:ff:15", channel: 15, capture: guiCapture},
		{name: "linux path rejected", device: "/dev/rfcomm0", channel: 15, capture: guiCapture, wantErr: true},
		{name: "bad channel", device: "", channel: 0, capture: guiCapture, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateFlags(tt.device, tt.addr, tt.channel, tt.capture, tt.trace)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateFlags() err=%v wantErr=%v", err, tt.wantErr)
			}
		})
	}
}

func TestCaptureDirDefaultGUIUsesAppSupport(t *testing.T) {
	got := captureDirDefault(ProfileGUI)
	if got == "captures" {
		t.Fatalf("GUI capture default should not be relative captures, got %q", got)
	}
	if !strings.Contains(got, "tws_manager") {
		t.Fatalf("GUI capture default = %q, want tws_manager in path", got)
	}
}
