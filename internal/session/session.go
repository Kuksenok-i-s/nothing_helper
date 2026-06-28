package session

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"tws_manager/internal/bt"
	"tws_manager/internal/security"
	"tws_manager/internal/spp"
	"tws_manager/internal/trace"
)

type EventKind string

const (
	EventPacketRX     EventKind = "packet_rx"
	EventPacketTX     EventKind = "packet_tx"
	EventBattery      EventKind = "battery"
	EventConnected    EventKind = "connected"
	EventDisconnected EventKind = "disconnected"
	EventError        EventKind = "error"
	EventModel        EventKind = "model"
	EventProgress     EventKind = "progress"
)

type Meta struct {
	Source      string
	Trigger     string
	UserComment string
}

type Event struct {
	Kind    EventKind
	Device  bt.Device
	Raw     []byte
	Packet  spp.Packet
	Parsed  spp.ParsedPacket
	Trace   trace.Event
	Error   error
	Source  string
	Trigger string
}

type Snapshot struct {
	Device    bt.Device
	Model     spp.ModelInfo
	Batteries map[string]spp.Battery
	DualList  []spp.DualDevice
	Connected bool
	LastError string
	// Config holds the latest decoded device configuration values keyed by
	// category: "anc", "lag" (low latency), "dual", "eq", "spatial".
	Config map[string]string
}

const (
	handshakeDelay     = 500 * time.Millisecond
	handshakeStep      = 700 * time.Millisecond
	probeIdentityDelay = 2 * time.Second
	probeBatteryDelay  = 3 * time.Second
	probeConfigStep    = 400 * time.Millisecond
)

type pendingTX struct {
	command string
	trigger string
}

type Session struct {
	mu             sync.Mutex
	connectMu      sync.Mutex
	txMu           sync.Mutex // serializes transport writes (IOBluetooth writeSync is not re-entrant)
	transport      bt.Transport
	device         bt.Device
	model          spp.ModelInfo
	batteries      map[string]spp.Battery
	logger         *trace.Logger
	events         chan Event
	subscribers    []chan Event
	lastTX         *trace.Event
	pending        map[byte]pendingTX
	dualList       []spp.DualDevice
	config         map[string]string
	captureDir     string
	rawCapture     *os.File
	allowUnsafe    bool
	probeEnabled   bool
	manualModel    bool
	batteryPolling bool
}

func New(logger *trace.Logger, allowUnsafe, probeEnabled bool) *Session {
	return &Session{
		model:        spp.DefaultModel(),
		logger:       logger,
		events:       make(chan Event, 256),
		batteries:    map[string]spp.Battery{},
		config:       map[string]string{},
		pending:      map[byte]pendingTX{},
		allowUnsafe:  allowUnsafe,
		probeEnabled: probeEnabled,
	}
}

// SetCaptureDir enables saving the raw incoming RFCOMM byte stream to a file
// (one per connection) under dir. Empty disables raw stream capture.
func (s *Session) SetCaptureDir(dir string) {
	s.mu.Lock()
	s.captureDir = dir
	s.mu.Unlock()
}

func (s *Session) Events() <-chan Event { return s.events }

func (s *Session) Subscribe() <-chan Event {
	ch := make(chan Event, 256)
	s.mu.Lock()
	s.subscribers = append(s.subscribers, ch)
	s.mu.Unlock()
	return ch
}

func (s *Session) Snapshot() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	batteries := make(map[string]spp.Battery, len(s.batteries))
	for k, v := range s.batteries {
		batteries[k] = v
	}
	dualList := append([]spp.DualDevice(nil), s.dualList...)
	config := make(map[string]string, len(s.config))
	for k, v := range s.config {
		config[k] = v
	}
	return Snapshot{Device: s.device, Model: s.model, Batteries: batteries, DualList: dualList, Connected: s.transport != nil, Config: config}
}

func (s *Session) SetModel(model spp.ModelInfo) {
	s.setModel(model, "tui", "manual model select")
}

func (s *Session) setModel(model spp.ModelInfo, source, trigger string) {
	s.mu.Lock()
	s.model = model
	s.manualModel = true
	dev := s.device
	logger := s.logger
	s.mu.Unlock()
	var tr trace.Event
	if logger != nil {
		tr = trace.Event{
			Direction:     "event",
			Source:        source,
			Trigger:       trigger,
			DeviceMAC:     dev.MAC,
			DeviceName:    dev.Name,
			ModelCodename: model.Codename,
			Summary:       fmt.Sprintf("model=%s", model.Codename),
		}
		logger.LogEvent(tr)
	}
	s.publish(Event{Kind: EventModel, Device: dev, Trace: tr, Source: source, Trigger: trigger})
}

func (s *Session) autoDetectModelLocked(device bt.Device) *trace.Event {
	if s.model.Codename != "" {
		return nil
	}
	model, source, ok := spp.ResolveModelFromBluetooth(device.Name, device.Alias, device.Info)
	if !ok {
		return nil
	}
	s.model = model
	return &trace.Event{
		Direction:     "event",
		Source:        "model",
		Trigger:       "auto-detected from bluetooth info: " + source,
		DeviceMAC:     device.MAC,
		DeviceName:    device.Name,
		ModelCodename: model.Codename,
		Summary:       fmt.Sprintf("model=%s", model.Codename),
	}
}

func (s *Session) autoDetectModelFromPacketLocked(device bt.Device, pkt spp.Packet, parsed spp.ParsedPacket) *trace.Event {
	if s.model.Codename != "" {
		return nil
	}
	model, source, ok := spp.ResolveModelFromBluetooth(
		device.Name,
		device.Alias,
		device.Info,
		parsed.Text,
		parsed.Summary,
		string(pkt.Payload),
		hex.EncodeToString(pkt.Payload),
	)
	if !ok {
		return nil
	}
	s.model = model
	return &trace.Event{
		Direction:     "event",
		Source:        "model",
		Trigger:       "auto-detected from device packet: " + source,
		DeviceMAC:     device.MAC,
		DeviceName:    device.Name,
		ModelCodename: model.Codename,
		Summary:       fmt.Sprintf("model=%s", model.Codename),
	}
}

func (s *Session) Connect(device bt.Device, transportRef string, channel int) error {
	if device.MAC != "" {
		if mac, err := security.NormalizeMAC(device.MAC); err != nil {
			return err
		} else {
			device.MAC = mac
		}
	}

	s.connectMu.Lock()
	defer s.connectMu.Unlock()

	// Idempotency guard: a second connect to the same device while a live link
	// exists would reopen transport and churn the RFCOMM session (duplicate
	// read loops and probes), which can wedge the earbuds. Skip it.
	s.mu.Lock()
	alreadyConnected := s.transport != nil && device.MAC != "" && s.device.MAC == device.MAC
	currentName := s.device.Name
	s.mu.Unlock()
	if alreadyConnected {
		name := currentName
		if name == "" {
			name = device.MAC
		}
		s.publish(Event{Kind: EventProgress, Device: device, Source: "connect", Trigger: fmt.Sprintf("already connected to %s", name)})
		return nil
	}

	if transportRef != "" {
		if _, err := security.ValidateTransportRef(transportRef); err != nil {
			return err
		}
	}
	if err := security.ValidateChannel(channel); err != nil {
		return err
	}

	if device.MAC == "" {
		if mac, ok := bt.LookupDeviceMAC(transportRef); ok {
			device.MAC = mac
			if device.Name == "" || device.Name == transportRef {
				device.Name = mac
			}
		}
	}

	if device.MAC != "" && (device.Info == "" || device.Name == "" || strings.EqualFold(device.Name, device.MAC)) {
		device = bt.EnrichDeviceInfo(device)
	}

	progress := func(step string) {
		s.publish(Event{Kind: EventProgress, Device: device, Source: "connect", Trigger: step})
	}

	channel = bt.ResolveDeviceChannel(device.MAC, channel)
	label := transportRef
	if label == "" && device.MAC != "" {
		label = device.MAC
	}
	s.publish(Event{Kind: EventProgress, Device: device, Source: "connect", Trigger: fmt.Sprintf("opening %s (channel %d)", label, channel)})
	transport, usedChannel, err := bt.OpenTransport(transportRef, device.MAC, channel, progress)
	if err != nil {
		s.publish(Event{Kind: EventError, Device: device, Error: err})
		return err
	}
	s.publish(Event{Kind: EventProgress, Device: device, Source: "connect", Trigger: fmt.Sprintf("opened %s on channel %d", transport.String(), usedChannel)})
	var modelEvent *trace.Event
	s.mu.Lock()
	if s.transport != nil {
		_ = s.transport.Close()
	}
	s.transport = transport
	device.Channel = usedChannel
	s.device = device
	s.clearLiveStateLocked()
	modelEvent = s.autoDetectModelLocked(device)
	if s.rawCapture != nil {
		_ = s.rawCapture.Close()
		s.rawCapture = nil
	}
	rawPath := s.openRawCaptureLocked()
	s.mu.Unlock()
	if modelEvent != nil {
		if s.logger != nil {
			s.logger.LogEvent(*modelEvent)
		}
		s.publish(Event{Kind: EventModel, Device: device, Trace: *modelEvent, Source: "model", Trigger: modelEvent.Trigger})
	}
	spp.ResetFSN()
	if rawPath != "" {
		s.publish(Event{Kind: EventProgress, Device: device, Source: "connect", Trigger: "raw stream -> " + rawPath})
	}
	if device.MAC != "" {
		_ = bt.RememberDeviceMAC(transportRef, device.MAC)
		_ = bt.RememberDeviceChannel(device.MAC, usedChannel)
	}
	s.publish(Event{Kind: EventConnected, Device: device, Source: "tui", Trigger: "connect"})
	s.publish(Event{Kind: EventProgress, Device: device, Source: "connect", Trigger: "starting read loop"})
	go s.readLoop(transport)
	if s.probeEnabled {
		go s.initialProbe(device)
	}
	return nil
}

func (s *Session) initialProbe(device bt.Device) {
	// Protocol handshake: activate the protocol via GET_PROTOCOL_VERSION
	// (0xC001), then query host version (0xC042). Both are read-only queries;
	// this wakes the device so it answers later GETs.
	time.Sleep(handshakeDelay)
	if !s.isCurrentDevice(device) {
		return
	}
	s.publish(Event{Kind: EventProgress, Device: device, Source: "handshake", Trigger: "activate protocol"})
	_ = s.SendCommand(spp.CmdGetProtocolVersion, Meta{Source: "handshake", Trigger: "protocol version"})

	time.Sleep(handshakeStep)
	if !s.isCurrentDevice(device) {
		return
	}
	s.publish(Event{Kind: EventProgress, Device: device, Source: "handshake", Trigger: "device version"})
	_ = s.SendCommand(spp.CmdGetFirmwareVersion, Meta{Source: "handshake", Trigger: "device version"})

	time.Sleep(probeIdentityDelay)
	if !s.isCurrentDevice(device) {
		return
	}
	s.publish(Event{Kind: EventProgress, Device: device, Source: "probe_identity", Trigger: "sending identity probe"})
	_ = s.SendCommand(spp.CmdGetIdentity, Meta{Source: "probe_identity", Trigger: "identity probe"})

	time.Sleep(probeBatteryDelay)
	if !s.isCurrentDevice(device) {
		return
	}
	s.publish(Event{Kind: EventProgress, Device: device, Source: "auto_poll", Trigger: "sending initial battery query"})
	_ = s.SendCommand(spp.CmdGetBattery, Meta{Source: "auto_poll", Trigger: "initial battery"})

	s.probeConfig(device)
}

// probeConfig queries the feature configuration (ANC, low latency, dual) right
// after connect so the UI shows the full device state without manual GETs.
func (s *Session) probeConfig(device bt.Device) {
	model := s.Snapshot().Model
	type probe struct {
		feature string
		cmd     uint16
		payload []byte
		trigger string
	}
	probes := []probe{
		{feature: "anc", cmd: spp.CmdGetNoiseReduction, trigger: "anc status"},
		{feature: "lag", cmd: spp.CmdGetLagMode, trigger: "low latency status"},
		{feature: "dual", cmd: spp.CmdGetDualEnable, trigger: "dual enable"},
		{feature: "dual", cmd: spp.CmdGetDualDeviceList, payload: []byte{0}, trigger: "dual device list"},
	}
	for _, p := range probes {
		time.Sleep(probeConfigStep)
		if !s.isCurrentDevice(device) {
			return
		}
		if !spp.ModelSupportsFeature(model, p.feature) {
			continue
		}
		s.publish(Event{Kind: EventProgress, Device: device, Source: "probe_config", Trigger: p.trigger})
		_ = s.Send(spp.Packet{Cmd: p.cmd, Payload: p.payload}, Meta{Source: "probe_config", Trigger: p.trigger})
	}
}

// RunQueryScan sends safe GET commands in [start,end] with delay between each packet.
func (s *Session) RunQueryScan(ctx context.Context, start, end uint16, delay time.Duration) error {
	if err := spp.ValidateScanRange(start, end, delay); err != nil {
		return err
	}
	dev := s.Snapshot().Device
	s.publish(Event{Kind: EventProgress, Device: dev, Source: "scan", Trigger: fmt.Sprintf("scan %04x..%04x every %s", start, end, delay)})
	for cmd := start; cmd <= end; cmd++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !s.isCurrentDevice(dev) {
			return fmt.Errorf("disconnected during scan")
		}
		trigger := fmt.Sprintf("scan %04x", cmd)
		s.publish(Event{Kind: EventProgress, Device: dev, Source: "scan", Trigger: trigger})
		if err := s.SendCommand(cmd, Meta{Source: "scan", Trigger: trigger}); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
	return nil
}

func (s *Session) isCurrentDevice(device bt.Device) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.transport != nil && s.device.MAC == device.MAC
}

func (s *Session) Close() error {
	s.mu.Lock()
	transport := s.transport
	s.mu.Unlock()
	if transport == nil {
		return nil
	}
	_, closeErr := s.finalizeDisconnect(transport, "shutdown", "RFCOMM closed", nil)
	return closeErr
}

// finalizeDisconnect tears down the active RFCOMM link and live session state,
// then publishes EventDisconnected. When f is non-nil it must match s.f unless
// another goroutine already finalized the link (stale read loop).
func (s *Session) finalizeDisconnect(transport bt.Transport, source, trigger string, err error) (bt.Device, error) {
	s.mu.Lock()
	if transport != nil && s.transport != transport {
		s.mu.Unlock()
		return bt.Device{}, nil
	}
	wasConnected := s.transport != nil
	dev := s.device
	var closeErr error
	if s.transport != nil {
		closeErr = s.transport.Close()
		s.transport = nil
	}
	if s.rawCapture != nil {
		_ = s.rawCapture.Close()
		s.rawCapture = nil
	}
	if wasConnected {
		s.clearLiveStateLocked()
	}
	s.mu.Unlock()
	if wasConnected {
		disconnectErr := err
		if disconnectErr == nil {
			disconnectErr = closeErr
		}
		s.publish(Event{Kind: EventDisconnected, Device: dev, Source: source, Trigger: trigger, Error: disconnectErr})
	}
	return dev, closeErr
}

func (s *Session) SendCommand(cmd uint16, meta Meta) error {
	return s.Send(spp.Packet{Cmd: cmd}, meta)
}

func (s *Session) authorizeCommand(cmd uint16) error {
	info := spp.CommandInfoFor(cmd)
	if info.Unsafe {
		return fmt.Errorf("command %s is blocked for device safety", spp.CommandLabel(cmd))
	}
	s.mu.Lock()
	allowUnsafe := s.allowUnsafe
	s.mu.Unlock()
	if info.Kind == "set" && !info.Safe && !allowUnsafe {
		return fmt.Errorf("command %s requires --unsafe", spp.CommandLabel(cmd))
	}
	return nil
}

func (s *Session) Send(pkt spp.Packet, meta Meta) error {
	if err := s.authorizeCommand(pkt.Cmd); err != nil {
		s.publish(Event{Kind: EventError, Device: s.Snapshot().Device, Error: err, Source: meta.Source, Trigger: meta.Trigger})
		return err
	}
	pkt.FSN = spp.NextFSN()
	pkt.FixedFSN = true
	raw := pkt.MarshalBinary()
	s.mu.Lock()
	transport := s.transport
	if transport == nil {
		s.mu.Unlock()
		return fmt.Errorf("not connected")
	}
	dev := s.device
	model := s.model
	fsn := pkt.FSN
	s.pending[fsn] = pendingTX{command: fmt.Sprintf("%04x", pkt.Cmd), trigger: meta.Trigger}
	s.mu.Unlock()

	s.txMu.Lock()
	_, err := transport.Write(raw)
	s.txMu.Unlock()

	if err != nil {
		s.mu.Lock()
		delete(s.pending, fsn)
		s.mu.Unlock()
		s.publish(Event{Kind: EventError, Device: dev, Error: err, Source: meta.Source, Trigger: meta.Trigger})
		if s.isCurrentTransport(transport) {
			_, _ = s.finalizeDisconnect(transport, meta.Source, "write error", err)
		}
		return err
	}

	s.mu.Lock()
	if s.transport != transport {
		delete(s.pending, fsn)
		s.mu.Unlock()
		return fmt.Errorf("not connected")
	}
	ctx := s.traceContext(meta, dev, model, "")
	tr := trace.Event{}
	if s.logger != nil {
		tr = s.logger.LogTX(raw, pkt, ctx)
	}
	s.lastTX = &tr
	s.mu.Unlock()
	s.publish(Event{Kind: EventPacketTX, Device: dev, Raw: raw, Packet: pkt, Trace: tr, Source: meta.Source, Trigger: meta.Trigger})
	return nil
}

func (s *Session) FeaturePacket(fields []string) (spp.Packet, []string, error) {
	s.mu.Lock()
	model := s.model
	allowUnsafe := s.allowUnsafe
	s.mu.Unlock()
	return spp.FeatureCommandPacket(fields, allowUnsafe, model)
}

func (s *Session) StartBatteryPolling(ctx context.Context, every time.Duration) {
	if every <= 0 {
		return
	}
	if every < 30*time.Second {
		every = 30 * time.Second
	}
	s.mu.Lock()
	if s.batteryPolling {
		s.mu.Unlock()
		return
	}
	s.batteryPolling = true
	s.mu.Unlock()
	go func() {
		defer func() {
			s.mu.Lock()
			s.batteryPolling = false
			s.mu.Unlock()
		}()
		ticker := time.NewTicker(every)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if !shouldSendBatteryPoll(s.Snapshot()) {
					continue
				}
				_ = s.SendCommand(spp.CmdGetBattery, Meta{Source: "auto_poll", Trigger: "battery refresh"})
			}
		}
	}()
}

func shouldSendBatteryPoll(snap Snapshot) bool {
	return snap.Connected
}

// openRawCaptureLocked opens a per-connection raw stream file. Caller holds s.mu.
func (s *Session) openRawCaptureLocked() string {
	if s.captureDir == "" {
		return ""
	}
	if err := os.MkdirAll(s.captureDir, 0o700); err != nil {
		return ""
	}
	path := filepath.Join(s.captureDir, "stream_"+time.Now().Format("2006-01-02_15-04-05")+".bin")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return ""
	}
	s.rawCapture = f
	return path
}

func (s *Session) readLoop(transport bt.Transport) {
	s.publish(Event{Kind: EventProgress, Device: s.Snapshot().Device, Source: "read_loop", Trigger: "waiting for packets"})
	s.mu.Lock()
	rawCapture := s.rawCapture
	s.mu.Unlock()
	var reader io.Reader = transport
	if rawCapture != nil {
		reader = io.TeeReader(transport, rawCapture)
	}
	for {
		raw, err := spp.ReadPacket(reader)
		if err != nil {
			if !s.isCurrentTransport(transport) {
				return
			}
			if err == io.EOF {
				_, _ = s.finalizeDisconnect(transport, "read_loop", "RFCOMM closed", err)
				return
			}
			_, _ = s.finalizeDisconnect(transport, "read_loop", "read error", err)
			return
		}
		s.handleRaw(raw)
	}
}

func (s *Session) isCurrentTransport(transport bt.Transport) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.transport == transport
}

func (s *Session) handleRaw(raw []byte) {
	s.mu.Lock()
	dev := s.device
	model := s.model
	lastTX := s.lastTX
	s.mu.Unlock()
	pkt, err := spp.DecodePacket(raw)
	if err != nil {
		related := ""
		if lastTX != nil && lastTX.Command != "" {
			related = lastTX.Command
		}
		var tr trace.Event
		if s.logger != nil {
			tr = s.logger.LogRX(raw, spp.Packet{}, spp.ParsedPacket{}, err, s.traceContext(Meta{Source: "device"}, dev, model, related))
		}
		s.publish(Event{Kind: EventError, Device: dev, Raw: raw, Trace: tr, Error: err})
		return
	}
	related := s.matchRequest(pkt)
	parsed := spp.ParsePacket(pkt, model)
	var modelEvent *trace.Event
	s.mu.Lock()
	modelEvent = s.autoDetectModelFromPacketLocked(dev, pkt, parsed)
	if modelEvent != nil {
		model = s.model
	}
	s.mu.Unlock()
	if modelEvent != nil {
		if s.logger != nil {
			s.logger.LogEvent(*modelEvent)
		}
		s.publish(Event{Kind: EventModel, Device: dev, Trace: *modelEvent, Source: "model", Trigger: modelEvent.Trigger})
	}
	s.recordConfig(parsed)
	if parsed.DualList != nil {
		s.mu.Lock()
		s.dualList = append([]spp.DualDevice(nil), parsed.DualList.Devices...)
		s.mu.Unlock()
	}
	if parsed.Kind == "dual_response" && len(pkt.Payload) > 0 && pkt.Payload[0] == 1 {
		go func() {
			time.Sleep(200 * time.Millisecond)
			_ = s.SendCommand(spp.CmdGetSupportedFeature, Meta{Source: "dual", Trigger: "dual supported feature"})
		}()
	}
	if parsed.Kind == "supported_features" && spp.SupportedFeatureDualList(pkt.Payload) {
		go func() {
			time.Sleep(200 * time.Millisecond)
			_ = s.Send(spp.Packet{Cmd: spp.CmdGetDualDeviceList, Payload: []byte{0}}, Meta{Source: "dual", Trigger: "dual device list"})
		}()
	}
	if parsed.Kind == "dual_connect_changed" {
		go func() {
			time.Sleep(200 * time.Millisecond)
			_ = s.Send(spp.Packet{Cmd: spp.CmdGetDualDeviceList, Payload: []byte{0}}, Meta{Source: "dual", Trigger: "dual device list refresh"})
		}()
	}
	var tr trace.Event
	if s.logger != nil {
		tr = s.logger.LogRX(raw, pkt, parsed, nil, s.traceContext(Meta{Source: "device"}, dev, model, related))
	}
	kind := EventPacketRX
	if len(parsed.Batteries) > 0 {
		kind = EventBattery
		s.mu.Lock()
		s.batteries = mergeBatteries(s.batteries, parsed.Batteries)
		parsed.Batteries = cloneBatteries(s.batteries)
		s.mu.Unlock()
	}
	s.publish(Event{Kind: kind, Device: dev, Raw: raw, Packet: pkt, Parsed: parsed, Trace: tr, Source: "device"})
}

func (s *Session) traceContext(meta Meta, dev bt.Device, model spp.ModelInfo, related string) trace.Context {
	return trace.Context{Source: meta.Source, Trigger: meta.Trigger, Device: trace.DeviceInfo{MAC: dev.MAC, Name: dev.Name}, Model: model, UserComment: meta.UserComment, RelatedTXCommand: related}
}
