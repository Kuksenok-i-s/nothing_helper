//go:build gio

package state

import (
	"tws_manager/internal/dualpolicy"
	"tws_manager/internal/session"
	"tws_manager/internal/spp"
	"tws_manager/internal/ui/presenter"
)

func (s *State) ensureHostMAC() {
	if s.hostMAC != "" || s.hostMACLoaded {
		return
	}
	s.hostMACLoaded = true
	mac, err := dualpolicy.HostAdapterMAC()
	if err != nil {
		s.hostMACErr = err.Error()
		return
	}
	s.hostMAC = mac
}

func (s *State) evaluateDualPrompt(snap session.Snapshot) {
	if !snap.Connected {
		s.clearDualPromptLocked()
		return
	}
	s.ensureHostMAC()
	phone, ok := dualpolicy.ShouldPrompt(s.pcPrimaryMode, snap.Model, snap.DualList, s.hostMAC)
	if !ok {
		s.dualPendingOK = false
		s.dualPromptVisible = false
		if status := dualpolicy.HostOwnerStatus(snap.DualList, s.hostMAC); status != "" {
			s.presenter.Status = status
		}
		return
	}
	s.dualPending = phone
	s.dualPendingOK = true
	if s.userInteracted {
		s.dualPromptVisible = true
	}
}

func (s *State) clearDualPromptLocked() {
	s.dualPendingOK = false
	s.dualPromptVisible = false
	s.dualPending = spp.DualDevice{}
	s.userInteracted = false
}

// MarkUserInteraction records the first user action in the window for this
// connect session. A pending dual prompt becomes visible; after decline it is
// shown again on the next interaction.
func (s *State) MarkUserInteraction() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.userInteracted = true
	if s.dualPendingOK {
		s.dualPromptVisible = true
	}
}

// AcceptDualPCPrimary sends dual connect for the local adapter MAC.
func (s *State) AcceptDualPCPrimary() {
	s.mu.Lock()
	hostMAC := s.hostMAC
	s.dualPromptVisible = false
	s.mu.Unlock()
	if hostMAC == "" {
		s.setErr("host bluetooth MAC unavailable")
		return
	}
	s.RunCommand(presenter.Command{
		Title:  "Dual: switch to PC",
		Fields: []string{"dual", "connect", hostMAC},
	})
}

// DeclineDualPCPrimary hides the prompt until the next window interaction.
func (s *State) DeclineDualPCPrimary() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dualPromptVisible = false
}
