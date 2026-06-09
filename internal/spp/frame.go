package spp

import (
	"encoding/binary"
	"sync"
)

const (
	SOF = 0x55

	// MaxPayloadLen is the largest accepted wire payload size (bytes).
	MaxPayloadLen = 4096
	// maxSOFScanBytes limits how many non-SOF bytes ReadPacket skips before failing.
	maxSOFScanBytes = 65536

	ControlCRC        = 0x20
	ControlMultiFrame = 0x40
	DeviceTypeTWS     = 1 << 8
	ControlTXDefault  = ControlCRC | ControlMultiFrame | DeviceTypeTWS // 0x0160

	// Response command IDs (request & 0x7FFF).
	CmdRspProtocolVersion  = 0x4001
	CmdRspIdentity         = 0x4005
	CmdRspRemoteConfig     = 0x4006
	CmdRspBattery          = 0x4007
	CmdRspStatus           = 0x400A
	CmdRspSupportedFeature = 0x400D
	CmdRspANC              = 0x401E
	CmdRspEQ               = 0x401F
	CmdRspDualEnable       = 0x4027
	CmdRspDualDeviceList   = 0x4028
	CmdRspLag              = 0x4041
	CmdRspFirmware         = 0x4042
	CmdRspSpatial          = 0x404F
)

type Packet struct {
	Control  uint16
	Cmd      uint16
	Length   uint16
	FSN      byte
	Payload  []byte
	CRC      uint16
	CRCValid bool
	Raw      []byte
	FixedFSN bool // if true, MarshalBinary uses FSN as-is (tests / replay)
}

func (p Packet) HasCRC() bool { return p.Control&ControlCRC != 0 }

func (p Packet) MultiFrame() bool { return p.Control&ControlMultiFrame != 0 }

func (p Packet) RspCode() byte { return byte(p.Control & 0x1F) }

func (p Packet) DeviceType() int { return int((p.Control & 0x0F00) >> 8) }

func (p Packet) ResponseCmd() uint16 { return p.Cmd & 0x7FFF }

// ParserKey returns the catalog/parser map key for this packet.
// Event commands (0xExxx) keep full cmd; GET responses use 0x40xx; requests use request cmd.
func (p Packet) ParserKey() uint16 {
	if p.Cmd&0xF000 == 0xE000 {
		return p.Cmd
	}
	if p.Cmd&0xF000 == 0x4000 {
		return p.Cmd
	}
	if p.Cmd&0x8000 != 0 {
		return p.ResponseCmd()
	}
	return p.Cmd
}

var txFSN struct {
	mu  sync.Mutex
	val int
}

func NextFSN() byte {
	txFSN.mu.Lock()
	defer txFSN.mu.Unlock()
	txFSN.val++
	if txFSN.val >= 254 {
		txFSN.val = 0
	}
	return byte(txFSN.val)
}

func ResetFSN() {
	txFSN.mu.Lock()
	txFSN.val = 0
	txFSN.mu.Unlock()
}

func CRC16(data []byte) uint16 {
	crc := uint16(0xFFFF)
	for _, b := range data {
		crc ^= uint16(b)
		for i := 0; i < 8; i++ {
			if crc&1 != 0 {
				crc = (crc >> 1) ^ 0xA001
			} else {
				crc >>= 1
			}
		}
	}
	return crc
}

func putUint16LE(buf []byte, off int, v uint16) {
	binary.LittleEndian.PutUint16(buf[off:], v)
}

func getUint16LE(buf []byte, off int) uint16 {
	return binary.LittleEndian.Uint16(buf[off:])
}

func (p Packet) MarshalBinary() []byte {
	control := p.Control
	if control == 0 {
		control = ControlTXDefault
	}

	payload := p.Payload
	length := uint16(len(payload))
	fsn := p.FSN
	if !p.FixedFSN {
		fsn = NextFSN()
	}

	bodyLen := 8 + len(payload)
	if control&ControlCRC != 0 {
		bodyLen += 2
	}

	buf := make([]byte, bodyLen)
	buf[0] = SOF
	putUint16LE(buf, 1, control)
	putUint16LE(buf, 3, p.Cmd)
	putUint16LE(buf, 5, length)
	buf[7] = fsn
	if len(payload) > 0 {
		copy(buf[8:], payload)
	}

	if control&ControlCRC != 0 {
		crc := CRC16(buf[:8+len(payload)])
		putUint16LE(buf, 8+len(payload), crc)
	}

	return buf
}

// BuildFrame encodes a frame with explicit control/fsn (for tests).
func BuildFrame(control, cmd uint16, fsn byte, payload []byte) []byte {
	return Packet{
		Control:  control,
		Cmd:      cmd,
		Payload:  append([]byte(nil), payload...),
		FSN:      fsn,
		FixedFSN: true,
	}.MarshalBinary()
}
