package spp

import "testing"

// Synthetic fixtures derived from the APK, not recordings from hardware.
func TestWalkieTalkieReceive(t *testing.T) {
	for _, cmd := range []uint16{CmdRspSuperMic, CmdRspWalkieTalkie, CmdWalkieTalkieChanged} {
		for _, payload := range [][]byte{nil, {0}, {1}, {2}, {255}, {1, 0}} {
			p := ParsePacket(Packet{Cmd: cmd, Payload: payload}, ModelInfo{})
			flag, other := p.WalkieTalkieEnabled, p.SuperMicEnabled
			if cmd == CmdRspSuperMic {
				flag, other = other, flag
			}
			valid := len(payload) == 1 && payload[0] <= 1
			if (flag != nil) != valid || other != nil {
				t.Fatalf("cmd=%04x payload=%x: unexpected flags %+v", cmd, payload, p)
			}
			if valid && (*flag != (payload[0] == 1) || len(p.Warnings) != 0) {
				t.Fatalf("cmd=%04x payload=%x: wrong state %+v", cmd, payload, p)
			}
			if !valid && len(p.Warnings) == 0 {
				t.Fatalf("cmd=%04x payload=%x: missing warning", cmd, payload)
			}
			if p.Batteries != nil || p.Text != "" || p.Summary == "" {
				t.Fatalf("unexpected fallback or missing summary: %+v", p)
			}
		}
	}
}
