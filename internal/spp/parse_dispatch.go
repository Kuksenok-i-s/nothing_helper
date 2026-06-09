package spp

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

type ParsedPacket struct {
	Kind      string
	Summary   string
	Text      string
	Batteries map[string]Battery
	DualList  *DualDeviceList
	Warnings  []string
}

// DualDevice is one peer entry from GET_DUAL_DEVICE_LIST (0x4028).
type DualDevice struct {
	MAC       string
	Name      string
	Connected bool
	Owner     bool
	RawState  byte
}

// DualDeviceList is the parsed payload of GET_DUAL_DEVICE_LIST.
type DualDeviceList struct {
	Total      byte
	Current    byte
	Devices    []DualDevice
	RawPayload []byte
}

type PacketParser func(Packet, ModelInfo) ParsedPacket

var packetParsers = map[uint16]PacketParser{
	CmdBatteryChanged:        parseBatteryPacket("battery_changed"),
	CmdBattery:               parseBatteryPacket("battery_full"),
	CmdBudsBattery:           parseBatteryPacket("battery_buds"),
	CmdStatus:                parseStatusPacket("status_changed"),
	CmdNoiseReductionChanged: parseANCPacket("anc_changed"),
	CmdDualSwitchChanged:     parseDualPacket("dual_switch_changed"),
	CmdDualConnectChanged:    parseDualPacket("dual_connect_changed"),
	CmdLagModeChanged:        parseLagPacket("lag_changed"),
	CmdIdentity:              parseRawPacket("identity_raw"),

	CmdRspBattery:          parseBatteryPacket("battery_response"),
	CmdRspStatus:           parseStatusPacket("status_response"),
	CmdRspIdentity:         parseRawPacket("identity_response"),
	CmdRspRemoteConfig:     parseConfigPacket("config_response"),
	CmdRspSupportedFeature: parseSupportedFeaturePacket("supported_features"),
	CmdRspFirmware:         parseTextPacket("firmware/version"),
	CmdRspProtocolVersion:  parseTextPacket("protocol/version"),
	CmdRspANC:              parseANCPacket("anc_response"),
	CmdRspEQ:               parseEQPacket("eq_response"),
	CmdRspDualEnable:       parseDualPacket("dual_response"),
	CmdRspDualDeviceList:   parseDualDeviceListPacket("dual_device_list"),
	CmdRspLag:              parseLagPacket("lag_response"),
	CmdRspSpatial:          parseSpatialPacket("spatial_response"),

	CmdAckSetLagMode:        parseSetAckPacket("lag_set_ack"),
	CmdAckSetNoiseReduction: parseSetAckPacket("anc_set_ack"),
	CmdAckSetEQMode:         parseSetAckPacket("eq_set_ack"),
	CmdAckSetSpatialAudio:   parseSetAckPacket("spatial_set_ack"),
	CmdAckSetDualEnable:     parseSetAckPacket("dual_enable_set_ack"),
	CmdAckSetConnectDevice:  parseSetAckPacket("dual_connect_set_ack"),
}

// parseCountedPairs decodes the counted "[count][id,val]*count" layout used by
// battery/status responses (DataExtKt.toPairs with count=1, id=1, val=1).
func parseCountedPairs(payload []byte) ([][2]byte, bool) {
	if len(payload) < 1 {
		return nil, false
	}
	count := int(payload[0])
	if len(payload) < 1+count*2 {
		return nil, false
	}
	out := make([][2]byte, 0, count)
	for i := 0; i < count; i++ {
		off := 1 + i*2
		out = append(out, [2]byte{payload[off], payload[off+1]})
	}
	return out, true
}

func parseBatteryPacket(kind string) PacketParser {
	return func(pkt Packet, model ModelInfo) ParsedPacket {
		pairs, ok := parseCountedPairs(pkt.Payload)
		if !ok {
			return ParsedPacket{
				Kind:    kind,
				Summary: fmt.Sprintf("%s: % x", kind, pkt.Payload),
				Warnings: []string{
					"payload does not match [count][id,value] battery layout",
				},
			}
		}

		data := map[string]Battery{}
		for _, p := range pairs {
			name, known := partNames[p[0]]
			if !known {
				name = fmt.Sprintf("id_%d", p[0])
			}
			data[name] = Battery{
				Percent:  int(p[1] & 0x7F),
				Charging: p[1]&0x80 != 0,
			}
		}
		data, warnings := NormalizeBatteryForModel(data, model)

		return ParsedPacket{
			Kind:      kind,
			Summary:   fmt.Sprintf("%s: %s", kind, FormatBattery(data)),
			Batteries: data,
			Warnings:  warnings,
		}
	}
}

// parseStatusPacket decodes GET_EARPHONE_STATUS (rsp 0x400A): [count][id,flags]*.
// Flag bits per EarphoneStatus: bit0=in_case/case_open, bit2=in_ear, bit7=connected.
func parseStatusPacket(kind string) PacketParser {
	return func(pkt Packet, model ModelInfo) ParsedPacket {
		pairs, ok := parseCountedPairs(pkt.Payload)
		if !ok {
			return ParsedPacket{
				Kind:    kind,
				Summary: fmt.Sprintf("%s: rsp=%02x payload=% x", kind, pkt.RspCode(), pkt.Payload),
			}
		}

		parts := make([]string, 0, len(pairs))
		for _, p := range pairs {
			name, known := partNames[p[0]]
			if !known {
				name = fmt.Sprintf("id_%d", p[0])
			}
			v := p[1]
			var flags []string
			if p[0] == 4 { // case: bit0 = lid open
				if v&0x01 != 0 {
					flags = append(flags, "open")
				} else {
					flags = append(flags, "closed")
				}
			} else {
				if v&0x04 != 0 {
					flags = append(flags, "in_ear")
				} else if v&0x01 != 0 {
					flags = append(flags, "in_case")
				} else {
					flags = append(flags, "out")
				}
				if v&0x80 != 0 {
					flags = append(flags, "connected")
				}
			}
			parts = append(parts, fmt.Sprintf("%s[%s]", name, strings.Join(flags, ",")))
		}

		return ParsedPacket{
			Kind:    kind,
			Summary: fmt.Sprintf("%s: %s", kind, strings.Join(parts, " ")),
		}
	}
}

var configTypeLabels = map[int]string{
	1: "hardware",
	2: "firmware",
	3: "firmware_backup",
	4: "serial",
	5: "manufacture_date",
	6: "bt_address",
}

// parseConfigPacket decodes GET_REMOTE_CONFIGURATION (rsp 0x4006): a leading
// count byte followed by newline-separated "device,type,value" CSV records.
func parseConfigPacket(kind string) PacketParser {
	return func(pkt Packet, model ModelInfo) ParsedPacket {
		if len(pkt.Payload) < 1 {
			return ParsedPacket{Kind: kind, Summary: kind + ": (empty)"}
		}

		text := strings.TrimSpace(string(pkt.Payload[1:]))
		grouped := map[byte][]string{}
		order := []byte{}
		for _, line := range strings.Split(text, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			fields := strings.SplitN(line, ",", 3)
			if len(fields) != 3 {
				continue
			}
			dev, err1 := strconv.Atoi(fields[0])
			typ, err2 := strconv.Atoi(fields[1])
			val := fields[2]
			if err1 != nil || err2 != nil || val == "" {
				continue
			}
			label, ok := configTypeLabels[typ]
			if !ok {
				label = fmt.Sprintf("type_%d", typ)
			}
			id := byte(dev)
			if _, seen := grouped[id]; !seen {
				order = append(order, id)
			}
			grouped[id] = append(grouped[id], fmt.Sprintf("%s=%s", label, val))
		}

		parts := make([]string, 0, len(order))
		for _, id := range order {
			name, known := partNames[id]
			if !known {
				name = fmt.Sprintf("id_%d", id)
			}
			parts = append(parts, fmt.Sprintf("%s{%s}", name, strings.Join(grouped[id], ", ")))
		}

		return ParsedPacket{
			Kind:    kind,
			Text:    text,
			Summary: fmt.Sprintf("%s: %s", kind, strings.Join(parts, " ")),
		}
	}
}

func parseSupportedFeaturePacket(kind string) PacketParser {
	return func(pkt Packet, model ModelInfo) ParsedPacket {
		return ParsedPacket{
			Kind:    kind,
			Summary: fmt.Sprintf("%s: dual_list=%t payload=% x", kind, SupportedFeatureDualList(pkt.Payload), pkt.Payload),
		}
	}
}

// SupportedFeatureDualList uses payload[1] bit 6 as GET_DUAL_DEVICE_LIST gate.
func SupportedFeatureDualList(payload []byte) bool {
	return len(payload) > 1 && payload[1]&0x40 != 0
}

func parseRawPacket(kind string) PacketParser {
	return func(pkt Packet, model ModelInfo) ParsedPacket {
		return ParsedPacket{
			Kind:    kind,
			Summary: fmt.Sprintf("%s: rsp=%02x payload=% x", kind, pkt.RspCode(), pkt.Payload),
		}
	}
}

func parseTextPacket(kind string) PacketParser {
	return func(pkt Packet, model ModelInfo) ParsedPacket {
		return ParsedPacket{
			Kind:    kind,
			Text:    string(pkt.Payload),
			Summary: fmt.Sprintf("%s raw=%q hex=% x", kind, pkt.Payload, pkt.Payload),
		}
	}
}

// ANC (GET_CURRENT_NOISE_REDUCTION, rsp 0x401E) is a list of 3-byte triples
// (type, value, none); type 1 = mode/tab, type 2 = last manual level.
var ancModeLabels = map[byte]string{
	0:   "off",
	1:   "high",
	2:   "mid",
	3:   "low",
	4:   "adaptive",
	5:   "off",
	7:   "transparency",
	254: "transparency",
}

var ancLevelLabels = map[byte]string{
	1: "high",
	2: "mid",
	3: "low",
	4: "adaptive",
}

func ancModeLabel(v byte) string {
	if label, ok := ancModeLabels[v]; ok {
		return label
	}
	if v >= 1 && v <= 127 {
		return fmt.Sprintf("anc(level=%d)", v)
	}
	return fmt.Sprintf("mode_%d", v)
}

func parseANCPacket(kind string) PacketParser {
	return func(pkt Packet, model ModelInfo) ParsedPacket {
		p := pkt.Payload
		if len(p) < 3 || len(p)%3 != 0 {
			return ParsedPacket{
				Kind:    kind,
				Summary: fmt.Sprintf("%s: rsp=%02x payload=% x", kind, pkt.RspCode(), p),
			}
		}
		mode, level := "", ""
		for i := 0; i+2 < len(p); i += 3 {
			switch p[i] {
			case 1:
				mode = ancModeLabel(p[i+1])
			case 2:
				if label, ok := ancLevelLabels[p[i+1]]; ok {
					level = label
				} else {
					level = fmt.Sprintf("level_%d", p[i+1])
				}
			}
		}
		summary := fmt.Sprintf("%s: mode=%s", kind, mode)
		if level != "" {
			summary += " last_level=" + level
		}
		return ParsedPacket{Kind: kind, Summary: summary}
	}
}

// EQ preset (GET_EQ_MODE, rsp 0x401F): single byte preset index.
var eqPresetLabels = map[byte]string{
	0: "balanced",
	1: "voice",
	2: "more_treble",
	3: "more_bass",
	4: "custom",
}

func parseEQPacket(kind string) PacketParser {
	return func(pkt Packet, model ModelInfo) ParsedPacket {
		if len(pkt.Payload) < 1 {
			return ParsedPacket{Kind: kind, Summary: kind + ": (empty)"}
		}
		v := pkt.Payload[0]
		label, ok := eqPresetLabels[v]
		if !ok {
			label = fmt.Sprintf("preset_%d", v)
		}
		return ParsedPacket{Kind: kind, Summary: fmt.Sprintf("%s: %s", kind, label)}
	}
}

// Spatial audio (GET_SPATIAL_AUDIO, rsp 0x404F): [spatial, head_tracking] flags.
func parseSpatialPacket(kind string) PacketParser {
	return func(pkt Packet, model ModelInfo) ParsedPacket {
		p := pkt.Payload
		if len(p) < 2 {
			return ParsedPacket{Kind: kind, Summary: fmt.Sprintf("%s: payload=% x", kind, p)}
		}
		onoff := func(b byte) string {
			if b != 0 {
				return "on"
			}
			return "off"
		}
		return ParsedPacket{
			Kind:    kind,
			Summary: fmt.Sprintf("%s: spatial=%s head_tracking=%s", kind, onoff(p[0]), onoff(p[1])),
		}
	}
}

func parseLagPacket(kind string) PacketParser {
	return func(pkt Packet, model ModelInfo) ParsedPacket {
		if len(pkt.Payload) < 1 {
			return ParsedPacket{
				Kind:    kind,
				Summary: fmt.Sprintf("%s: rsp=%02x payload=% x", kind, pkt.RspCode(), pkt.Payload),
			}
		}
		mode := pkt.Payload[0]
		if len(pkt.Payload) >= 2 && pkt.Payload[0] == 0 {
			mode = pkt.Payload[1]
		}
		state := "on"
		if mode == 2 {
			state = "off"
		}
		return ParsedPacket{
			Kind:    kind,
			Summary: fmt.Sprintf("%s: low_latency=%s mode=%d", kind, state, mode),
		}
	}
}

// parseSetAckPacket decodes the 0x7XXX acknowledgement for a 0xFXXX SET command.
// The first payload byte is a status code where 0x00 means success.
func parseSetAckPacket(kind string) PacketParser {
	return func(pkt Packet, model ModelInfo) ParsedPacket {
		if len(pkt.Payload) == 0 {
			return ParsedPacket{
				Kind:    kind,
				Summary: fmt.Sprintf("%s: ok", kind),
			}
		}
		status := pkt.Payload[0]
		result := "ok"
		if status != 0 {
			result = fmt.Sprintf("error=0x%02x", status)
		}
		if len(pkt.Payload) > 1 {
			return ParsedPacket{
				Kind:    kind,
				Summary: fmt.Sprintf("%s: %s payload=% x", kind, result, pkt.Payload),
			}
		}
		return ParsedPacket{
			Kind:    kind,
			Summary: fmt.Sprintf("%s: %s", kind, result),
		}
	}
}

func parseDualPacket(kind string) PacketParser {
	return func(pkt Packet, model ModelInfo) ParsedPacket {
		if len(pkt.Payload) < 1 {
			return ParsedPacket{
				Kind:    kind,
				Summary: fmt.Sprintf("%s: rsp=%02x payload=% x", kind, pkt.RspCode(), pkt.Payload),
			}
		}
		enabled := pkt.Payload[0] == 1
		state := "off"
		if enabled {
			state = "on"
		}
		return ParsedPacket{
			Kind:    kind,
			Summary: fmt.Sprintf("%s: dual=%s value=%d", kind, state, pkt.Payload[0]),
		}
	}
}

func parseDualDeviceListPacket(kind string) PacketParser {
	return func(pkt Packet, model ModelInfo) ParsedPacket {
		if len(pkt.Payload) == 0 {
			return ParsedPacket{Kind: kind, Summary: kind + ": empty"}
		}
		list, err := ParseDualDeviceListPayload(pkt.Payload)
		if err != nil {
			return ParsedPacket{
				Kind:     kind,
				Summary:  fmt.Sprintf("%s: payload=% x", kind, pkt.Payload),
				Warnings: []string{err.Error()},
			}
		}
		summaries := FormatDualDeviceSummaries(list.Devices)
		if len(summaries) == 0 {
			return ParsedPacket{
				Kind:     kind,
				Summary:  fmt.Sprintf("%s: total=%d current=%d count=%d payload=% x", kind, list.Total, list.Current, len(list.Devices), pkt.Payload),
				DualList: &list,
			}
		}
		return ParsedPacket{
			Kind:     kind,
			Summary:  fmt.Sprintf("%s: total=%d current=%d count=%d %s", kind, list.Total, list.Current, len(list.Devices), strings.Join(summaries, " | ")),
			DualList: &list,
		}
	}
}

// ParseDualDeviceListPayload decodes the dual device list response body.
func ParseDualDeviceListPayload(payload []byte) (DualDeviceList, error) {
	if len(payload) < 3 {
		return DualDeviceList{}, fmt.Errorf("dual device list payload too short: %d bytes", len(payload))
	}
	list := DualDeviceList{
		Total:      payload[0],
		Current:    payload[1],
		RawPayload: append([]byte(nil), payload...),
	}
	count := int(payload[2])
	devices, err := parseDualDeviceRecords(payload[3:], count)
	if err != nil {
		return DualDeviceList{}, err
	}
	list.Devices = devices
	return list, nil
}

func parseDualDeviceRecords(payload []byte, count int) ([]DualDevice, error) {
	devices := make([]DualDevice, 0, count)
	off := 0
	for i := 0; i < count; i++ {
		if off+7 > len(payload) {
			return nil, fmt.Errorf("dual list truncated: want %d records, parsed %d", count, i)
		}
		state := payload[off]
		macBytes := payload[off+1 : off+7]
		off += 7

		name := ""
		if off < len(payload) {
			nameLen := int(payload[off] & 0x7F)
			remainingDevices := count - i - 1
			if nameLen <= 31 && off+1+nameLen <= len(payload) && len(payload)-(off+1+nameLen) >= remainingDevices*7 {
				name = cleanDualDeviceName(payload[off+1 : off+1+nameLen])
				off += 1 + nameLen
			} else if off+31 <= len(payload) && len(payload)-(off+31) >= remainingDevices*7 {
				name = cleanDualDeviceName(payload[off : off+31])
				off += 31
			}
		}

		devices = append(devices, DualDevice{
			MAC:       formatDualMAC(macBytes),
			Name:      name,
			Connected: state&0x0F != 0,
			Owner:     state&0xF0 != 0,
			RawState:  state,
		})
	}
	if len(devices) != count {
		return nil, fmt.Errorf("dual list truncated: want %d records, parsed %d", count, len(devices))
	}
	return devices, nil
}

func FormatDualDeviceSummaries(devices []DualDevice) []string {
	items := make([]string, 0, len(devices))
	for _, dev := range devices {
		connected := "disconnected"
		if dev.Connected {
			connected = "connected"
		}
		owner := "other"
		if dev.Owner {
			owner = "owner"
		}
		if dev.Name != "" {
			items = append(items, fmt.Sprintf("%s %q [%s,%s]", dev.MAC, dev.Name, connected, owner))
		} else {
			items = append(items, fmt.Sprintf("%s [%s,%s]", dev.MAC, connected, owner))
		}
	}
	return items
}

// ParseDualMAC accepts AA:BB:CC:DD:EE:FF, AA-BB-..., or AABBCCDDEEFF.
func ParseDualMAC(raw string) ([6]byte, error) {
	var out [6]byte
	s := strings.ToUpper(strings.TrimSpace(raw))
	s = strings.ReplaceAll(s, ":", "")
	s = strings.ReplaceAll(s, "-", "")
	if len(s) != 12 {
		return out, fmt.Errorf("invalid MAC %q: want 6 bytes in AA:BB:CC:DD:EE:FF form", raw)
	}
	for i := 0; i < 6; i++ {
		part, err := strconv.ParseUint(s[i*2:i*2+2], 16, 8)
		if err != nil {
			return out, fmt.Errorf("invalid MAC %q: %w", raw, err)
		}
		out[i] = byte(part)
	}
	return out, nil
}

// BuildDualConnectPayload builds the SET_CONNECT_DEVICE (0xF01B) payload.
func BuildDualConnectPayload(connect bool, mac string) ([]byte, []string, error) {
	addr, err := ParseDualMAC(mac)
	if err != nil {
		return nil, nil, err
	}
	flag := byte(0)
	if connect {
		flag = 1
	}
	payload := make([]byte, 7)
	payload[0] = flag
	copy(payload[1:], addr[:])
	return payload, nil, nil
}

func cleanDualDeviceName(raw []byte) string {
	end := len(raw)
	for end > 0 && (raw[end-1] == 0 || raw[end-1] == ' ') {
		end--
	}
	return strings.TrimSpace(string(raw[:end]))
}

func formatDualMAC(raw []byte) string {
	if len(raw) < 6 {
		return fmt.Sprintf("% x", raw)
	}
	return fmt.Sprintf("%02X:%02X:%02X:%02X:%02X:%02X", raw[0], raw[1], raw[2], raw[3], raw[4], raw[5])
}

func ParsePacket(pkt Packet, model ModelInfo) ParsedPacket {
	for _, key := range []uint16{pkt.ParserKey(), pkt.Cmd, pkt.ResponseCmd()} {
		if parser, ok := packetParsers[key]; ok {
			return parser(pkt, model)
		}
	}
	return parseUnknownPacket(pkt, model)
}

func parseUnknownPacket(pkt Packet, model ModelInfo) ParsedPacket {
	if text, ok := printableText(pkt.Payload); ok {
		return ParsedPacket{
			Kind:    "unknown_text",
			Text:    text,
			Summary: fmt.Sprintf("unknown_text: rsp=%02x cmd=%04x text=%q hex=% x", pkt.RspCode(), pkt.Cmd, text, pkt.Payload),
		}
	}

	if data := ParsePairs(pkt.Payload); data != nil {
		data, warnings := NormalizeBatteryForModel(data, model)
		warnings = append(warnings, "unknown command has payload shaped like battery id/value pairs")

		return ParsedPacket{
			Kind:      "unknown_battery_pairs",
			Summary:   fmt.Sprintf("unknown_battery_pairs: rsp=%02x cmd=%04x %s", pkt.RspCode(), pkt.Cmd, FormatBattery(data)),
			Batteries: data,
			Warnings:  warnings,
		}
	}

	bitView := ""
	if len(pkt.Payload) > 0 && len(pkt.Payload) <= 8 {
		parts := make([]string, 0, len(pkt.Payload))
		for _, b := range pkt.Payload {
			parts = append(parts, fmt.Sprintf("%08b", b))
		}
		bitView = " bits=" + strings.Join(parts, " ")
	}

	return ParsedPacket{
		Kind:    "unknown",
		Summary: fmt.Sprintf("unknown: rsp=%02x cmd=%04x payload=% x%s", pkt.RspCode(), pkt.Cmd, pkt.Payload, bitView),
	}
}

func printableText(payload []byte) (string, bool) {
	if len(payload) == 0 || !utf8.Valid(payload) {
		return "", false
	}

	text := payload
	for len(text) > 0 {
		r, size := utf8.DecodeRune(text)
		if r == utf8.RuneError && size == 1 {
			return "", false
		}
		if r != '\n' && r != '\r' && r != '\t' && !unicode.IsPrint(r) {
			return "", false
		}
		text = text[size:]
	}

	return string(payload), true
}

func ReadPacket(r io.Reader) ([]byte, error) {
	skipped := 0
	for {
		b := make([]byte, 1)
		if _, err := io.ReadFull(r, b); err != nil {
			return nil, err
		}
		if b[0] != SOF {
			skipped++
			if skipped > maxSOFScanBytes {
				return nil, fmt.Errorf("no SOF found within %d bytes", maxSOFScanBytes)
			}
			continue
		}
		buf := []byte{SOF}
			headerRest, err := readExact(r, 7)
			if err != nil {
				return nil, err
			}
			buf = append(buf, headerRest...)

			length := int(getUint16LE(buf, 5))
			if length > MaxPayloadLen {
				return nil, fmt.Errorf("payload length too large: %d", length)
			}

			payload, err := readExact(r, length)
			if err != nil {
				return nil, err
			}
			buf = append(buf, payload...)

			if getUint16LE(buf, 1)&ControlCRC != 0 {
				crc, err := readExact(r, 2)
				if err != nil {
					return nil, err
				}
				buf = append(buf, crc...)
			}

			return buf, nil
	}
}

