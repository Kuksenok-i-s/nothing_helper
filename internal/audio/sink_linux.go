//go:build linux

package audio

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"tws_manager/internal/security"
)

var (
	execLookPath      = exec.LookPath
	execCommandOutput = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return exec.CommandContext(ctx, name, args...).Output()
	}
)

// IsDefaultOutputForMAC reports whether the system default playback sink
// belongs to the Bluetooth device with the given MAC (bluez_output.XX_...).
func IsDefaultOutputForMAC(ctx context.Context, mac string) (bool, error) {
	prefix, err := sinkPrefixForMAC(mac)
	if err != nil {
		return false, err
	}
	sink := defaultPlaybackSink(ctx)
	if sink == "" {
		return false, nil
	}
	return strings.HasPrefix(sink, prefix), nil
}

// HasBluetoothOutputForMAC reports whether Pulse/PipeWire exposes a bluez_output
// sink for the device. The buds need not be the system default — only available
// as an audio endpoint (typical once A2DP is up).
func HasBluetoothOutputForMAC(ctx context.Context, mac string) (bool, error) {
	prefix, err := sinkPrefixForMAC(mac)
	if err != nil {
		return false, err
	}
	sinks, err := listPlaybackSinks(ctx)
	if err != nil {
		// If we cannot enumerate sinks, do not block RFCOMM when BT is connected.
		return true, nil
	}
	for _, sink := range sinks {
		if strings.HasPrefix(sink, prefix) {
			return true, nil
		}
	}
	return false, nil
}

func sinkPrefixForMAC(mac string) (string, error) {
	norm, err := security.NormalizeMAC(mac)
	if err != nil {
		return "", err
	}
	return "bluez_output." + strings.ReplaceAll(norm, ":", "_"), nil
}

func defaultPlaybackSink(ctx context.Context) string {
	if sink, err := pactlDefaultSink(ctx); err == nil && sink != "" {
		return sink
	}
	if sink, err := wpctlDefaultSink(ctx); err == nil && sink != "" {
		return sink
	}
	return ""
}

func listPlaybackSinks(ctx context.Context) ([]string, error) {
	if sinks, err := pactlListSinks(ctx); err == nil {
		return sinks, nil
	}
	return nil, fmt.Errorf("no sink list backend available")
}

func pactlListSinks(ctx context.Context) ([]string, error) {
	if _, err := execLookPath("pactl"); err != nil {
		return nil, err
	}
	out, err := execCommandOutput(ctx, "pactl", "list", "short", "sinks")
	if err != nil {
		return nil, fmt.Errorf("pactl list short sinks: %w", err)
	}
	var sinks []string
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			sinks = append(sinks, fields[1])
		}
	}
	return sinks, nil
}

func pactlDefaultSink(ctx context.Context) (string, error) {
	if _, err := execLookPath("pactl"); err != nil {
		return "", err
	}
	out, err := execCommandOutput(ctx, "pactl", "get-default-sink")
	if err != nil {
		return "", fmt.Errorf("pactl get-default-sink: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func wpctlDefaultSink(ctx context.Context) (string, error) {
	if _, err := execLookPath("wpctl"); err != nil {
		return "", err
	}
	out, err := execCommandOutput(ctx, "wpctl", "inspect", "@DEFAULT_AUDIO_SINK@")
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
