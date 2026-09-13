package session

import (
	"testing"
	"nothing_helper/internal/spp"
)

func TestMicWriteGuardsAndReadback(t *testing.T) {
	for _, unsafe := range []bool{false, true} {
		for _, cmd := range []uint16{spp.CmdSetSuperMic, spp.CmdSetWalkieTalkie} {
			s, w := findSession(t)
			s.allowUnsafe = unsafe
			for _, payload := range [][]byte{nil, {2}, {1, 0}} {
				if err := s.Send(spp.Packet{Cmd: cmd, Payload: payload}, Meta{}); err == nil {
					t.Fatal("malformed write allowed")
				}
			}
			s.model, _ = spp.ResolveModelInfo("EarTwo")
			if err := s.Send(spp.Packet{Cmd: cmd, Payload: []byte{1}}, Meta{}); err == nil {
				t.Fatal("unsupported model allowed")
			}
			if len(w.packets) != 0 {
				t.Fatal("blocked command reached wire")
			}
			s.model, _ = spp.ResolveModelInfo("EarThree")
			s.config["walkie-talkie"] = "enabled=false"
			s.config["super-mic"] = "enabled=false"
			if err := s.Send(spp.Packet{Cmd: cmd, Payload: []byte{1}}, Meta{}); err != nil {
				t.Fatal(err)
			}
			key, rsp := "walkie-talkie", uint16(spp.CmdRspWalkieTalkie)
			if cmd == spp.CmdSetSuperMic {
				key, rsp = "super-mic", spp.CmdRspSuperMic
			}
			if _, ok := s.Snapshot().Config[key]; ok {
				t.Fatal("old state retained during write")
			}
			s.recordConfig(spp.ParsePacket(spp.Packet{Cmd: rsp, Payload: []byte{1}}, s.model))
			if s.Snapshot().Config[key] != "enabled=true" {
				t.Fatal("response not recorded")
			}
			s.recordConfig(spp.ParsePacket(spp.Packet{Cmd: rsp, Payload: []byte{2}}, s.model))
			if _, ok := s.Snapshot().Config[key]; ok {
				t.Fatal("invalid response retained state")
			}
		}
	}
}
