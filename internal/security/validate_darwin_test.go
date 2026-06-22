//go:build darwin

package security

import "testing"

func TestValidateTransportRef(t *testing.T) {
	tests := []struct {
		ref     string
		want    string
		wantErr bool
	}{
		{ref: "", want: ""},
		{ref: "aa:bb:cc:dd:ee:ff", want: "AA:BB:CC:DD:EE:FF"},
		{ref: "rfcomm:aa:bb:cc:dd:ee:ff:15", want: "rfcomm:AA:BB:CC:DD:EE:FF:15"},
		{ref: "/dev/rfcomm0", wantErr: true},
		{ref: "rfcomm:bad:15", wantErr: true},
		{ref: "rfcomm:aa:bb:cc:dd:ee:ff:0", wantErr: true},
	}
	for _, tt := range tests {
		got, err := ValidateTransportRef(tt.ref)
		if tt.wantErr {
			if err == nil {
				t.Fatalf("ValidateTransportRef(%q) expected error", tt.ref)
			}
			continue
		}
		if err != nil || got != tt.want {
			t.Fatalf("ValidateTransportRef(%q) = %q, %v; want %q", tt.ref, got, err, tt.want)
		}
	}
}
