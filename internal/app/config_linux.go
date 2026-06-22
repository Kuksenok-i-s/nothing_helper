//go:build linux

package app

import (
	"tws_manager/internal/security"
)

// ValidateFlags normalizes and validates flag values on Linux.
func ValidateFlags(devicePath, address string, channel int, captureDir, tracePath string) (Config, error) {
	devPath, err := security.ValidateRFCOMMDevice(devicePath)
	if err != nil {
		return Config{}, err
	}
	if err := security.ValidateChannel(channel); err != nil {
		return Config{}, err
	}
	if address != "" {
		address, err = security.NormalizeMAC(address)
		if err != nil {
			return Config{}, err
		}
	}
	captureDirPath, err := security.ValidateWritablePath(captureDir)
	if err != nil {
		return Config{}, err
	}
	traceLogPath := tracePath
	if traceLogPath != "" {
		traceLogPath, err = security.ValidateWritablePath(traceLogPath)
		if err != nil {
			return Config{}, err
		}
	}
	return Config{
		RFCOMMDevice: devPath,
		Address:      address,
		Channel:      channel,
		TracePath:    traceLogPath,
		CaptureDir:   captureDirPath,
	}, nil
}
