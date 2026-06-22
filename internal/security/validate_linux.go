//go:build linux

package security

// ValidateTransportRef on Linux accepts only /dev/rfcommN device paths.
func ValidateTransportRef(ref string) (string, error) {
	return ValidateRFCOMMDevice(ref)
}
