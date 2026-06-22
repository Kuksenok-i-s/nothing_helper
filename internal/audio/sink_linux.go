//go:build linux

package audio

import (
	"fmt"
	"os/exec"
	"strings"

	"tws_manager/internal/security"
)

// IsDefaultOutputForMAC reports whether the system default playback sink
// belongs to the Bluetooth device with the given MAC (bluez_output.XX_...).
func IsDefaultOutputForMAC(mac string) (bool, error) {
	prefix, err := sinkPrefixForMAC(mac)
	if err != nil {
		return false, err
	}
	sink, err := defaultPlaybackSink()
	if err != nil {
		return false, err
	}
	if sink == "" {
		return false, nil
	}
	return strings.HasPrefix(sink, prefix), nil
}

func sinkPrefixForMAC(mac string) (string, error) {
	norm, err := security.NormalizeMAC(mac)
	if err != nil {
		return "", err
	}
	return "bluez_output." + strings.ReplaceAll(norm, ":", "_"), nil
}

func defaultPlaybackSink() (string, error) {
	if sink, err := pactlDefaultSink(); err == nil && sink != "" {
		return sink, nil
	}
	if sink, err := wpctlDefaultSink(); err == nil && sink != "" {
		return sink, nil
	}
	return "", nil
}

func pactlDefaultSink() (string, error) {
	if _, err := exec.LookPath("pactl"); err != nil {
		return "", err
	}
	out, err := exec.Command("pactl", "get-default-sink").Output()
	if err != nil {
		return "", fmt.Errorf("pactl get-default-sink: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func wpctlDefaultSink() (string, error) {
	if _, err := exec.LookPath("wpctl"); err != nil {
		return "", err
	}
	out, err := exec.Command("wpctl", "inspect", "@DEFAULT_AUDIO_SINK@").Output()
	if err != nil {
		return "", fmt.Errorf("wpctl inspect @DEFAULT_AUDIO_SINK@: %w", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok {
			continue
		}
		if strings.TrimSpace(key) == "node.name" {
			return strings.Trim(strings.TrimSpace(value), `"`), nil
		}
	}
	return "", fmt.Errorf("wpctl inspect: node.name not found")
}
