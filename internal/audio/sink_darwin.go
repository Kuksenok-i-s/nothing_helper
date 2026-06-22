//go:build darwin

package audio

// IsDefaultOutputForMAC is not implemented on macOS; returns false without error.
func IsDefaultOutputForMAC(mac string) (bool, error) {
	return false, nil
}
