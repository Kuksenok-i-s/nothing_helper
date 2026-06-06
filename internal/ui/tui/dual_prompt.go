package tui

import (
	"tws_manager/internal/dualpolicy"
	"tws_manager/internal/session"
)

func (m *Model) ensureHostMAC() {
	if m.hostMAC != "" || m.hostMACLoaded {
		return
	}
	m.hostMACLoaded = true
	mac, err := dualpolicy.HostAdapterMAC()
	if err != nil {
		m.hostMACErr = err.Error()
		return
	}
	m.hostMAC = mac
}

func (m *Model) evaluateDualPrompt(snap session.Snapshot) {
	if !snap.Connected {
		m.dualPendingOK = false
		m.dualPromptVisible = false
		m.userInteracted = false
		return
	}
	m.ensureHostMAC()
	phone, ok := dualpolicy.ShouldPrompt(m.pcPrimaryMode, snap.Model, snap.DualList, m.hostMAC)
	if !ok {
		m.dualPendingOK = false
		m.dualPromptVisible = false
		if status := dualpolicy.HostOwnerStatus(snap.DualList, m.hostMAC); status != "" {
			m.presenter.Status = status
		}
		return
	}
	m.dualPending = phone
	m.dualPendingOK = true
	if m.userInteracted {
		m.dualPromptVisible = true
	}
}

func (m *Model) markUserInteraction() {
	m.userInteracted = true
	if m.dualPendingOK {
		m.dualPromptVisible = true
	}
}

func (m *Model) acceptDualPCPrimary() {
	m.dualPromptVisible = false
	if m.hostMAC == "" {
		m.presenter.Err = "host bluetooth MAC unavailable"
		return
	}
	fields := []string{"dual", "connect", m.hostMAC}
	pkt, _, err := m.session.FeaturePacket(fields)
	if err != nil {
		m.presenter.Err = err.Error()
		return
	}
	if err := m.session.Send(pkt, session.Meta{Source: "tui", Trigger: "Dual: switch to PC"}); err != nil {
		m.presenter.Err = err.Error()
		return
	}
	m.presenter.Status = "dual: switching to PC"
	m.presenter.Err = ""
}

func (m *Model) declineDualPCPrimary() {
	m.dualPromptVisible = false
}

func (m *Model) dualPromptLine() string {
	if !m.dualPromptVisible || !m.dualPendingOK {
		return ""
	}
	return dualpolicy.PromptText(m.dualPending) + "  [y] switch  [n] not now"
}
