package spp

import "fmt"

// Derived from Nothing X 3.5.3 ARM64 AOT; see walkie_talkie.md.
// SETs are restricted to Ear (3) and a single boolean byte.
const (
	CmdGetSuperMic         = 0xC05E
	CmdSetSuperMic         = 0xF05F
	CmdGetWalkieTalkie     = 0xC060
	CmdSetWalkieTalkie     = 0xF061
	CmdRspSuperMic         = 0x405E
	CmdRspWalkieTalkie     = 0x4060
	CmdWalkieTalkieChanged = 0xE01A
)

func parseSuperMicPacket(pkt Packet, _ ModelInfo) ParsedPacket {
	p, enabled := parseMicFlag(pkt, "super_mic_response")
	p.SuperMicEnabled = enabled
	return p
}

func parseWalkieTalkiePacket(pkt Packet, _ ModelInfo) ParsedPacket {
	kind := "walkie_talkie_response"
	if pkt.Cmd == CmdWalkieTalkieChanged {
		kind = "walkie_talkie_changed"
	}
	p, enabled := parseMicFlag(pkt, kind)
	p.WalkieTalkieEnabled = enabled
	return p
}

func parseMicFlag(pkt Packet, kind string) (ParsedPacket, *bool) {
	p := ParsedPacket{Kind: kind}
	// Conservatively reject extensions and unknown values until captured on
	// hardware. Missing/unsupported data must not become an observed "off".
	if len(pkt.Payload) != 1 || pkt.Payload[0] > 1 {
		p.Summary = fmt.Sprintf("%s: unknown payload_bytes=%d", kind, len(pkt.Payload))
		p.Warnings = []string{"expected one boolean byte (0 or 1)"}
		return p, nil
	}
	enabled := pkt.Payload[0] == 1
	p.Summary = fmt.Sprintf("%s: enabled=%t", kind, enabled)
	return p, &enabled
}

func ValidateMicCommand(pkt Packet, model ModelInfo) error {
	if model.Codename != "EarThree" {
		return fmt.Errorf("Super Mic / Walkie Talkie requires Ear (3)")
	}
	if pkt.Cmd == CmdGetSuperMic || pkt.Cmd == CmdGetWalkieTalkie {
		if len(pkt.Payload) != 0 {
			return fmt.Errorf("microphone query requires an empty payload")
		}
		return nil
	}
	if len(pkt.Payload) != 1 || pkt.Payload[0] > 1 {
		return fmt.Errorf("microphone mode requires one boolean byte")
	}
	return nil
}
func buildMicPayload(model ModelInfo, args []string) ([]byte, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("expected on or off")
	}
	b, err := parseBoolByte(args[0])
	if err != nil {
		return nil, err
	}
	pkt := Packet{Cmd: CmdSetWalkieTalkie, Payload: []byte{b}}
	if err := ValidateMicCommand(pkt, model); err != nil {
		return nil, err
	}
	return pkt.Payload, nil
}
