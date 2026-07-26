package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"tws_manager/internal/bt"
	"tws_manager/internal/session"
	"tws_manager/internal/ui/presenter"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.commenting {
		return m.updateCommenting(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	case devicesMsg:
		return m.updateDevices(msg)
	case eventMsg:
		m.applyEvent(session.Event(msg))
		return m, waitEventCmd(m.events)
	case errMsg:
		m.presenter.Err = error(msg).Error()
	}

	return m.forwardToCommands(msg)
}

func (m Model) updateCommenting(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "enter":
			m.commenting = false
			return m, nil
		case "esc":
			m.commenting = false
			m.comment.SetValue("")
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.comment, cmd = m.comment.Update(msg)
	return m, cmd
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.markUserInteraction()
	if m.dualPrompt.Visible {
		if cmd, handled := m.handleDualPromptKey(msg); handled {
			return m, cmd
		}
	}
	return m.handleNormalKey(msg)
}

func (m Model) handleNormalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		// Do not Close here — RFCOMM teardown can block. Run's defer and
		// app.Shutdown close the session after the UI exits.
		return m, tea.Quit
	case "tab":
		m.activeTab = (m.activeTab + 1) % 3
	case "r":
		m.presenter.Status = "refreshing discovery..."
		return m, discoverCmd(m.manager)
	case "c":
		m.commenting = true
		m.comment.Focus()
	case "s":
		return m, m.exportCmd()
	case "enter":
		return m.handleEnter()
	}
	return m.forwardToCommands(msg)
}

func (m Model) handleDualPromptKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	switch msg.String() {
	case "y", "Y":
		m.acceptDualPCPrimary()
		return waitEventCmd(m.events), true
	case "n", "N":
		m.declineDualPCPrimary()
		return waitEventCmd(m.events), true
	default:
		return nil, false
	}
}

func (m Model) handleEnter() (tea.Model, tea.Cmd) {
	if m.activeTab == 0 && len(m.devices) > 0 {
		dev := m.devices[0]
		m.presenter.Status = "connecting " + dev.Name
		return m, connectCmd(m.manager, dev)
	}
	if m.activeTab == 1 {
		return m.handleEnterCommand()
	}
	return m, nil
}

func (m Model) handleEnterCommand() (tea.Model, tea.Cmd) {
	item, ok := m.commands.SelectedItem().(commandItem)
	if !ok {
		return m, nil
	}
	if presenter.IsScanCommand(item.Command) {
		return m.handleScanEnter(item.Command)
	}
	// All catalog commands (GET and SET presets) are safe by
	// construction; only the raw scan above needs gating.
	return m, sendCommandCmd(m.session, item.Command, m.comment.Value())
}

func (m Model) handleScanEnter(cmd presenter.Command) (tea.Model, tea.Cmd) {
	if !m.options.AllowUnsafe {
		m.presenter.Err = "scan requires --unsafe"
		return m, nil
	}
	fields := strings.Fields(strings.TrimSpace(m.comment.Value()))
	if len(fields) != 4 || !strings.EqualFold(fields[0], "scan") {
		m.presenter.Err = "comment scan: scan c001 c020 500ms (GET 0xC0xx only, delay >= 200ms, max 32 cmds)"
		m.presenter.Status = "Enter scan range in comment (c), then confirm with Enter"
		return m, nil
	}
	if m.pendingUnsafeItem == nil || m.pendingUnsafeItem.Title != cmd.Title {
		m.pendingUnsafeItem = &cmd
		m.presenter.Status = fmt.Sprintf("Confirm scan %s %s %s: press Enter again", fields[1], fields[2], fields[3])
		m.presenter.Err = ""
		return m, nil
	}
	m.pendingUnsafeItem = nil
	return m, runScanCmd(m.options.Ctx, m.session, fields)
}

func (m Model) updateDevices(msg devicesMsg) (tea.Model, tea.Cmd) {
	m.devices = []bt.Device(msg)
	if len(m.devices) == 0 {
		m.presenter.Status = "no compatible TWS devices found; pass --addr for manual connection"
	} else {
		m.presenter.Status = fmt.Sprintf("found %d candidate(s); Enter connects first", len(m.devices))
	}
	return m, nil
}

func (m Model) forwardToCommands(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if m.activeTab == 1 {
		m.commands, cmd = m.commands.Update(msg)
	}
	return m, cmd
}
