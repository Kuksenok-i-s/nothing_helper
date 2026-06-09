package spp

import (
	"fmt"
	"io"
)

func readExact(r io.Reader, n int) ([]byte, error) {
	buf := make([]byte, n)
	_, err := io.ReadFull(r, buf)
	return buf, err
}

func DecodePacket(raw []byte) (Packet, error) {
	if len(raw) < 8 {
		return Packet{Raw: raw}, fmt.Errorf("packet too short: %d bytes", len(raw))
	}

	if raw[0] != SOF {
		return Packet{Raw: raw}, fmt.Errorf("invalid SOF: %02x", raw[0])
	}

	control := getUint16LE(raw, 1)
	cmd := getUint16LE(raw, 3)
	length := getUint16LE(raw, 5)
	fsn := raw[7]
	if length > MaxPayloadLen {
		return Packet{Raw: raw}, fmt.Errorf("payload length too large: %d", length)
	}

	want := 8 + int(length)
	if control&ControlCRC != 0 {
		want += 2
	}
	if len(raw) < want {
		return Packet{Raw: raw}, fmt.Errorf("truncated payload: length=%d raw_len=%d", length, len(raw))
	}

	pkt := Packet{
		Control: control,
		Cmd:     cmd,
		Length:  length,
		FSN:     fsn,
		Raw:     append([]byte(nil), raw...),
	}

	if length > 0 {
		pkt.Payload = append([]byte(nil), raw[8:8+length]...)
	}

	if control&ControlCRC != 0 {
		off := 8 + int(length)
		pkt.CRC = getUint16LE(raw, off)
		computed := CRC16(raw[:off])
		pkt.CRCValid = computed == pkt.CRC
	}

	return pkt, nil
}

func ParsePairs(payload []byte) map[string]Battery {
	if len(payload) == 0 || len(payload)%2 != 0 {
		return nil
	}

	result := map[string]Battery{}

	for i := 0; i < len(payload); i += 2 {
		kind := payload[i]
		val := payload[i+1]

		name, ok := partNames[kind]
		if !ok {
			return nil
		}

		result[name] = Battery{
			Percent:  int(val & 0x7F),
			Charging: val&0x80 != 0,
		}
	}

	return result
}

func FormatBattery(data map[string]Battery) string {
	required := []string{"left", "right", "case"}
	extras := []string{"watch", "tws", "stereo"}
	out := ""

	appendPart := func(part string) {
		if out == "" {
			out = part
		} else {
			out += " | " + part
		}
	}

	formatPart := func(name string, item Battery) string {
		part := fmt.Sprintf("%s: %d%%", name, item.Percent)
		if item.Charging {
			part += " charging"
		}

		return part
	}

	for _, name := range required {
		item, ok := data[name]
		if !ok {
			appendPart(fmt.Sprintf("%s: n/a", name))
			continue
		}

		appendPart(formatPart(name, item))
	}

	for _, name := range extras {
		item, ok := data[name]
		if !ok {
			continue
		}

		appendPart(formatPart(name, item))
	}

	return out
}

func NormalizeBatteryForModel(data map[string]Battery, model ModelInfo) (map[string]Battery, []string) {
	if data == nil {
		return nil, nil
	}

	out := make(map[string]Battery, len(data)+1)
	for key, value := range data {
		out[key] = value
	}

	var warnings []string
	if model.BatteryCaseSource != "stereo" {
		return out, warnings
	}

	stereo, hasStereo := out["stereo"]
	if !hasStereo {
		return out, warnings
	}

	if _, hasCase := out["case"]; hasCase {
		warnings = append(warnings, "model uses stereo battery as case battery, but payload also includes case; keeping explicit case value")
		return out, warnings
	}

	out["case"] = stereo
	delete(out, "stereo")

	if model.Codename != "" {
		warnings = append(warnings, fmt.Sprintf("%s maps stereo battery (id=6) to case battery", model.Codename))
	}

	return out, warnings
}

