package spp

import (
	"bytes"
	"testing"
)

func TestCRC16KnownVector(t *testing.T) {
	// SOF + control + cmd + length + fsn for GET battery, fsn=1.
	data := []byte{0x55, 0x60, 0x01, 0x07, 0xC0, 0x00, 0x00, 0x01}
	got := CRC16(data)
	if got == 0 {
		t.Fatal("CRC16 returned zero")
	}
	raw := Packet{Cmd: CmdGetBattery, FSN: 1, FixedFSN: true}.MarshalBinary()
	if len(raw) < 10 {
		t.Fatalf("marshal too short: % x", raw)
	}
	pktCRC := getUint16LE(raw, len(raw)-2)
	if CRC16(raw[:len(raw)-2]) != pktCRC {
		t.Fatalf("frame CRC %04x != computed %04x", pktCRC, CRC16(raw[:len(raw)-2]))
	}
}

func TestMarshalDecodeRoundTrip(t *testing.T) {
	ResetFSN()
	tests := []struct {
		name    string
		cmd     uint16
		payload []byte
	}{
		{"battery_get", CmdGetBattery, nil},
		{"identity_get", CmdGetIdentity, nil},
		{"anc_get", CmdGetNoiseReduction, []byte{3}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := Packet{Cmd: tt.cmd, Payload: tt.payload}.MarshalBinary()
			pkt, err := DecodePacket(raw)
			if err != nil {
				t.Fatalf("DecodePacket: %v", err)
			}
			if pkt.Cmd != tt.cmd {
				t.Fatalf("cmd=%04x want %04x", pkt.Cmd, tt.cmd)
			}
			if !bytes.Equal(pkt.Payload, tt.payload) {
				t.Fatalf("payload=% x want % x", pkt.Payload, tt.payload)
			}
			if !pkt.CRCValid {
				t.Fatal("CRCValid=false")
			}
			if raw[0] != SOF || getUint16LE(raw, 1) != ControlTXDefault {
				t.Fatalf("header=% x", raw[:8])
			}
		})
	}
}

func TestReadPacketFraming(t *testing.T) {
	payload := []byte{0x02, 0x46, 0x03, 0x55, 0x04, 0x64}
	frame1 := BuildFrame(ControlCRC|ControlMultiFrame, CmdBattery, 2, payload)
	textPayload := []byte("1.0.0\n")
	frame2 := BuildFrame(ControlCRC|ControlMultiFrame, CmdRspProtocolVersion, 3, textPayload)

	stream := bytes.NewReader(append([]byte{0x00, 0xff}, append(frame1, frame2...)...))
	got1, err := ReadPacket(stream)
	if err != nil || !bytes.Equal(got1, frame1) {
		t.Fatalf("frame1: err=%v got=% x want=% x", err, got1, frame1)
	}
	got2, err := ReadPacket(stream)
	if err != nil || !bytes.Equal(got2, frame2) {
		t.Fatalf("frame2: err=%v got=% x want=% x", err, got2, frame2)
	}
}

func TestDecodeProtocolVersionResponse(t *testing.T) {
	raw := BuildFrame(ControlCRC|ControlMultiFrame, CmdRspProtocolVersion, 1, []byte("1.0.0\n"))
	pkt, err := DecodePacket(raw)
	if err != nil {
		t.Fatalf("DecodePacket: %v", err)
	}
	if pkt.ResponseCmd() != CmdRspProtocolVersion {
		t.Fatalf("response cmd=%04x", pkt.ResponseCmd())
	}
	parsed := ParsePacket(pkt, DefaultModel())
	if parsed.Kind != "protocol/version" {
		t.Fatalf("kind=%q", parsed.Kind)
	}
	if parsed.Text != "1.0.0\n" {
		t.Fatalf("text=%q", parsed.Text)
	}
}

func TestDecodePacketRejectsOversizedLength(t *testing.T) {
	raw := []byte{SOF, 0x60, 0x01, 0x07, 0xC0, 0x10, 0x00, 0x01}
	_, err := DecodePacket(raw)
	if err == nil {
		t.Fatal("DecodePacket() error = nil, want payload length error")
	}
}

func TestReadPacketRejectsOversizedLength(t *testing.T) {
	stream := bytes.NewReader([]byte{SOF, 0x60, 0x01, 0x07, 0xC0, 0x10, 0x00, 0x01})
	_, err := ReadPacket(stream)
	if err == nil {
		t.Fatal("ReadPacket() error = nil, want payload length error")
	}
}

func TestReadPacketSOFScanLimit(t *testing.T) {
	garbage := bytes.Repeat([]byte{0x00}, maxSOFScanBytes+1)
	_, err := ReadPacket(bytes.NewReader(garbage))
	if err == nil {
		t.Fatal("ReadPacket() error = nil, want SOF scan limit error")
	}
}

func TestNextFSNSequence(t *testing.T) {
	ResetFSN()
	if got := NextFSN(); got != 1 {
		t.Fatalf("first FSN=%d want 1", got)
	}
	if got := NextFSN(); got != 2 {
		t.Fatalf("second FSN=%d want 2", got)
	}
}
