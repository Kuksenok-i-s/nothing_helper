//go:build darwin

package bt

import (
	"errors"
	"strings"
)

// channelCandidates on macOS uses a single SDP-resolved channel.
// Probing multiple channels destabilizes IOBluetooth.
func channelCandidates(preferred int) []int {
	if preferred <= 0 {
		preferred = DefaultRFCOMMChannel
	}
	return []int{preferred}
}

func shouldProbeNextChannel(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrInvalidBluetoothMAC) {
		return false
	}
	msg := err.Error()
	// -5 RFCOMM open timed out
	if strings.Contains(msg, "code -2") || strings.Contains(msg, "code -3") || strings.Contains(msg, "code -5") {
		return false
	}
	return isRecoverableRFCOMMOpenError(err)
}
