package session

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"tws_manager/internal/bt"
	"tws_manager/internal/spp"
)

func TestSendBlocksUnsafe(t *testing.T) {
	s := New(nil, false, false)
	err := s.SendCommand(0xF03D, Meta{Source: "test"})
	if err == nil {
		t.Fatal("expected error for restore_factory_setting")
	}
	if !strings.Contains(err.Error(), "blocked") {
		t.Fatalf("error = %q, want device safety block", err.Error())
	}
}

func TestSendBlocksSetWithoutUnsafe(t *testing.T) {
	s := New(nil, false, false)
	err := s.SendCommand(0xF041, Meta{Source: "test"})
	if err == nil {
		t.Fatal("expected error for set_custom_eq without --unsafe")
	}
	if !strings.Contains(err.Error(), "--unsafe") {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestAuthorizeAllowsUISafeSetWithoutUnsafe(t *testing.T) {
	s := New(nil, false, false)
	for _, cmd := range []uint16{spp.CmdSetNoiseReduction, spp.CmdSetEQMode, spp.CmdSetLagMode, spp.CmdSetSpatialAudio, spp.CmdSetDualEnable, spp.CmdSetConnectDevice} {
		if err := s.authorizeCommand(cmd); err != nil {
			t.Fatalf("authorize UI-safe command %s without --unsafe: %v", spp.CommandLabel(cmd), err)
		}
	}
}

func TestMatchRequestByFSN(t *testing.T) {
	s := New(nil, false, false)
	s.pending[7] = pendingTX{command: "c042", trigger: "Info: firmware"}

	// Response echoes FSN 7 -> pairs with the firmware request.
	got := s.matchRequest(spp.Packet{Cmd: spp.CmdRspFirmware, FSN: 7})
	if got != "c042" {
		t.Fatalf("matchRequest by FSN = %q, want c042", got)
	}
	// Pending entry must be consumed.
	if _, ok := s.pending[7]; ok {
		t.Fatal("pending entry for FSN 7 was not consumed")
	}
}

func TestMatchRequestFallbackFromResponseCmd(t *testing.T) {
	s := New(nil, false, false)
	// No pending entry: derive request 0xC007 from response 0x4007.
	got := s.matchRequest(spp.Packet{Cmd: spp.CmdRspBattery, FSN: 99})
	if got != "c007" {
		t.Fatalf("matchRequest fallback = %q, want c007", got)
	}
}

func TestRawStreamCaptureTeesIncomingBytes(t *testing.T) {
	dir := t.TempDir()
	s := New(nil, false, false)
	s.SetCaptureDir(dir)

	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	s.mu.Lock()
	s.transport = bt.NewTestTransport(pr, "", 15, "/dev/rfcomm0")
	rawPath := s.openRawCaptureLocked()
	s.mu.Unlock()
	if rawPath == "" {
		t.Fatal("openRawCaptureLocked returned empty path")
	}
	go s.readLoop(s.transport)

	frame := spp.BuildFrame(spp.ControlCRC|spp.ControlMultiFrame, spp.CmdRspBattery, 1, []byte{0x02, 0x46})
	if _, err := pw.Write(frame); err != nil {
		t.Fatal(err)
	}
	time.Sleep(150 * time.Millisecond)
	_ = pw.Close()
	_ = s.Close()

	data, err := os.ReadFile(rawPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, frame) {
		t.Fatalf("raw capture %x does not contain frame %x", data, frame)
	}
}

func TestAutoDetectModelLockedFromDevice(t *testing.T) {
	s := New(nil, false, false)
	dev := bt.Device{
		MAC:   "AA:BB:CC:DD:EE:FF",
		Name:  "2C:BE:EE:4A:EC:9E",
		Alias: "Nothing Ear (a)",
	}

	s.mu.Lock()
	event := s.autoDetectModelLocked(dev)
	model := s.model
	s.mu.Unlock()

	if event == nil {
		t.Fatal("autoDetectModelLocked returned nil event")
	}
	if model.Codename != "EarColor" {
		t.Fatalf("model = %s, want EarColor", model.Codename)
	}
	if event.ModelCodename != "EarColor" {
		t.Fatalf("event model = %s, want EarColor", event.ModelCodename)
	}
}

func TestAutoDetectModelLockedDoesNotOverrideManualModel(t *testing.T) {
	s := New(nil, false, false)
	manual, ok := spp.ResolveModelInfo("EarThree")
	if !ok {
		t.Fatal("EarThree model not found")
	}
	s.SetModel(manual)

	s.mu.Lock()
	event := s.autoDetectModelLocked(bt.Device{Alias: "Nothing Ear (a)"})
	model := s.model
	s.mu.Unlock()

	if event != nil {
		t.Fatalf("autoDetectModelLocked returned event for manual model: %+v", event)
	}
	if model.Codename != "EarThree" {
		t.Fatalf("model = %s, want EarThree", model.Codename)
	}
}

func TestAutoDetectModelFromPacketLocked(t *testing.T) {
	s := New(nil, false, false)
	pkt := spp.Packet{Cmd: spp.CmdRspIdentity, Payload: []byte("identity product=Ear (3)")}
	parsed := spp.ParsedPacket{Kind: "identity_response", Text: string(pkt.Payload), Summary: "identity_response: product=Ear (3)"}

	s.mu.Lock()
	event := s.autoDetectModelFromPacketLocked(bt.Device{MAC: "AA:BB:CC:DD:EE:FF"}, pkt, parsed)
	model := s.model
	s.mu.Unlock()

	if event == nil {
		t.Fatal("autoDetectModelFromPacketLocked returned nil event")
	}
	if model.Codename != "EarThree" {
		t.Fatalf("model = %s, want EarThree", model.Codename)
	}
	if !strings.Contains(event.Trigger, "device packet") {
		t.Fatalf("trigger = %q", event.Trigger)
	}
}

func TestConnectIdempotentWhenAlreadyConnected(t *testing.T) {
	s := New(nil, false, false)
	// Simulate a live link to a device.
	f, err := os.CreateTemp(t.TempDir(), "rfcomm")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	s.mu.Lock()
	transport := bt.NewTestTransport(f, "2C:BE:EE:4A:EC:9E", 15, "/dev/rfcomm0")
	s.transport = transport
	s.device = bt.Device{MAC: "2C:BE:EE:4A:EC:9E", Name: "Nothing Ear (3)"}
	s.mu.Unlock()

	// A second connect to the same MAC must be a no-op (no reopen, no error).
	err = s.Connect(bt.Device{MAC: "2c:be:ee:4a:ec:9e"}, "2C:BE:EE:4A:EC:9E", 15)
	if err != nil {
		t.Fatalf("idempotent connect returned error: %v", err)
	}
	s.mu.Lock()
	stillSame := s.transport == transport
	s.mu.Unlock()
	if !stillSame {
		t.Fatal("session transport was replaced by redundant connect")
	}
}

func TestHandleRawStoresDualDeviceList(t *testing.T) {
	s := New(nil, false, false)
	payload := []byte{
		0x01, 0x00, 0x01,
		0x11, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff,
		0x02, 'P', 'C',
	}
	raw := spp.BuildFrame(spp.ControlCRC|spp.ControlMultiFrame, spp.CmdRspDualDeviceList, 1, payload)

	s.handleRaw(raw)

	snap := s.Snapshot()
	if len(snap.DualList) != 1 {
		t.Fatalf("DualList len = %d, want 1", len(snap.DualList))
	}
	dev := snap.DualList[0]
	if dev.MAC != "AA:BB:CC:DD:EE:FF" || dev.Name != "PC" || !dev.Connected || !dev.Owner {
		t.Fatalf("dual device = %+v", dev)
	}
}

func TestFinalizeDisconnectOnEOF(t *testing.T) {
	s := New(nil, false, false)
	events := s.Subscribe()

	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	s.mu.Lock()
	s.transport = bt.NewTestTransport(pr, "AA:BB:CC:DD:EE:FF", 15, "/dev/rfcomm0")
	s.device = bt.Device{MAC: "AA:BB:CC:DD:EE:FF", Name: "Nothing Ear"}
	s.batteries = map[string]spp.Battery{"left": {Percent: 80}}
	s.config = map[string]string{"anc": "on"}
	s.dualList = []spp.DualDevice{{MAC: "11:22:33:44:55:66", Connected: true}}
	s.model, _ = spp.ResolveModelInfo("EarThree")
	s.mu.Unlock()

	go s.readLoop(s.transport)
	_ = pw.Close()

	deadline := time.After(2 * time.Second)
	var gotDisconnect bool
	for !gotDisconnect {
		select {
		case event := <-events:
			if event.Kind == EventDisconnected {
				gotDisconnect = true
			}
		case <-deadline:
			t.Fatal("timeout waiting for disconnect event")
		}
	}

	snap := s.Snapshot()
	if snap.Connected {
		t.Fatal("snapshot still connected after EOF")
	}
	if len(snap.Batteries) != 0 {
		t.Fatalf("batteries = %+v, want empty", snap.Batteries)
	}
	if len(snap.Config) != 0 {
		t.Fatalf("config = %+v, want empty", snap.Config)
	}
	if len(snap.DualList) != 0 {
		t.Fatalf("dualList = %+v, want empty", snap.DualList)
	}
	if snap.Model.Codename == "EarThree" {
		t.Fatalf("model = %+v, want default model reset", snap.Model)
	}
	if snap.Device.MAC != "AA:BB:CC:DD:EE:FF" {
		t.Fatalf("device MAC = %q, want preserved for reconnect hint", snap.Device.MAC)
	}
}

func TestHandleRawMergesPartialBatteryEvents(t *testing.T) {
	s := New(nil, false, false)
	events := s.Subscribe()

	full := spp.BuildFrame(spp.ControlCRC|spp.ControlMultiFrame, spp.CmdRspBattery, 1, []byte{
		0x03,
		0x02, 0x46,
		0x03, 0x55,
		0x04, 0x64,
	})
	s.handleRaw(full)
	<-events

	partial := spp.BuildFrame(spp.ControlCRC|spp.ControlMultiFrame, spp.CmdBatteryChanged, 2, []byte{
		0x02,
		0x02, 0x45,
		0x03, 0x54,
	})
	s.handleRaw(partial)
	event := <-events

	snap := s.Snapshot()
	if got := snap.Batteries["case"]; got.Percent != 100 {
		t.Fatalf("snapshot case battery = %+v, want 100%%", got)
	}
	if got := snap.Batteries["left"]; got.Percent != 69 {
		t.Fatalf("snapshot left battery = %+v, want 69%%", got)
	}
	if got := event.Parsed.Batteries["case"]; got.Percent != 100 {
		t.Fatalf("event case battery = %+v, want 100%%", got)
	}
}

func TestMergeBatteriesDropsStaleStereoWhenCaseUpdated(t *testing.T) {
	current := map[string]spp.Battery{
		"stereo": {Percent: 80},
	}
	update := map[string]spp.Battery{
		"case": {Percent: 75},
	}
	merged := mergeBatteries(current, update)
	if _, ok := merged["stereo"]; ok {
		t.Fatal("stereo key should be removed when case is updated")
	}
	if got := merged["case"].Percent; got != 75 {
		t.Fatalf("case = %d, want 75", got)
	}
}

func TestSendLeavesPendingAfterSuccessfulWrite(t *testing.T) {
	spp.ResetFSN()
	s := New(nil, false, false)
	f, err := os.CreateTemp(t.TempDir(), "rfcomm")
	if err != nil {
		t.Fatal(err)
	}

	s.mu.Lock()
	s.transport = bt.NewTestTransport(f, "AA:BB:CC:DD:EE:FF", 15, "/dev/rfcomm0")
	s.device = bt.Device{MAC: "AA:BB:CC:DD:EE:FF", Name: "Nothing Ear"}
	s.mu.Unlock()

	if err := s.SendCommand(spp.CmdGetBattery, Meta{Source: "test", Trigger: "battery"}); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	s.mu.Lock()
	_, ok := s.pending[1]
	s.mu.Unlock()
	if !ok {
		t.Fatal("pending entry must remain after successful write for RX correlation")
	}
}

func TestSendRemovesPendingOnWriteError(t *testing.T) {
	spp.ResetFSN()
	s := New(nil, false, false)
	f, err := os.CreateTemp(t.TempDir(), "rfcomm")
	if err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	s.mu.Lock()
	s.transport = bt.NewTestTransport(f, "AA:BB:CC:DD:EE:FF", 15, "/dev/rfcomm0")
	s.device = bt.Device{MAC: "AA:BB:CC:DD:EE:FF", Name: "Nothing Ear"}
	s.mu.Unlock()

	if err := s.SendCommand(spp.CmdGetBattery, Meta{Source: "test", Trigger: "battery"}); err == nil {
		t.Fatal("expected write error on closed file")
	}
	s.mu.Lock()
	pendingLen := len(s.pending)
	s.mu.Unlock()
	if pendingLen != 0 {
		t.Fatalf("pending map len = %d, want 0 after write failure", pendingLen)
	}
}

func TestConnectClearsLiveStateOnDeviceSwitch(t *testing.T) {
	s := New(nil, false, false)
	f, err := os.CreateTemp(t.TempDir(), "rfcomm")
	if err != nil {
		t.Fatal(err)
	}
	s.mu.Lock()
	s.transport = bt.NewTestTransport(f, "AA:BB:CC:DD:EE:FF", 15, "/dev/rfcomm0")
	s.device = bt.Device{MAC: "AA:BB:CC:DD:EE:FF", Name: "Ear A"}
	s.batteries = map[string]spp.Battery{"left": {Percent: 80}}
	s.config = map[string]string{"anc": "on"}
	s.dualList = []spp.DualDevice{{MAC: "11:22:33:44:55:66"}}
	s.model, _ = spp.ResolveModelInfo("EarThree")
	s.mu.Unlock()

	f2, err := os.CreateTemp(t.TempDir(), "rfcomm2")
	if err != nil {
		t.Fatal(err)
	}
	s.mu.Lock()
	if s.transport != nil {
		_ = s.transport.Close()
	}
	s.transport = bt.NewTestTransport(f2, "BB:CC:DD:EE:FF:00", 15, "/dev/rfcomm1")
	s.device = bt.Device{MAC: "BB:CC:DD:EE:FF:00", Name: "Ear B"}
	s.clearLiveStateLocked()
	s.mu.Unlock()

	snap := s.Snapshot()
	if len(snap.Batteries) != 0 {
		t.Fatalf("batteries = %+v, want empty after device switch", snap.Batteries)
	}
	if len(snap.Config) != 0 {
		t.Fatalf("config = %+v, want empty after device switch", snap.Config)
	}
	if len(snap.DualList) != 0 {
		t.Fatalf("dualList = %+v, want empty after device switch", snap.DualList)
	}
	if snap.Model.Codename != "" {
		t.Fatalf("model = %+v, want default after device switch", snap.Model)
	}
}

func TestClearLiveStatePreservesManualModel(t *testing.T) {
	s := New(nil, false, false)
	manual, ok := spp.ResolveModelInfo("EarThree")
	if !ok {
		t.Fatal("EarThree model not found")
	}
	s.SetModel(manual)

	s.mu.Lock()
	s.clearLiveStateLocked()
	model := s.model
	s.mu.Unlock()

	if model.Codename != "EarThree" {
		t.Fatalf("model = %q, want manual EarThree preserved across reconnect", model.Codename)
	}
}

func TestPublishDeliversPriorityEventsWhenBufferFull(t *testing.T) {
	s := New(nil, false, false)
	events := s.Subscribe()
	for i := 0; i < 256; i++ {
		s.publish(Event{Kind: EventProgress, Trigger: "fill"})
	}

	got := make(chan EventKind, 1)
	go func() {
		for {
			ev := <-events
			if ev.Kind == EventBattery {
				got <- ev.Kind
				return
			}
		}
	}()

	s.publish(Event{Kind: EventBattery, Parsed: spp.ParsedPacket{Summary: "battery"}})

	select {
	case kind := <-got:
		if kind != EventBattery {
			t.Fatalf("kind = %q, want battery", kind)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for priority battery event")
	}
}
