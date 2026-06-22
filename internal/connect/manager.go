package connect

import (
	"context"
	"fmt"
	"strings"

	"tws_manager/internal/bt"
	"tws_manager/internal/security"
	"tws_manager/internal/session"
)

// Options holds RFCOMM connection parameters for UI-driven connect flows.
type Options struct {
	RFCOMMPath string
	Channel    int
}

// Manager provides a GUI-friendly facade over bt discovery/bind and session connect.
type Manager struct {
	sess *session.Session
	opts Options
}

// New creates a connect manager for the given session and RFCOMM options.
func New(sess *session.Session, opts Options) *Manager {
	return &Manager{sess: sess, opts: opts}
}

// Session returns the underlying session (read-only use from UI).
func (m *Manager) Session() *session.Session {
	return m.sess
}

// Options returns a copy of connection options.
func (m *Manager) Options() Options {
	return m.opts
}

// Discover lists candidate compatible Bluetooth devices.
func (m *Manager) Discover(ctx context.Context) ([]bt.Device, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return bt.Discover()
}

// DeviceFromAddress builds a device from an explicit MAC (e.g. --addr).
func DeviceFromAddress(address string, channel int) (bt.Device, error) {
	mac, err := security.NormalizeMAC(address)
	if err != nil {
		return bt.Device{}, err
	}
	return bt.Device{MAC: mac, Name: mac, Channel: bt.ResolveDeviceChannel(mac, channel)}, nil
}

// DeviceForExistingRFCOMM returns a minimal device when the RFCOMM node already exists.
func (m *Manager) DeviceForExistingRFCOMM(address string) bt.Device {
	if address == "" {
		if mac, ok := bt.LookupDeviceMAC(m.opts.RFCOMMPath); ok {
			address = mac
		}
	}
	dev := bt.Device{Channel: m.opts.Channel}
	if address != "" {
		dev.MAC = address
		dev.Name = address
		dev.Channel = bt.ResolveDeviceChannel(address, m.opts.Channel)
	} else {
		dev.Name = m.opts.RFCOMMPath
	}
	return dev
}

// Connect opens the RFCOMM session for device via session.Connect.
func (m *Manager) Connect(ctx context.Context, device bt.Device) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	ch := bt.ResolveDeviceChannel(device.MAC, device.Channel)
	if ch == 0 {
		ch = m.opts.Channel
	}
	return m.sess.Connect(device, m.opts.RFCOMMPath, ch)
}

// Disconnect closes the active session link.
func (m *Manager) Disconnect() error {
	return m.sess.Close()
}

// SwitchTo connects to a specific device, rebinding the RFCOMM node if it is
// currently bound to a different peer. Used when selecting among 2+ devices.
func (m *Manager) SwitchTo(ctx context.Context, device bt.Device) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if device.MAC == "" {
		return fmt.Errorf("device MAC is required to connect")
	}
	// If already linked to this device, keep it (idempotent).
	if snap := m.sess.Snapshot(); snap.Connected &&
		strings.EqualFold(snap.Device.MAC, device.MAC) {
		return nil
	}
	_ = m.sess.Close()
	_ = m.releaseTransportNode()
	if err := m.Bind(ctx, device); err != nil {
		return err
	}
	return m.Connect(ctx, device)
}
