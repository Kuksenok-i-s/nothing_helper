package spp

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestParsePairsCaseBatteryCharging(t *testing.T) {
	data := ParsePairs([]byte{0x04, 0xe4})
	if data == nil {
		t.Fatal("ParsePairs returned nil")
	}
	got, ok := data["case"]
	if !ok {
		t.Fatal("case battery is missing")
	}
	if got.Percent != 100 {
		t.Fatalf("case percent = %d, want 100", got.Percent)
	}
	if !got.Charging {
		t.Fatal("case charging = false, want true")
	}
}

func TestNormalizeBatteryStereoCaseForHeadphoneModels(t *testing.T) {
	model, ok := ResolveModelInfo("2D6FDA")
	if !ok {
		t.Fatal("Elekid model not found")
	}
	data, warnings := NormalizeBatteryForModel(map[string]Battery{"stereo": {Percent: 88, Charging: true}}, model)
	got, ok := data["case"]
	if !ok {
		t.Fatal("case battery is missing")
	}
	if got.Percent != 88 || !got.Charging {
		t.Fatalf("case battery = %+v, want 88%% charging", got)
	}
	if _, ok := data["stereo"]; ok {
		t.Fatal("stereo battery should be folded into case for Elekid")
	}
	if len(warnings) == 0 || !strings.Contains(warnings[0], "Elekid") {
		t.Fatalf("warnings = %v, want Elekid mapping warning", warnings)
	}
}

func TestParseBatteryUsesModelStereoCase(t *testing.T) {
	model, ok := ResolveModelInfo("Crobat")
	if !ok {
		t.Fatal("Crobat model not found")
	}
	pkt := Packet{Cmd: CmdBattery, Payload: []byte{0x01, 0x06, 0x64}}
	parsed := ParsePacket(pkt, model)
	want := "battery_full: left: n/a | right: n/a | case: 100%"
	if parsed.Summary != want {
		t.Fatalf("summary = %q, want %q", parsed.Summary, want)
	}
	if parsed.Batteries["case"].Percent != 100 {
		t.Fatalf("case percent = %d, want 100", parsed.Batteries["case"].Percent)
	}
}

func TestFormatBatteryCaseCharging(t *testing.T) {
	got := FormatBattery(map[string]Battery{"case": {Percent: 100, Charging: true}})
	want := "left: n/a | right: n/a | case: 100% charging"
	if got != want {
		t.Fatalf("FormatBattery() = %q, want %q", got, want)
	}
}

func TestDecodeAndParseBatteryFixture(t *testing.T) {
	payload := []byte{0x03, 0x02, 0x46, 0x03, 0x55, 0x04, 0x64}
	raw := BuildFrame(ControlCRC|ControlMultiFrame, CmdBattery, 1, payload)
	pkt, err := DecodePacket(raw)
	if err != nil {
		t.Fatalf("DecodePacket() error = %v", err)
	}
	if pkt.Cmd != CmdBattery {
		t.Fatalf("cmd = %04x, want %04x", pkt.Cmd, CmdBattery)
	}
	parsed := ParsePacket(pkt, DefaultModel())
	want := "battery_full: left: 70% | right: 85% | case: 100%"
	if parsed.Summary != want {
		t.Fatalf("summary = %q, want %q", parsed.Summary, want)
	}
}

func TestParseBatteryRealDeviceFixture(t *testing.T) {
	// Real Ear (3) GET_BATTERY response payload: count=2, (left,60%) (right,55%).
	pkt := Packet{Cmd: CmdRspBattery, Payload: []byte{0x02, 0x02, 0x3c, 0x03, 0x37}}
	parsed := ParsePacket(pkt, DefaultModel())
	want := "battery_response: left: 60% | right: 55% | case: n/a"
	if parsed.Summary != want {
		t.Fatalf("summary = %q, want %q", parsed.Summary, want)
	}
}

func TestParseStatusRealDeviceFixture(t *testing.T) {
	// Real Ear (3) status: count=3, left=0x8c right=0x8c case=0x01 (lid open).
	pkt := Packet{Cmd: CmdRspStatus, Payload: []byte{0x03, 0x02, 0x8c, 0x03, 0x8c, 0x04, 0x01}}
	parsed := ParsePacket(pkt, DefaultModel())
	want := "status_response: left[in_ear,connected] right[in_ear,connected] case[open]"
	if parsed.Summary != want {
		t.Fatalf("summary = %q, want %q", parsed.Summary, want)
	}
}

func TestParseConfigRealDeviceFixture(t *testing.T) {
	payload := append([]byte{0x09}, []byte(
		"2,1,\n2,2,1.0.1.68\n2,4,SH10252535011028\n"+
			"3,1,\n3,2,1.0.1.68\n3,4,SH10252535011028\n"+
			"4,1,\n4,2,1.0.1.68\n4,4,SH10252535011028\n"+
			"3,6,9EEC4AEEBE2C")...)
	pkt := Packet{Cmd: CmdRspRemoteConfig, Payload: payload}
	parsed := ParsePacket(pkt, DefaultModel())
	want := "config_response: " +
		"left{firmware=1.0.1.68, serial=SH10252535011028} " +
		"right{firmware=1.0.1.68, serial=SH10252535011028, bt_address=9EEC4AEEBE2C} " +
		"case{firmware=1.0.1.68, serial=SH10252535011028}"
	if parsed.Summary != want {
		t.Fatalf("summary = %q, want %q", parsed.Summary, want)
	}
}

func TestParseANCRealDeviceFixture(t *testing.T) {
	// Real Ear (3) ANC: triples (1=mode,1=high,0) (2=level,1=high,0).
	pkt := Packet{Cmd: CmdRspANC, Payload: []byte{0x01, 0x01, 0x00, 0x02, 0x01, 0x00}}
	parsed := ParsePacket(pkt, DefaultModel())
	want := "anc_response: mode=high last_level=high"
	if parsed.Summary != want {
		t.Fatalf("summary = %q, want %q", parsed.Summary, want)
	}
}

func TestParseNoiseReductionEventFixture(t *testing.T) {
	// Real Ear (3) EVENT_NOISE_REDUCTION_LEVEL_CHANGED (0xE003): transparency, then high.
	trans := Packet{Cmd: CmdNoiseReductionChanged, Payload: []byte{0x01, 0x07, 0x00}}
	if got := ParsePacket(trans, DefaultModel()).Summary; got != "anc_changed: mode=transparency" {
		t.Fatalf("transparency summary = %q", got)
	}
	high := Packet{Cmd: CmdNoiseReductionChanged, Payload: []byte{0x01, 0x01, 0x00}}
	if got := ParsePacket(high, DefaultModel()).Summary; got != "anc_changed: mode=high" {
		t.Fatalf("high summary = %q", got)
	}
}

func TestParseEQRealDeviceFixture(t *testing.T) {
	pkt := Packet{Cmd: CmdRspEQ, Payload: []byte{0x00}}
	parsed := ParsePacket(pkt, DefaultModel())
	want := "eq_response: balanced"
	if parsed.Summary != want {
		t.Fatalf("summary = %q, want %q", parsed.Summary, want)
	}
}

func TestParseSpatialRealDeviceFixture(t *testing.T) {
	pkt := Packet{Cmd: CmdRspSpatial, Payload: []byte{0x00, 0x00}}
	parsed := ParsePacket(pkt, DefaultModel())
	want := "spatial_response: spatial=off head_tracking=off"
	if parsed.Summary != want {
		t.Fatalf("summary = %q, want %q", parsed.Summary, want)
	}
}

func TestParseLagModeFixture(t *testing.T) {
	pkt := Packet{Cmd: CmdRspLag, Payload: []byte{0x01}}
	if got := ParsePacket(pkt, DefaultModel()).Summary; got != "lag_response: low_latency=on mode=1" {
		t.Fatalf("lag on summary = %q", got)
	}
	pkt = Packet{Cmd: CmdRspLag, Payload: []byte{0x02}}
	if got := ParsePacket(pkt, DefaultModel()).Summary; got != "lag_response: low_latency=off mode=2" {
		t.Fatalf("lag off summary = %q", got)
	}
	pkt = Packet{Cmd: CmdAckSetLagMode}
	if got := ParsePacket(pkt, DefaultModel()).Summary; got != "lag_set_ack: ok" {
		t.Fatalf("lag set ack summary = %q", got)
	}
	pkt = Packet{Cmd: CmdLagModeChanged, Payload: []byte{0x00, 0x01}}
	if got := ParsePacket(pkt, DefaultModel()).Summary; got != "lag_changed: low_latency=on mode=1" {
		t.Fatalf("lag changed summary = %q", got)
	}
}

func TestParseSetAckPackets(t *testing.T) {
	cases := []struct {
		name    string
		cmd     uint16
		payload []byte
		want    string
	}{
		{"anc ok", CmdAckSetNoiseReduction, []byte{0x00}, "anc_set_ack: ok"},
		{"anc error", CmdAckSetNoiseReduction, []byte{0x01}, "anc_set_ack: error=0x01"},
		{"eq ok", CmdAckSetEQMode, []byte{0x00}, "eq_set_ack: ok"},
		{"spatial ok", CmdAckSetSpatialAudio, nil, "spatial_set_ack: ok"},
		{"dual enable ok", CmdAckSetDualEnable, []byte{0x00}, "dual_enable_set_ack: ok"},
		{"dual connect ok", CmdAckSetConnectDevice, []byte{0x00}, "dual_connect_set_ack: ok"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pkt := Packet{Cmd: tc.cmd, Payload: tc.payload}
			if got := ParsePacket(pkt, DefaultModel()).Summary; got != tc.want {
				t.Fatalf("summary = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestCommandLabelKnownResponses(t *testing.T) {
	cases := map[uint16]string{
		0x4027: "4027/rsp_dual_enable",
		0x4028: "4028/rsp_dual_device_list",
		0x700F: "700f/ack_set_noise_reduction",
		0x7010: "7010/ack_set_eq_mode",
		0x701A: "701a/ack_set_dual_enable",
		0x701B: "701b/ack_set_connect_device",
		0x7052: "7052/ack_set_spatial_audio",
	}
	for cmd, want := range cases {
		if got := CommandLabel(cmd); got != want {
			t.Fatalf("CommandLabel(%#04x) = %q, want %q", cmd, got, want)
		}
	}
}

func TestParseDualConnectionFixture(t *testing.T) {
	pkt := Packet{Cmd: CmdRspDualEnable, Payload: []byte{0x01}}
	if got := ParsePacket(pkt, DefaultModel()).Summary; got != "dual_response: dual=on value=1" {
		t.Fatalf("dual on summary = %q", got)
	}
	pkt = Packet{Cmd: CmdDualConnectChanged, Payload: []byte{0x00}}
	if got := ParsePacket(pkt, DefaultModel()).Summary; got != "dual_connect_changed: dual=off value=0" {
		t.Fatalf("dual event summary = %q", got)
	}
	pkt = Packet{Cmd: CmdRspDualDeviceList, Payload: []byte{
		0x01, 0x00, 0x01,
		0x11, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff,
		0x02, 'P', 'C',
	}}
	if got := ParsePacket(pkt, DefaultModel()).Summary; got != `dual_device_list: total=1 current=0 count=1 AA:BB:CC:DD:EE:FF "PC" [connected,owner]` {
		t.Fatalf("dual list summary = %q", got)
	}
	pkt = Packet{Cmd: CmdRspSupportedFeature, Payload: []byte{0x00, 0x40}}
	if got := ParsePacket(pkt, DefaultModel()).Summary; got != "supported_features: dual_list=true payload=00 40" {
		t.Fatalf("supported feature summary = %q", got)
	}
	if !SupportedFeatureDualList(pkt.Payload) {
		t.Fatal("SupportedFeatureDualList = false, want true")
	}
}

func TestDecodeAndParseFirmwareTextFixture(t *testing.T) {
	raw := BuildFrame(ControlCRC|ControlMultiFrame, CmdRspProtocolVersion, 1, []byte("1.0.0\n"))
	pkt, err := DecodePacket(raw)
	if err != nil {
		t.Fatalf("DecodePacket() error = %v", err)
	}
	if got := string(pkt.Payload); got != "1.0.0\n" {
		t.Fatalf("payload = %q, want %q", got, "1.0.0\n")
	}
	parsed := ParsePacket(pkt, DefaultModel())
	if parsed.Kind != "protocol/version" {
		t.Fatalf("kind = %q, want protocol/version", parsed.Kind)
	}
}

func TestCommandCatalogIncludesKnownCommands(t *testing.T) {
	tests := map[uint16]string{
		0xC007: "get_remote_battery_level",
		0xC042: "get_host_version_device",
		0x4007: "rsp_battery_level",
		0xF052: "set_spatial_audio",
		0xF01B: "set_connect_device",
		0xE00E: "event_dual_device_connect_state",
		0xFC09: "debug_get_debug_info",
	}
	for cmd, want := range tests {
		if got := CommandInfoFor(cmd).Name; got != want {
			t.Fatalf("CommandInfoFor(%04x) = %q, want %q", cmd, got, want)
		}
	}
}

func TestResolveModelAliases(t *testing.T) {
	for _, input := range []string{"Elekid", "2d6fda", "Nothing Headphone (1)"} {
		model, ok := ResolveModelInfo(input)
		if !ok {
			t.Fatalf("ResolveModelInfo(%q) failed", input)
		}
		if model.Codename != "Elekid" {
			t.Fatalf("ResolveModelInfo(%q) = %s, want Elekid", input, model.Codename)
		}
	}
}

func TestResolveModelFromBluetooth(t *testing.T) {
	model, source, ok := ResolveModelFromBluetooth("Nothing Ear (3)", "", "")
	if !ok {
		t.Fatal("ResolveModelFromBluetooth by name failed")
	}
	if model.Codename != "EarThree" {
		t.Fatalf("model = %s, want EarThree", model.Codename)
	}
	if source != "Nothing Ear (3)" {
		t.Fatalf("source = %q", source)
	}

	info := "ManufacturerData Key: 0x00e0\n  FEB1C7 01 02 03"
	model, source, ok = ResolveModelFromBluetooth("", "", info)
	if !ok {
		t.Fatal("ResolveModelFromBluetooth by Fast Pair ID failed")
	}
	if model.Codename != "EarTwos" {
		t.Fatalf("model = %s, want EarTwos", model.Codename)
	}
	if source != "fast_pair_id:FEB1C7" {
		t.Fatalf("source = %q", source)
	}

	model, source, ok = ResolveModelFromBluetooth("", "", "identity_response: product=Ear (3) firmware=1.0")
	if !ok {
		t.Fatal("ResolveModelFromBluetooth by embedded product failed")
	}
	if model.Codename != "EarThree" {
		t.Fatalf("model = %s, want EarThree", model.Codename)
	}
	if source != "Ear (3)" {
		t.Fatalf("source = %q", source)
	}
}

func TestFeatureCommandPackets(t *testing.T) {
	ResetFSN()
	pkt, warnings, err := FeatureCommandPacket([]string{"anc", "get"}, false, DefaultModel())
	if err != nil {
		t.Fatalf("anc get error = %v", err)
	}
	if pkt.Cmd != CmdGetNoiseReduction || !bytes.Equal(pkt.Payload, []byte{3}) {
		t.Fatalf("anc get packet = cmd=%04x payload=% x", pkt.Cmd, pkt.Payload)
	}
	raw := pkt.MarshalBinary()
	if getUint16LE(raw, 3) != CmdGetNoiseReduction {
		t.Fatalf("marshaled cmd=%04x", getUint16LE(raw, 3))
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v, want none", warnings)
	}
	_, _, err = FeatureCommandPacket([]string{"spatial", "set", "on"}, true, DefaultModel())
	if err == nil {
		t.Fatal("spatial set with unknown model succeeded")
	}
	_, _, err = FeatureCommandPacket([]string{"spatial", "set", "on", "off"}, true, DefaultModel())
	if err == nil {
		t.Fatal("spatial set with unknown model succeeded")
	}
	model, ok := ResolveModelInfo("EarThree")
	if !ok {
		t.Fatal("EarThree model not found")
	}
	pkt, _, err = FeatureCommandPacket([]string{"spatial", "set", "on", "off"}, false, model)
	if err != nil {
		t.Fatalf("spatial set error = %v", err)
	}
	if pkt.Cmd != CmdSetSpatialAudio || !bytes.Equal(pkt.Payload, []byte{1, 0}) {
		t.Fatalf("spatial set packet = cmd=%04x payload=% x", pkt.Cmd, pkt.Payload)
	}

	_, _, err = FeatureCommandPacket([]string{"anc", "set", "99"}, true, DefaultModel())
	if err == nil {
		t.Fatal("anc set with invalid mode 99 should fail")
	}
	pkt, _, err = FeatureCommandPacket([]string{"anc", "set", "off"}, false, model)
	if err != nil {
		t.Fatalf("anc set off: %v", err)
	}
	if pkt.Cmd != CmdSetNoiseReduction || !bytes.Equal(pkt.Payload, []byte{1, 5, 0}) {
		t.Fatalf("EarThree anc set off = cmd=%04x payload=% x, want f00f [01 05 00]", pkt.Cmd, pkt.Payload)
	}
	pkt, _, err = FeatureCommandPacket([]string{"lag", "set", "on"}, false, model)
	if err != nil {
		t.Fatalf("lag set on: %v", err)
	}
	if pkt.Cmd != CmdSetLagMode || !bytes.Equal(pkt.Payload, []byte{1}) {
		t.Fatalf("lag set on packet = cmd=%04x payload=% x", pkt.Cmd, pkt.Payload)
	}
	pkt, _, err = FeatureCommandPacket([]string{"lag", "set", "off"}, false, model)
	if err != nil {
		t.Fatalf("lag set off: %v", err)
	}
	if pkt.Cmd != CmdSetLagMode || !bytes.Equal(pkt.Payload, []byte{2}) {
		t.Fatalf("lag set off packet = cmd=%04x payload=% x", pkt.Cmd, pkt.Payload)
	}
	pkt, _, err = FeatureCommandPacket([]string{"dual", "get"}, false, model)
	if err != nil {
		t.Fatalf("dual get: %v", err)
	}
	if pkt.Cmd != CmdGetDualEnable {
		t.Fatalf("dual get cmd=%04x", pkt.Cmd)
	}
	pkt, _, err = FeatureCommandPacket([]string{"dual", "list"}, false, model)
	if err != nil {
		t.Fatalf("dual list: %v", err)
	}
	if pkt.Cmd != CmdGetDualDeviceList {
		t.Fatalf("dual list cmd=%04x", pkt.Cmd)
	}
	if !bytes.Equal(pkt.Payload, []byte{0}) {
		t.Fatalf("dual list payload=% x, want 00", pkt.Payload)
	}
	pkt, _, err = FeatureCommandPacket([]string{"dual", "set", "on"}, false, model)
	if err != nil {
		t.Fatalf("dual set on: %v", err)
	}
	if pkt.Cmd != CmdSetDualEnable || !bytes.Equal(pkt.Payload, []byte{1}) {
		t.Fatalf("dual set on packet = cmd=%04x payload=% x", pkt.Cmd, pkt.Payload)
	}
	pkt, warnings, err = FeatureCommandPacket([]string{"dual", "connect", "AA:BB:CC:DD:EE:FF"}, false, model)
	if err != nil {
		t.Fatalf("dual connect: %v", err)
	}
	if pkt.Cmd != CmdSetConnectDevice {
		t.Fatalf("dual connect cmd=%04x, want %04x", pkt.Cmd, CmdSetConnectDevice)
	}
	wantConnect := []byte{0x01, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}
	if !bytes.Equal(pkt.Payload, wantConnect) {
		t.Fatalf("dual connect payload=% x, want % x", pkt.Payload, wantConnect)
	}
	if len(warnings) != 0 {
		t.Fatalf("dual connect warnings = %v, want none", warnings)
	}
	pkt, _, err = FeatureCommandPacket([]string{"dual", "disconnect", "AABBCCDDEEFF"}, false, model)
	if err != nil {
		t.Fatalf("dual disconnect: %v", err)
	}
	if pkt.Cmd != CmdSetConnectDevice {
		t.Fatalf("dual disconnect cmd=%04x, want %04x", pkt.Cmd, CmdSetConnectDevice)
	}
	wantDisconnect := []byte{0x00, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}
	if !bytes.Equal(pkt.Payload, wantDisconnect) {
		t.Fatalf("dual disconnect payload=% x, want % x", pkt.Payload, wantDisconnect)
	}
}

func TestParseDualDeviceListPayload(t *testing.T) {
	list, err := ParseDualDeviceListPayload([]byte{
		0x01, 0x00, 0x01,
		0x11, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff,
		0x02, 'P', 'C',
	})
	if err != nil {
		t.Fatalf("ParseDualDeviceListPayload() error = %v", err)
	}
	if list.Total != 1 || list.Current != 0 || len(list.Devices) != 1 {
		t.Fatalf("list = %+v", list)
	}
	dev := list.Devices[0]
	if dev.MAC != "AA:BB:CC:DD:EE:FF" || dev.Name != "PC" || !dev.Connected || !dev.Owner || dev.RawState != 0x11 {
		t.Fatalf("device = %+v", dev)
	}
	parsed := ParsePacket(Packet{Cmd: CmdRspDualDeviceList, Payload: list.RawPayload}, DefaultModel())
	if parsed.DualList == nil || len(parsed.DualList.Devices) != 1 {
		t.Fatalf("parsed DualList = %+v", parsed.DualList)
	}

	list, err = ParseDualDeviceListPayload([]byte{
		0x02, 0x00, 0x01,
		0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66,
	})
	if err != nil {
		t.Fatalf("ParseDualDeviceListPayload() disconnected error = %v", err)
	}
	dev = list.Devices[0]
	if dev.MAC != "11:22:33:44:55:66" || dev.Name != "" || dev.Connected || dev.Owner {
		t.Fatalf("disconnected device = %+v", dev)
	}
	got := ParsePacket(Packet{Cmd: CmdRspDualDeviceList, Payload: list.RawPayload}, DefaultModel()).Summary
	if got != "dual_device_list: total=2 current=0 count=1 11:22:33:44:55:66 [disconnected,other]" {
		t.Fatalf("disconnected summary = %q", got)
	}
}

func TestParseDualDeviceListPayloadTruncated(t *testing.T) {
	_, err := ParseDualDeviceListPayload([]byte{0x01, 0x00, 0x02, 0x11, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff})
	if err == nil {
		t.Fatal("ParseDualDeviceListPayload() error = nil, want truncated list error")
	}
}

func TestCmdBudsBatteryCatalogMatchesParser(t *testing.T) {
	info := CommandInfoFor(CmdBudsBattery)
	if info.Name != "event_buds_battery" || info.Kind != "battery_pairs" {
		t.Fatalf("catalog = %+v, want event_buds_battery/battery_pairs", info)
	}
	parsed := ParsePacket(Packet{Cmd: CmdBudsBattery, Payload: []byte{0x01, 0x02, 0x46}}, DefaultModel())
	if parsed.Kind != "battery_buds" {
		t.Fatalf("parsed kind = %q, want battery_buds", parsed.Kind)
	}
}

func TestParseDualMAC(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    [6]byte
		wantErr bool
	}{
		{name: "colon", input: "AA:BB:CC:DD:EE:FF", want: [6]byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}},
		{name: "dash", input: "AA-BB-CC-DD-EE-FF", want: [6]byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}},
		{name: "compact", input: "aabbccddeeff", want: [6]byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}},
		{name: "short", input: "AA:BB:CC", wantErr: true},
		{name: "invalid hex", input: "GG:BB:CC:DD:EE:FF", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDualMAC(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("ParseDualMAC() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseDualMAC() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("ParseDualMAC() = % x, want % x", got, tt.want)
			}
		})
	}
}

func TestBuildDualConnectPayload(t *testing.T) {
	payload, warnings, err := BuildDualConnectPayload(true, "AA-BB-CC-DD-EE-FF")
	if err != nil {
		t.Fatalf("BuildDualConnectPayload() error = %v", err)
	}
	want := []byte{0x01, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}
	if !bytes.Equal(payload, want) {
		t.Fatalf("payload = % x, want % x", payload, want)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v, want none", warnings)
	}

	payload, _, err = BuildDualConnectPayload(false, "AA:BB:CC:DD:EE:FF")
	if err != nil {
		t.Fatalf("BuildDualConnectPayload() disconnect error = %v", err)
	}
	if payload[0] != 0 {
		t.Fatalf("disconnect flag = %d, want 0", payload[0])
	}
}

func TestReadPacketUsesLengthField(t *testing.T) {
	payload := []byte{0xab, 0xcd}
	raw := BuildFrame(ControlCRC|ControlMultiFrame, 0xC020, 5, payload)
	got, err := ReadPacket(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("ReadPacket() error = %v", err)
	}
	if !bytes.Equal(got, raw) {
		t.Fatalf("ReadPacket() = % x, want % x", got, raw)
	}
}

func TestParseScanCommand(t *testing.T) {
	start, end, delay, err := ParseScanCommand([]string{"scan", "c001", "c020", "500ms"})
	if err != nil {
		t.Fatalf("ParseScanCommand() error = %v", err)
	}
	if start != 0xc001 || end != 0xc020 || delay != 500*time.Millisecond {
		t.Fatalf("scan = %04x %04x %s, want c001 c020 500ms", start, end, delay)
	}
}
