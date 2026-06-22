//go:build linux

package audio

import (
	"strings"
	"testing"
)

func TestSinkPrefixForMAC(t *testing.T) {
	got, err := sinkPrefixForMAC("2c:be:ee:4a:ec:9e")
	if err != nil {
		t.Fatal(err)
	}
	want := "bluez_output.2C_BE_EE_4A_EC_9E"
	if got != want {
		t.Fatalf("prefix=%q want %q", got, want)
	}
}

func TestDefaultSinkMatchesMACPrefix(t *testing.T) {
	mac := "2C:BE:EE:4A:EC:9E"
	prefix, err := sinkPrefixForMAC(mac)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		sink string
		want bool
	}{
		{sink: "bluez_output.2C_BE_EE_4A_EC_9E.1", want: true},
		{sink: "alsa_output.pci-0000_0d_00.4.iec958-stereo", want: false},
		{sink: "", want: false},
	}
	for _, tt := range tests {
		got := strings.HasPrefix(tt.sink, prefix)
		if got != tt.want {
			t.Fatalf("sink %q match=%v want %v", tt.sink, got, tt.want)
		}
	}
}
