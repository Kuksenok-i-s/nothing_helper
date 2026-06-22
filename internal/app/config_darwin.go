//go:build darwin

package app

import (
	"tws_manager/internal/security"
)

// ValidateFlags normalizes and validates flag values on Darwin.
func ValidateFlags(devicePath, address string, channel int, captureDir, tracePath string) (Config, error) {
	if err := security.ValidateChannel(channel); err != nil {
		return Config{}, err
	}
	normAddr := address
	if normAddr != "" {
		var err error
		normAddr, err = security.NormalizeMAC(normAddr)
		if err != nil {
			return Config{}, err
		}
	}
	transportRef := devicePath
	if transportRef != "" {
		if _, err := security.ValidateTransportRef(transportRef); err != nil {
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
		RFCOMMDevice: transportRef,
		Address:      normAddr,
		Channel:      channel,
		TracePath:    traceLogPath,
		CaptureDir:   captureDirPath,
	}, nil
}
