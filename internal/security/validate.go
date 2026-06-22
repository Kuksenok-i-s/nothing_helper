package security

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const (
	minRFCOMMChannel = 1
	maxRFCOMMChannel = 63
)

var (
	macPattern       = regexp.MustCompile(`^([0-9A-Fa-f]{2}:){5}[0-9A-Fa-f]{2}$`)
	rfcommNumPattern = regexp.MustCompile(`^[0-9]+$`)
)

// NormalizeMAC returns uppercase colon-separated MAC or error.
// Accepts colon (Linux/BlueZ) and dash (macOS IOBluetooth) separators.
func NormalizeMAC(mac string) (string, error) {
	mac = strings.TrimSpace(mac)
	mac = strings.ReplaceAll(mac, "-", ":")
	if !macPattern.MatchString(mac) {
		return "", fmt.Errorf("invalid Bluetooth MAC %q", mac)
	}
	return strings.ToUpper(mac), nil
}

// ValidateMAC checks MAC format.
func ValidateMAC(mac string) error {
	_, err := NormalizeMAC(mac)
	return err
}

// ValidateRFCOMMDevice ensures path is exactly /dev/rfcomm<N>.
func ValidateRFCOMMDevice(device string) (string, error) {
	clean := filepath.Clean(device)
	if clean != device && !strings.HasPrefix(device, "/dev/") {
		clean = filepath.Clean("/" + strings.TrimPrefix(device, string(filepath.Separator)))
	}
	if !strings.HasPrefix(clean, "/dev/rfcomm") {
		return "", fmt.Errorf("RFCOMM device must be /dev/rfcomm<N>, got %q", device)
	}
	num := strings.TrimPrefix(filepath.Base(clean), "rfcomm")
	if num == "" || !rfcommNumPattern.MatchString(num) {
		return "", fmt.Errorf("invalid RFCOMM device %q", device)
	}
	return clean, nil
}

// RFCOMMNumber returns the numeric suffix for /dev/rfcommN.
func RFCOMMNumber(device string) (string, error) {
	path, err := ValidateRFCOMMDevice(device)
	if err != nil {
		return "", err
	}
	return strings.TrimPrefix(filepath.Base(path), "rfcomm"), nil
}

// MinRFCOMMChannel is the lowest valid RFCOMM server channel number.
func MinRFCOMMChannel() int { return minRFCOMMChannel }

// MaxRFCOMMChannel is the highest valid RFCOMM server channel number.
func MaxRFCOMMChannel() int { return maxRFCOMMChannel }

// ValidateChannel checks RFCOMM channel range.
func ValidateChannel(channel int) error {
	if channel < minRFCOMMChannel || channel > maxRFCOMMChannel {
		return fmt.Errorf("RFCOMM channel must be %d..%d, got %d", minRFCOMMChannel, maxRFCOMMChannel, channel)
	}
	return nil
}

// ValidateWritablePath resolves an absolute path safe for logs/captures (no .. segments).
func ValidateWritablePath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("path must not be empty")
	}
	if strings.Contains(path, "..") {
		return "", fmt.Errorf("path must not contain .. segments: %q", path)
	}
	if !filepath.IsAbs(path) {
		abs, err := filepath.Abs(path)
		if err != nil {
			return "", fmt.Errorf("resolve path %q: %w", path, err)
		}
		path = abs
	}
	clean := filepath.Clean(path)
	if strings.Contains(clean, "..") {
		return "", fmt.Errorf("path must not contain .. segments: %q", path)
	}
	return clean, nil
}

// ValidateRFCOMMNumber validates rfcomm CLI numeric argument.
func ValidateRFCOMMNumber(num string) error {
	if !rfcommNumPattern.MatchString(num) {
		return fmt.Errorf("invalid rfcomm number %q", num)
	}
	if _, err := strconv.Atoi(num); err != nil {
		return fmt.Errorf("invalid rfcomm number %q", num)
	}
	return nil
}
