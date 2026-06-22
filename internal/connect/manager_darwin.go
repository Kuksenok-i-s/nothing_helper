//go:build darwin

package connect

import (
	"context"
	"fmt"

	"tws_manager/internal/bt"
	"tws_manager/internal/security"
)

// RFCOMMExists on Darwin reports true when a MAC or transport ref is configured.
func (m *Manager) RFCOMMExists() (bool, error) {
	if m.opts.RFCOMMPath == "" {
		return false, nil
	}
	if _, err := security.ValidateTransportRef(m.opts.RFCOMMPath); err == nil {
		return true, nil
	}
	if _, err := security.NormalizeMAC(m.opts.RFCOMMPath); err == nil {
		return true, nil
	}
	return false, nil
}

// Bind on Darwin is a no-op; RFCOMM opens directly via IOBluetooth.
func (m *Manager) Bind(ctx context.Context, device bt.Device) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if device.MAC == "" {
		return fmt.Errorf("device MAC is required to connect on darwin")
	}
	if err := bt.RememberDeviceMAC(device.MAC, device.MAC); err != nil {
		return fmt.Errorf("save device mapping: %w", err)
	}
	return nil
}

func (m *Manager) releaseTransportNode() error { return nil }
