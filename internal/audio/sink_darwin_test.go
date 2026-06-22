//go:build darwin

package audio

import "testing"

func TestIsDefaultOutputForMACDarwinSkipsGate(t *testing.T) {
	ok, err := IsDefaultOutputForMAC("aa:bb:cc:dd:ee:ff")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("darwin should skip audio-output gate for RFCOMM connect")
	}
}
