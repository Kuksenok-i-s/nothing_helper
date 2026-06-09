package tui

import (
	"tws_manager/internal/session"
	"tws_manager/internal/ui/actions"
	"tws_manager/internal/ui/presenter"
)

func (m *Model) evaluateDualPrompt(snap session.Snapshot) {
	if status := m.dualPrompt.OnSnapshot(snap); status != "" {
		m.presenter.Status = status
	}
}

func (m *Model) markUserInteraction() {
	m.dualPrompt.OnInteraction()
}

func (m *Model) acceptDualPCPrimary() {
	fields, err := m.dualPrompt.AcceptFields()
	if err != nil {
		m.presenter.Err = err.Error()
		return
	}
	result := actions.Execute(m.session, presenter.Command{
		Title:  "Dual: switch to PC",
		Fields: fields,
	}, actions.ExecOpts{Source: "tui"})
	if result.Err != nil {
		m.presenter.Err = result.Err.Error()
		return
	}
	m.presenter.Status = "dual: switching to PC"
	m.presenter.Err = ""
}

func (m *Model) declineDualPCPrimary() {
	m.dualPrompt.Decline()
}

func (m *Model) dualPromptLine() string {
	if line := m.dualPrompt.PromptLine(); line != "" {
		return line + "  [y] switch  [n] not now"
	}
	return ""
}
