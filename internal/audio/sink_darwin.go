//go:build darwin

package audio

import "context"

// IsDefaultOutputForMAC on macOS always returns true: IOBluetooth RFCOMM does not
// require the earbuds to be the system default audio output (unlike Linux/BlueZ).
func IsDefaultOutputForMAC(ctx context.Context, mac string) (bool, error) {
	_ = ctx
	_ = mac
	return true, nil
}
