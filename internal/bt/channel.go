package bt

import (
	"fmt"
	"strings"
	"time"

	"tws_manager/internal/security"
)

const DefaultRFCOMMChannel = 15

// ResolveDeviceChannel returns a saved RFCOMM channel for mac when present,
// otherwise hint, otherwise DefaultRFCOMMChannel.
func ResolveDeviceChannel(mac string, hint int) int {
	if mac != "" {
		if ch, ok := LookupDeviceChannel(mac); ok {
			return ch
		}
	}
	if hint > 0 {
		return hint
	}
	return DefaultRFCOMMChannel
}

// LookupDeviceChannel returns a persisted RFCOMM channel for the given MAC.
func LookupDeviceChannel(mac string) (int, bool) {
	normMAC, err := security.NormalizeMAC(mac)
	if err != nil {
		return 0, false
	}
	cfg, err := LoadConfig(ConfigPath())
	if err != nil {
		return 0, false
	}
	ch, ok := cfg.Channels[normMAC]
	if !ok || ch <= 0 {
		return 0, false
	}
	return ch, true
}

// RememberDeviceChannel persists the working RFCOMM channel for a MAC address.
func RememberDeviceChannel(mac string, channel int) error {
	normMAC, err := security.NormalizeMAC(mac)
	if err != nil {
		return err
	}
	if err := security.ValidateChannel(channel); err != nil {
		return err
	}
	path := ConfigPath()
	cfg, err := LoadConfig(path)
	if err != nil {
		return err
	}
	if cfg.Channels == nil {
		cfg.Channels = map[string]int{}
	}
	cfg.Channels[normMAC] = channel
	return SaveConfig(path, cfg)
}

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
	if isRecoverableRFCOMMOpenError(err) {
		return true
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "permission") ||
		strings.Contains(msg, "privilege") ||
		strings.Contains(msg, "sudo") ||
		strings.Contains(msg, "invalid bluetooth mac") {
		return false
	}
	return strings.Contains(msg, "bind") ||
		strings.Contains(msg, "rfcomm") ||
		strings.Contains(msg, "timed out") ||
		strings.Contains(msg, "wait for") ||
		strings.Contains(msg, "revive failed") ||
		strings.Contains(msg, "create ") ||
		strings.Contains(msg, "no working")
}

// BindRFCOMMWithProbe binds device to address, trying alternate RFCOMM channels
// when the preferred channel fails. Returns the channel that worked.
func BindRFCOMMWithProbe(device, address string, preferred int, progress RFCOMMProgress) (int, error) {
	preferred = ResolveDeviceChannel(address, preferred)
	var lastErr error
	for i, ch := range channelCandidates(preferred) {
		if i > 0 {
			report(progress, fmt.Sprintf("bind: trying RFCOMM channel %d", ch))
			_ = ReleaseRFCOMMDevice(device)
		}
		if err := BindRFCOMMDevice(device, address, ch); err != nil {
			lastErr = err
			if shouldProbeNextChannel(err) {
				continue
			}
			return 0, err
		}
		if err := waitForDevice(device, 2*time.Second); err != nil {
			lastErr = err
			_ = ReleaseRFCOMMDevice(device)
			if shouldProbeNextChannel(err) {
				continue
			}
			return 0, err
		}
		_ = RememberDeviceChannel(address, ch)
		return ch, nil
	}
	return 0, fmt.Errorf("no RFCOMM channel worked for %s: %w", address, lastErr)
}
