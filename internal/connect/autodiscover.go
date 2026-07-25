package connect

import (
	"context"
	"errors"
	"time"

	"tws_manager/internal/audio"
	"tws_manager/internal/bt"
)

var (
	errNoCandidate           = errors.New("no compatible TWS device found")
	errWaitingForBluetooth   = errors.New("waiting for bluetooth connection")
	errWaitingForAudioOutput = errors.New("waiting for bluetooth audio output")
)

// autoTestHooks optional overrides for unit tests (nil in production).
type autoTestHooks struct {
	rfcommExists func(m *Manager) (bool, error)
	discover     func(m *Manager, ctx context.Context) ([]bt.Device, error)
	bind         func(m *Manager, ctx context.Context, dev bt.Device) error
	connect      func(m *Manager, ctx context.Context, dev bt.Device) error
}

var autoTestHooksVar *autoTestHooks

var (
	hookIsDeviceConnected    = bt.IsDeviceConnected
	hookIsDefaultAudioOutput = audio.IsDefaultOutputForMAC
	hookReleaseRFCOMMDevice  = bt.ReleaseRFCOMMDevice
)

// AutoOptions tunes the auto-discovery loop.
type AutoOptions struct {
	// Interval is the delay between connection attempts. Defaults to 5s.
	Interval time.Duration
	// OnStatus, if set, receives human-readable progress messages.
	OnStatus func(string)
}

type statusReporter struct {
	on   func(string)
	last string
}

func newStatusReporter(onStatus func(string)) *statusReporter {
	if onStatus == nil {
		onStatus = func(string) {}
	}
	return &statusReporter{on: onStatus}
}

func (r *statusReporter) report(msg string) {
	if msg == r.last {
		return
	}
	r.last = msg
	r.on(msg)
}

func (r *statusReporter) reset() {
	r.last = ""
}

func nilSafeStatus(status func(string)) func(string) {
	if status == nil {
		return func(string) {}
	}
	return status
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
	m.runAutoConnectLoop(ctx, interval, newStatusReporter(o.OnStatus))
}

func (m *Manager) runAutoConnectLoop(ctx context.Context, interval time.Duration, report *statusReporter) {
	for {
		if ctx.Err() != nil {
			return
		}
		if m.sess.Snapshot().Connected {
			report.reset()
			if !sleep(ctx, interval) {
				return
			}
			continue
		}
		m.runAutoConnectAttempt(ctx, report)
		if !sleep(ctx, interval) {
			return
		}
	}
}

func (m *Manager) runAutoConnectAttempt(ctx context.Context, report *statusReporter) {
	err := m.ConnectBest(ctx, report.report)
	if err != nil && ctx.Err() == nil {
		reportAutoConnectErr(report.report, err)
	}
}

// ConnectBest performs a single discover → bind → connect attempt to the best
// available compatible TWS device, reusing an existing RFCOMM node when present.
func (m *Manager) ConnectBest(ctx context.Context, status func(string)) error {
	status = nilSafeStatus(status)
	if err := ctx.Err(); err != nil {
		return err
	}

	exists, err := m.autoRFCOMMExists()
	if err != nil {
		return err
	}
	if exists {
		return m.connectBestExisting(ctx, status)
	}
	return m.connectBestFresh(ctx, status)
}

func (m *Manager) connectBestExisting(ctx context.Context, status func(string)) error {
	dev := m.DeviceForExistingRFCOMM("")
	if dev.MAC != "" {
		return m.connectViaExisting(ctx, dev, status)
	}
	status("auto: existing " + m.opts.RFCOMMPath + " has no saved MAC; scanning for device metadata")
	dev, ok, err := m.discoverBestDevice(ctx, status)
	if err != nil || !ok {
		return err
	}
	status("auto: rebinding existing " + m.opts.RFCOMMPath + " to " + dev.MAC)
	_ = hookReleaseRFCOMMDevice(m.opts.RFCOMMPath)
	return m.bindAndConnect(ctx, dev, status, "auto: rebinding existing "+m.opts.RFCOMMPath+" to")
}

func (m *Manager) connectBestFresh(ctx context.Context, status func(string)) error {
	dev, ok, err := m.discoverBestDevice(ctx, status)
	if err != nil || !ok {
		return err
	}
	return m.bindAndConnect(ctx, dev, status, "auto: binding")
}

func (m *Manager) connectViaExisting(ctx context.Context, dev bt.Device, status func(string)) error {
	if err := m.requireRFCOMMReady(ctx, dev.MAC); err != nil {
		if errors.Is(err, errWaitingForBluetooth) || errors.Is(err, errWaitingForAudioOutput) {
			return err
		}
		if ctx.Err() == nil {
			status("auto: readiness check failed: " + err.Error())
		}
		return err
	}
	status("auto: connecting via existing " + m.opts.RFCOMMPath)
	err := m.autoConnect(ctx, dev)
	if err != nil && ctx.Err() == nil {
		status("auto: connect failed: " + err.Error())
	}
	return err
}

func (m *Manager) bindAndConnect(ctx context.Context, dev bt.Device, status func(string), bindLabel string) error {
	status(bindLabel + " " + dev.MAC)
	if err := m.autoBind(ctx, dev); err != nil {
		return m.reportAutoStepError(status, ctx, "auto: bind failed: ", err)
	}
	status("auto: connecting " + dev.MAC)
	if err := m.autoConnect(ctx, dev); err != nil {
		return m.reportAutoStepError(status, ctx, "auto: connect failed: ", err)
	}
	return nil
}

func (m *Manager) reportAutoStepError(status func(string), ctx context.Context, prefix string, err error) error {
	if ctx.Err() == nil {
		status(prefix + err.Error())
	}
	return err
}

func (m *Manager) discoverBestDevice(ctx context.Context, status func(string)) (bt.Device, bool, error) {
	status("auto: scanning for compatible TWS devices")
	devices, err := m.autoDiscover(ctx)
	if err != nil {
		if ctx.Err() == nil {
			status("auto: discover failed: " + err.Error())
		}
		return bt.Device{}, false, err
	}
	dev, ok := BestConnectedCandidate(devices)
	if !ok {
		if len(devices) == 0 {
			return bt.Device{}, false, errNoCandidate
		}
		return bt.Device{}, false, errWaitingForBluetooth
	}
	if err := m.requireRFCOMMReady(ctx, dev.MAC); err != nil {
		return bt.Device{}, false, err
	}
	return dev, true, nil
}

func (m *Manager) autoRFCOMMExists() (bool, error) {
	if autoTestHooksVar != nil && autoTestHooksVar.rfcommExists != nil {
		return autoTestHooksVar.rfcommExists(m)
	}
	return m.RFCOMMExists()
}

func (m *Manager) autoDiscover(ctx context.Context) ([]bt.Device, error) {
	if autoTestHooksVar != nil && autoTestHooksVar.discover != nil {
		return autoTestHooksVar.discover(m, ctx)
	}
	return m.Discover(ctx)
}

func (m *Manager) autoBind(ctx context.Context, dev bt.Device) error {
	if autoTestHooksVar != nil && autoTestHooksVar.bind != nil {
		return autoTestHooksVar.bind(m, ctx, dev)
	}
	return m.Bind(ctx, dev)
}

func (m *Manager) autoConnect(ctx context.Context, dev bt.Device) error {
	if autoTestHooksVar != nil && autoTestHooksVar.connect != nil {
		return autoTestHooksVar.connect(m, ctx, dev)
	}
	return m.Connect(ctx, dev)
}

func (m *Manager) requireRFCOMMReady(ctx context.Context, mac string) error {
	connected, err := hookIsDeviceConnected(mac)
	if err != nil {
		return err
	}
	if !connected {
		return errWaitingForBluetooth
	}
	isOutput, err := hookIsDefaultAudioOutput(ctx, mac)
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
