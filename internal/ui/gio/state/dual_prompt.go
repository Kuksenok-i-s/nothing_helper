//go:build gio

package state

import (
	"tws_manager/internal/dualpolicy"
	"tws_manager/internal/session"
	"tws_manager/internal/ui/presenter"
)

func (s *State) preloadHostMAC() {
	mac, err := dualpolicy.HostAdapterMAC()
	s.mu.Lock()
	s.dualPrompt.HostLoaded = true
	if err != nil {
		s.dualPrompt.HostErr = err.Error()
	} else {
		s.dualPrompt.HostMAC = mac
	}
	s.mu.Unlock()
}

func (s *State) evaluateDualPrompt(snap session.Snapshot) {
	if status := s.dualPrompt.OnSnapshot(snap); status != "" {
		s.presenter.Status = status
	}
}

func (s *State) clearDualPromptLocked() {
	s.dualPrompt.OnDisconnected()
}

func (s *State) MarkUserInteraction() {
	s.mu.Lock()
	wasVisible := s.dualPrompt.Visible
	s.dualPrompt.OnInteraction()
	changed := s.dualPrompt.Visible != wasVisible
	s.mu.Unlock()
	if changed {
		s.invalidate()
	}
}

func (s *State) AcceptDualPCPrimary() {
	s.mu.Lock()
	wasVisible := s.dualPrompt.Visible
	fields, err := s.dualPrompt.AcceptFields()
	changed := s.dualPrompt.Visible != wasVisible
	s.mu.Unlock()
	if err != nil {
		s.setErr(err.Error())
		return
	}
	if changed {
		s.invalidate()
	}
	s.enqueueCommand(presenter.Command{Title: "Dual: switch to PC", Fields: fields})
}

func (s *State) DeclineDualPCPrimary() {
	s.mu.Lock()
	wasVisible := s.dualPrompt.Visible
	s.dualPrompt.Decline()
	changed := s.dualPrompt.Visible != wasVisible
	s.mu.Unlock()
	if changed {
		s.invalidate()
	}
}
