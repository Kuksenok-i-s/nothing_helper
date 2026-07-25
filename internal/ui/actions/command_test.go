package actions

import (
	"context"
	"os"
	"testing"

	"tws_manager/internal/bt"
	"tws_manager/internal/session"
	"tws_manager/internal/spp"
	"tws_manager/internal/trace"
	"tws_manager/internal/ui/presenter"
)

func TestExecuteRejectsScanCommand(t *testing.T) {
	res := Execute(nil, presenter.Command{Title: "Advanced: raw scan", Advanced: true}, ExecOpts{})
	if res.Err == nil {
		t.Fatal("scan command must be rejected by Execute")
	}
}

func TestExecuteSendCommand(t *testing.T) {
	spp.ResetFSN()
	s := session.New(nil, false, false)
	f, err := os.CreateTemp(t.TempDir(), "rfcomm")
	if err != nil {
		t.Fatal(err)
	}
	dev := bt.Device{MAC: "AA:BB:CC:DD:EE:FF", Name: "Ear"}
	s.AttachTestLink(bt.NewTestTransport(f, dev.MAC, 15, "/dev/rfcomm0"), dev)

	res := Execute(s, presenter.Command{Title: "Info: battery", Cmd: spp.CmdGetBattery}, ExecOpts{Source: "test", Comment: "note"})
	if res.Err != nil {
		t.Fatalf("Execute() = %v", res.Err)
	}
}

func TestExecuteSendCommandNotConnected(t *testing.T) {
	res := Execute(session.New(nil, false, false), presenter.Command{Title: "Info: battery", Cmd: spp.CmdGetBattery}, ExecOpts{})
	if res.Err == nil {
		t.Fatal("expected not connected error")
	}
}

func TestExecuteFeatureFields(t *testing.T) {
	spp.ResetFSN()
	s := session.New(nil, false, false)
	f, err := os.CreateTemp(t.TempDir(), "rfcomm")
	if err != nil {
		t.Fatal(err)
	}
	dev := bt.Device{MAC: "AA:BB:CC:DD:EE:FF", Name: "Ear"}
	s.AttachTestLink(bt.NewTestTransport(f, dev.MAC, 15, "/dev/rfcomm0"), dev)

	res := Execute(s, presenter.Command{Title: "ANC: get", Fields: []string{"anc", "get"}}, ExecOpts{Source: "tui"})
	if res.Err != nil {
		t.Fatalf("Execute() = %v", res.Err)
	}
}

func TestExecuteFeaturePacketError(t *testing.T) {
	res := Execute(session.New(nil, false, false), presenter.Command{Fields: []string{"not-a-command"}}, ExecOpts{})
	if res.Err == nil {
		t.Fatal("expected feature packet error")
	}
}

func TestExecuteDefaultSource(t *testing.T) {
	spp.ResetFSN()
	s := session.New(nil, false, false)
	f, err := os.CreateTemp(t.TempDir(), "rfcomm")
	if err != nil {
		t.Fatal(err)
	}
	dev := bt.Device{MAC: "AA:BB:CC:DD:EE:FF", Name: "Ear"}
	s.AttachTestLink(bt.NewTestTransport(f, dev.MAC, 15, "/dev/rfcomm0"), dev)

	res := Execute(s, presenter.Command{Title: "Info: battery", Cmd: spp.CmdGetBattery}, ExecOpts{})
	if res.Err != nil {
		t.Fatalf("Execute() = %v", res.Err)
	}
}

func TestExecuteScan(t *testing.T) {
	spp.ResetFSN()
	s := session.New(nil, false, false)
	f, err := os.CreateTemp(t.TempDir(), "rfcomm")
	if err != nil {
		t.Fatal(err)
	}
	dev := bt.Device{MAC: "AA:BB:CC:DD:EE:FF", Name: "Ear"}
	s.AttachTestLink(bt.NewTestTransport(f, dev.MAC, 15, "/dev/rfcomm0"), dev)

	err = ExecuteScan(context.Background(), s, []string{"scan", "c001", "c001", "200ms"})
	if err != nil {
		t.Fatalf("ExecuteScan() = %v", err)
	}
}

func TestExportPacketsEmpty(t *testing.T) {
	if err := ExportPackets("out.json", nil, "", false); err == nil {
		t.Fatal("expected empty export error")
	}
}

func TestExportPacketsWrites(t *testing.T) {
	path := t.TempDir() + "/out.json"
	events := []trace.Event{{Direction: "tx", Summary: "sent"}}
	if err := ExportPackets(path, events, "note", false); err != nil {
		t.Fatalf("ExportPackets() = %v", err)
	}
}
