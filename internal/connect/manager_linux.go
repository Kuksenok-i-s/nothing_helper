//go:build linux

package connect

import (
	"context"
	"fmt"
	"os"

	"tws_manager/internal/bt"
)

// RFCOMMExists reports whether the configured RFCOMM device node exists.
func (m *Manager) RFCOMMExists() (bool, error) {
	_, err := os.Stat(m.opts.RFCOMMPath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("check %q: %w", m.opts.RFCOMMPath, err)
}

// Bind creates the RFCOMM device node for the given Bluetooth device.
func (m *Manager) Bind(ctx context.Context, device bt.Device) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if device.MAC == "" {
		return fmt.Errorf("device MAC is required to bind %s", m.opts.RFCOMMPath)
	}
	ch := bt.ResolveDeviceChannel(device.MAC, device.Channel)
	if ch == 0 {
		ch = m.opts.Channel
	}
	usedChannel, err := bt.BindRFCOMMWithProbe(m.opts.RFCOMMPath, device.MAC, ch, nil)
	if err != nil {
		return err
	}
	device.Channel = usedChannel
	if err := bt.RememberDeviceMAC(m.opts.RFCOMMPath, device.MAC); err != nil {
		return fmt.Errorf("save device mapping: %w", err)
	}
	return nil
}

func (m *Manager) releaseTransportNode() error {
	return bt.ReleaseRFCOMMDevice(m.opts.RFCOMMPath)
}
