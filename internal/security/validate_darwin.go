//go:build darwin

package security

import (
	"fmt"
	"strconv"
	"strings"
)

// ValidateTransportRef accepts rfcomm:MAC:CHANNEL or a bare MAC address.
func ValidateTransportRef(ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", nil
	}
	if mac, err := NormalizeMAC(ref); err == nil {
		return mac, nil
	}
	const prefix = "rfcomm:"
	if !strings.HasPrefix(strings.ToLower(ref), prefix) {
		return "", fmt.Errorf("transport ref must be MAC or rfcomm:MAC:CHANNEL, got %q", ref)
	}
	body := ref[len(prefix):]
	parts := strings.Split(body, ":")
	if len(parts) != 7 {
		return "", fmt.Errorf("transport ref must be rfcomm:MAC:CHANNEL, got %q", ref)
	}
	macPart := strings.Join(parts[:6], ":")
	chPart := parts[6]
	mac, err := NormalizeMAC(macPart)
	if err != nil {
		return "", fmt.Errorf("transport ref MAC: %w", err)
	}
	ch, err := strconv.Atoi(chPart)
	if err != nil {
		return "", fmt.Errorf("transport ref channel: %w", err)
	}
	if err := ValidateChannel(ch); err != nil {
		return "", err
	}
	return fmt.Sprintf("rfcomm:%s:%d", mac, ch), nil
}
