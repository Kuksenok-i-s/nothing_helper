package app

import (
	"time"

	"tws_manager/internal/security"
)

// Config holds validated CLI/runtime settings shared by TUI and Gio entrypoints.
type Config struct {
	RFCOMMDevice        string
	Address             string
	Channel             int
	TracePath           string
	CaptureDir          string
	PrivilegeMode       string
	PrivilegeHelperPath string
	ModelName           string
	AllowUnsafe         bool
	ProbeEnabled        bool
	LogRaw              bool
	QueryEvery          time.Duration
	AutoDiscover        bool
	Notify              bool
	PCPrimary           string
}

// ValidateFlags normalizes and validates flag values.
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
