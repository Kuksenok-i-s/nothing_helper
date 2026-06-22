//go:build linux

package bt

import (
	"errors"

	"tws_manager/internal/security"
)

// channelCandidates orders RFCOMM channels to try: preferred, then 15, then
// 1..63 excluding 15 and any already listed.
func channelCandidates(preferred int) []int {
	if preferred <= 0 {
		preferred = DefaultRFCOMMChannel
	}
	seen := map[int]struct{}{}
	out := make([]int, 0, security.MaxRFCOMMChannel())
	add := func(ch int) {
		if ch < security.MinRFCOMMChannel() || ch > security.MaxRFCOMMChannel() {
			return
		}
		if _, ok := seen[ch]; ok {
			return
		}
		seen[ch] = struct{}{}
		out = append(out, ch)
	}
	add(preferred)
	add(DefaultRFCOMMChannel)
	for ch := security.MinRFCOMMChannel(); ch <= security.MaxRFCOMMChannel(); ch++ {
		if ch != DefaultRFCOMMChannel {
			add(ch)
		}
	}
	return out
}

func shouldProbeNextChannel(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrRFCOMMPermission) ||
		errors.Is(err, ErrInvalidBluetoothMAC) {
		return false
	}
	if isRecoverableRFCOMMOpenError(err) {
		return true
	}
	if errors.Is(err, ErrRFCOMMBindFailed) ||
		errors.Is(err, ErrRFCOMMWaitFailed) ||
		errors.Is(err, ErrRFCOMMOpenFailed) ||
		errors.Is(err, ErrRFCOMMReviveFailed) ||
		errors.Is(err, ErrRFCOMMNoChannel) {
		return true
	}
	return false
}
