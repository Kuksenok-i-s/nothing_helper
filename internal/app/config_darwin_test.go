//go:build darwin

package app

import "testing"

func TestValidateFlags(t *testing.T) {
	tests := []struct {
		name    string
		device  string
		addr    string
		channel int
		capture string
		trace   string
		wantErr bool
	}{
		{name: "empty device", device: "", channel: 15, capture: "captures"},
		{name: "mac ref", device: "aa:bb:cc:dd:ee:ff", channel: 15, capture: "captures"},
		{name: "rfcomm ref", device: "rfcomm:aa:bb:cc:dd:ee:ff:15", channel: 15, capture: "captures"},
		{name: "linux path rejected", device: "/dev/rfcomm0", channel: 15, capture: "captures", wantErr: true},
		{name: "bad channel", device: "", channel: 0, capture: "captures", wantErr: true},
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
