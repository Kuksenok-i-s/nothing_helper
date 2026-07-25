package tui

import (
	"context"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"tws_manager/internal/bt"
	"tws_manager/internal/connect"
	"tws_manager/internal/session"
	"tws_manager/internal/spp"
	"tws_manager/internal/ui/dualprompt"
	"tws_manager/internal/ui/presenter"
)

func testModel(t *testing.T) Model {
	t.Helper()
	s := session.New(nil, false, false)
	m := New(s, Options{AllowUnsafe: false})
	m.presenter = presenter.NewState(false)
	return m
}

func TestUpdateCommentingEnterAndEsc(t *testing.T) {
	m := testModel(t)
	m.commenting = true
	m.comment.SetValue("note")

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := next.(Model)
	if got.commenting {
		t.Fatal("enter should leave commenting mode")
	}

	m = testModel(t)
	m.commenting = true
	m.comment.SetValue("note")
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	got = next.(Model)
	if got.commenting {
		t.Fatal("esc should leave commenting mode")
	}
	if got.comment.Value() != "" {
		t.Fatalf("esc should clear comment, got %q", got.comment.Value())
	}
}

func TestUpdateDevicesMsg(t *testing.T) {
	m := testModel(t)
	next, _ := m.Update(devicesMsg(nil))
	got := next.(Model)
	if got.presenter.Status == "" {
		t.Fatal("expected empty-device status")
	}

	next, _ = m.Update(devicesMsg{{MAC: "AA:BB:CC:DD:EE:FF", Name: "Ear"}})
	got = next.(Model)
	if len(got.devices) != 1 {
		t.Fatalf("devices=%d", len(got.devices))
	}
}

func TestHandleScanEnterRequiresUnsafe(t *testing.T) {
	m := testModel(t)
	m.activeTab = 1
	m.commands = list.New([]list.Item{commandItem{Command: presenter.Command{Title: "Scan", Fields: []string{"scan"}}}}, list.NewDefaultDelegate(), 20, 10)
	m.commands.Select(0)
	next, _ := m.handleScanEnter(presenter.Command{Title: "Scan"})
	got := next.(Model)
	if got.presenter.Err != "scan requires --unsafe" {
		t.Fatalf("err=%q", got.presenter.Err)
	}
}

func TestHandleScanEnterConfirmFlow(t *testing.T) {
	m := testModel(t)
	m.options.AllowUnsafe = true
	m.comment.SetValue("scan c001 c002 500ms")
	cmd := presenter.Command{Title: "Scan"}
	next, teaCmd := m.handleScanEnter(cmd)
	got := next.(Model)
	if got.pendingUnsafeItem == nil {
		t.Fatal("expected pending confirm")
	}
	if teaCmd != nil {
		t.Fatal("first enter should not start scan")
	}
	next, teaCmd = got.handleScanEnter(cmd)
	got = next.(Model)
	if got.pendingUnsafeItem != nil {
		t.Fatal("second enter should clear pending")
	}
	if teaCmd == nil {
		t.Fatal("second enter should return scan cmd")
	}
}

func TestHandleDualPromptKeys(t *testing.T) {
	m := testModel(t)
	m.dualPrompt = dualprompt.Controller{Visible: true, Mode: "ask"}
	if _, ok := m.handleDualPromptKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}); !ok {
		t.Fatal("y should be handled")
	}
	if _, ok := m.handleDualPromptKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}); ok {
		t.Fatal("x should not be handled")
	}
}

func TestUpdateDevicesTabEnterConnects(t *testing.T) {
	m := testModel(t)
	m.activeTab = 0
	m.devices = []bt.Device{{MAC: "AA:BB:CC:DD:EE:FF", Name: "Ear"}}
	next, cmd := m.handleEnter()
	got := next.(Model)
	if got.presenter.Status != "connecting Ear" {
		t.Fatalf("status=%q", got.presenter.Status)
	}
	if cmd == nil {
		t.Fatal("expected connect cmd")
	}
}

func TestErrMsgSetsPresenter(t *testing.T) {
	m := testModel(t)
	next, _ := m.Update(errMsg(errString("boom")))
	got := next.(Model)
	if got.presenter.Err != "boom" {
		t.Fatalf("err=%q", got.presenter.Err)
	}
}

func TestHandleKeyTabAndComment(t *testing.T) {
	m := testModel(t)
	next, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyTab})
	got := next.(Model)
	if got.activeTab != 1 {
		t.Fatalf("tab=%d want 1", got.activeTab)
	}

	next, _ = got.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	got = next.(Model)
	if !got.commenting {
		t.Fatal("c should enter commenting mode")
	}
}

func TestHandleKeyQuit(t *testing.T) {
	m := testModel(t)
	next, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("q should quit")
	}
	if _, ok := next.(Model); !ok {
		t.Fatal("expected model")
	}
}

func TestHandleKeyRefreshRequiresManager(t *testing.T) {
	m := testModel(t)
	m.manager = connect.New(m.session, connect.Options{RFCOMMPath: "/dev/rfcomm0", Channel: 15})
	next, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	got := next.(Model)
	if got.presenter.Status != "refreshing discovery..." {
		t.Fatalf("status=%q", got.presenter.Status)
	}
	if cmd == nil {
		t.Fatal("expected discover cmd")
	}
}

func TestHandleKeyExport(t *testing.T) {
	m := testModel(t)
	m.options.CaptureDir = t.TempDir()
	_, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if cmd == nil {
		t.Fatal("expected export cmd")
	}
}

func TestHandleKeyControlTab(t *testing.T) {
	m := testModel(t)
	m.activeTab = 1
	m.commands = list.New([]list.Item{commandItem{Command: presenter.Command{Title: "Battery"}}}, list.NewDefaultDelegate(), 20, 10)
	next, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	got := next.(Model)
	if got.activeTab != 1 {
		t.Fatalf("activeTab=%d", got.activeTab)
	}
	_ = cmd
}

func TestHandleKeyDualPromptDecline(t *testing.T) {
	m := testModel(t)
	m.dualPrompt.Visible = true
	m.dualPrompt.PendingOK = true
	_, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if cmd == nil {
		t.Fatal("expected wait event cmd")
	}
}

func TestHandleKeyDualPromptAccept(t *testing.T) {
	m := testModel(t)
	m.dualPrompt.Visible = true
	m.dualPrompt.PendingOK = true
	m.dualPrompt.HostMAC = "AA:BB:CC:DD:EE:FF"
	_, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil {
		t.Fatal("expected wait event cmd")
	}
}

func TestDeclineDualPCPrimary(t *testing.T) {
	m := testModel(t)
	m.dualPrompt.Visible = true
	m.declineDualPCPrimary()
	if m.dualPrompt.Visible {
		t.Fatal("decline should hide prompt")
	}
}

func TestHandleKeyCtrlC(t *testing.T) {
	m := testModel(t)
	_, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("ctrl+c should quit")
	}
}

func TestViewDualPromptLine(t *testing.T) {
	m := testModel(t)
	m.dualPrompt.Visible = true
	m.dualPrompt.PendingOK = true
	m.dualPrompt.Pending = spp.DualDevice{Name: "Phone", MAC: "11:22:33:44:55:66"}
	out := m.View()
	if !strings.Contains(out, "switch") || !strings.Contains(out, "not now") {
		t.Fatalf("dual prompt line missing: %s", out)
	}
}

func TestViewRendersTabsAndDevice(t *testing.T) {
	m := testModel(t)
	m.devices = []bt.Device{{MAC: "AA:BB:CC:DD:EE:FF", Name: "Ear", Connected: true}}
	m.presenter.Status = "ready"
	m.presenter.Err = "warn"
	m.activeTab = 0
	out := m.View()
	for _, want := range []string{"tws_manager", "ready", "warn", "Devices", "Ear", "AA:BB:CC:DD:EE:FF"} {
		if !strings.Contains(out, want) {
			t.Fatalf("View missing %q:\n%s", want, out)
		}
	}
}

func TestViewControlAndLogTabs(t *testing.T) {
	m := testModel(t)
	m.activeTab = 1
	out := m.View()
	if !strings.Contains(out, "[Control]") {
		t.Fatalf("control tab not highlighted: %s", out)
	}
	m.activeTab = 2
	m.commenting = true
	out = m.View()
	if !strings.Contains(out, "Log") {
		t.Fatalf("log tab missing: %s", out)
	}
}

func TestDevicesViewEmpty(t *testing.T) {
	m := testModel(t)
	if !strings.Contains(m.devicesView(), "No discovered devices") {
		t.Fatal("expected empty devices message")
	}
}

type errString string

func (e errString) Error() string { return string(e) }

func TestRunScanCmdReturnsErrorWhenDisconnected(t *testing.T) {
	s := session.New(nil, false, false)
	cmd := runScanCmd(context.Background(), s, []string{"scan", "c001", "c002", "500ms"})
	msg := cmd()
	if _, ok := msg.(errMsg); !ok {
		t.Fatalf("msg = %T, want errMsg", msg)
	}
}
