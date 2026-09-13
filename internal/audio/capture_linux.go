//go:build linux

package audio

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"nothing_helper/internal/security"
)

// HasActiveCaptureForMAC checks the physical HFP node, not the persistent
// virtual source that PipeWire also exposes in A2DP mode.
func HasActiveCaptureForMAC(ctx context.Context, mac string) (bool, error) {
	mac, err := security.NormalizeMAC(mac)
	if err != nil {
		return false, err
	}
	raw, err := execCommandOutput(ctx, "pw-dump")
	if err != nil {
		return false, fmt.Errorf("проверка аудиопотока PipeWire: %w", err)
	}
	return activeCaptureFromDump(raw, mac)
}

func activeCaptureFromDump(raw []byte, mac string) (bool, error) {
	var objects []struct {
		Info struct {
			State string                     `json:"state"`
			Props map[string]json.RawMessage `json:"props"`
		} `json:"info"`
	}
	if err := json.Unmarshal(raw, &objects); err != nil {
		return false, err
	}
	for _, obj := range objects {
		prop := func(key string) string {
			var value string
			_ = json.Unmarshal(obj.Info.Props[key], &value)
			return value
		}
		if obj.Info.State != "running" || prop("media.class") != "Audio/Source" {
			continue
		}
		profile := prop("api.bluez5.profile")
		if profile != "headset-head-unit" {
			continue
		}
		if strings.EqualFold(prop("api.bluez5.address"), mac) {
			return true, nil
		}
	}
	return false, nil
}
