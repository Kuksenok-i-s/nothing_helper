//go:build darwin

package audio

// IsDefaultOutputForMAC on macOS always returns true: IOBluetooth RFCOMM does not
// require the earbuds to be the system default audio output (unlike Linux/BlueZ).
func IsDefaultOutputForMAC(mac string) (bool, error) {
	_ = mac
	return true, nil
}
