package connect

import (
	"context"
	"errors"
	"time"

	"tws_manager/internal/audio"
	"tws_manager/internal/bt"
)

var (
	errNoCandidate          = errors.New("no compatible TWS device found")
	errWaitingForBluetooth  = errors.New("waiting for bluetooth connection")
	errWaitingForAudioOutput = errors.New("waiting for bluetooth audio output")
)

// AutoOptions tunes the auto-discovery loop.
type AutoOptions struct {
	// Interval is the delay between connection attempts. Defaults to 5s.
	Interval time.Duration
	// OnStatus, if set, receives human-readable progress messages.
	OnStatus func(string)
}

// BestCandidate picks the most likely earbuds from a discovery result.
//
// Preference order: a connected device exposing the compatible SPP profile, then
// any SPP device, then any connected device, then the first candidate.
func BestCandidate(devices []bt.Device) (bt.Device, bool) {
	var spp, connected, any *bt.Device
	for i := range devices {
		d := &devices[i]
		if d.MAC == "" {
			continue
		}
		if d.SPP && d.Connected {
			return *d, true
		}
		if d.SPP && spp == nil {
			spp = d
		}
		if d.Connected && connected == nil {
			connected = d
		}
		if any == nil {
			any = d
		}
	}
	switch {
	case spp != nil:
		return *spp, true
	case connected != nil:
		return *connected, true
	case any != nil:
		return *any, true
	}
	return bt.Device{}, false
}

// BestConnectedCandidate picks the best compatible TWS device that is currently
// connected at the Bluetooth layer. Used by auto-connect to avoid RFCOMM retries
// while earbuds are in the case or powered off.
func BestConnectedCandidate(devices []bt.Device) (bt.Device, bool) {
	var connected *bt.Device
	for i := range devices {
		d := &devices[i]
		if d.MAC == "" || !d.Connected {
			continue
		}
		if d.SPP {
			return *d, true
		}
		if connected == nil {
			connected = d
		}
	}
	if connected != nil {
		return *connected, true
	}
	return bt.Device{}, false
}

// AutoConnect repeatedly discovers and connects to the best compatible TWS device until
// the context is cancelled. While a link is up it idles; if the link drops it
// retries, providing reconnect-on-loss for tray/daemon usage.
func (m *Manager) AutoConnect(ctx context.Context, o AutoOptions) {
	interval := o.Interval
	if interval <= 0 {
		interval = 5 * time.Second
	}
	status := o.OnStatus
	if status == nil {
		status = func(string) {}
	}
	var lastStatus string
	report := func(msg string) {
		if msg == lastStatus {
			return
		}
		lastStatus = msg
		status(msg)
	}

	for {
		if ctx.Err() != nil {
			return
		}
		if m.sess.Snapshot().Connected {
			lastStatus = ""
			if !sleep(ctx, interval) {
				return
			}
			continue
		}
		err := m.ConnectBest(ctx, report)
		if err != nil && ctx.Err() == nil {
			switch {
			case errors.Is(err, errWaitingForBluetooth):
				report("auto: waiting for headphones (Bluetooth disconnected)")
			case errors.Is(err, errWaitingForAudioOutput):
				report("auto: waiting for headphones to become audio output")
			case errors.Is(err, errNoCandidate):
				report("auto: no compatible TWS device found")
			}
		}
		if !sleep(ctx, interval) {
			return
		}
	}
}

// ConnectBest performs a single discover → bind → connect attempt to the best
// available compatible TWS device, reusing an existing RFCOMM node when present.
func (m *Manager) ConnectBest(ctx context.Context, status func(string)) error {
	if status == nil {
		status = func(string) {}
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	exists, _ := m.RFCOMMExists()
	if exists {
		dev := m.DeviceForExistingRFCOMM("")
		if dev.MAC != "" {
			if err := m.requireRFCOMMReady(dev.MAC); err != nil {
				if errors.Is(err, errWaitingForBluetooth) || errors.Is(err, errWaitingForAudioOutput) {
					return err
				}
				if ctx.Err() == nil {
					status("auto: readiness check failed: " + err.Error())
				}
				return err
			}
			status("auto: connecting via existing " + m.opts.RFCOMMPath)
			err := m.Connect(ctx, dev)
			if err != nil && ctx.Err() == nil {
				status("auto: connect failed: " + err.Error())
			}
			return err
		}
		status("auto: existing " + m.opts.RFCOMMPath + " has no saved MAC; scanning for device metadata")
	}

	status("auto: scanning for compatible TWS devices")
	devices, err := m.Discover(ctx)
	if err != nil {
		if ctx.Err() == nil {
			status("auto: discover failed: " + err.Error())
		}
		return err
	}
	dev, ok := BestConnectedCandidate(devices)
	if !ok {
		return errWaitingForBluetooth
	}
	if err := m.requireRFCOMMReady(dev.MAC); err != nil {
		return err
	}

	if exists {
		status("auto: rebinding existing " + m.opts.RFCOMMPath + " to " + dev.MAC)
		_ = bt.ReleaseRFCOMMDevice(m.opts.RFCOMMPath)
		if err := m.Bind(ctx, dev); err != nil {
			if ctx.Err() == nil {
				status("auto: bind failed: " + err.Error())
			}
			return err
		}
		status("auto: connecting " + dev.MAC)
		if err := m.Connect(ctx, dev); err != nil {
			if ctx.Err() == nil {
				status("auto: connect failed: " + err.Error())
			}
			return err
		}
		return nil
	}

	status("auto: binding " + dev.MAC)
	if err := m.Bind(ctx, dev); err != nil {
		if ctx.Err() == nil {
			status("auto: bind failed: " + err.Error())
		}
		return err
	}
	status("auto: connecting " + dev.MAC)
	if err := m.Connect(ctx, dev); err != nil {
		if ctx.Err() == nil {
			status("auto: connect failed: " + err.Error())
		}
		return err
	}
	return nil
}

func (m *Manager) requireRFCOMMReady(mac string) error {
	connected, err := bt.IsDeviceConnected(mac)
	if err != nil {
		return err
	}
	if !connected {
		return errWaitingForBluetooth
	}
	isOutput, err := audio.IsDefaultOutputForMAC(mac)
	if err != nil {
		return err
	}
	if !isOutput {
		return errWaitingForAudioOutput
	}
	return nil
}

func sleep(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
