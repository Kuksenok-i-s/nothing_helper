//go:build gio

package state

import (
	"context"
	"fmt"
	"strings"
	"time"

	"tws_manager/internal/session"
	"tws_manager/internal/spp"
	"tws_manager/internal/ui/presenter"
)

// PumpEvents subscribes to session events and updates presenter state.
func (s *State) PumpEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-s.events:
			if !ok {
				return
			}
			s.mu.Lock()
			snap := s.session.Snapshot()
			s.presenter.ApplyEvent(event, snap)
			if event.Kind == session.EventDisconnected {
				s.resetOnDisconnectLocked()
				s.clearDualPromptLocked()
			} else {
				s.commands = presenter.BuildCommands(snap.Model, snap.DualList, s.allowUnsafe)
				s.syncCmdButtons()
				s.evaluateDualPrompt(snap)
			}
			s.mu.Unlock()
			s.invalidate()
		}
	}
}

// RunCommand sends a presenter command through the session.
func (s *State) RunCommand(cmd presenter.Command) {
	if len(cmd.Fields) > 0 {
		pkt, _, err := s.session.FeaturePacket(cmd.Fields)
		if err != nil {
			s.setErr(err.Error())
			return
		}
		if err := s.session.Send(pkt, session.Meta{Source: "gio", Trigger: cmd.Title}); err != nil {
			s.setErr(err.Error())
			return
		}
		s.refreshFeatureAfterSet(cmd.Fields)
		return
	}
	if err := s.session.SendCommand(cmd.Cmd, session.Meta{Source: "gio", Trigger: cmd.Title}); err != nil {
		s.setErr(err.Error())
	}
}

// RunToggle sends the on/off SET for a feature switch and then re-reads the
// state so the switch settles to the value the device actually reports.
func (s *State) RunToggle(tf presenter.ToggleFeature, on bool) {
	fields := tf.OffFields
	if on {
		fields = tf.OnFields
	}
	s.MarkTogglePending(tf.Feature, on)
	go func() {
		s.RunCommand(presenter.Command{Title: tf.Label + " set", Fields: fields})
		time.Sleep(350 * time.Millisecond)
		s.RunCommand(presenter.Command{Title: tf.Label + " get", Fields: []string{tf.Feature, "get"}})
	}()
}

// RunDualAction connects or disconnects a dual-connection peer device.
func (s *State) RunDualAction(dev spp.DualDevice) {
	action := "connect"
	if dev.Connected {
		action = "disconnect"
	}
	name := dev.Name
	if name == "" {
		name = dev.MAC
	}
	go s.RunCommand(presenter.Command{
		Title:  fmt.Sprintf("Dual: %s %s", action, name),
		Fields: []string{"dual", action, dev.MAC},
	})
}

// RefreshDualList requests the paired dual-connection device list.
func (s *State) RefreshDualList() {
	go s.RunCommand(presenter.Command{
		Title:  "Dual: list",
		Fields: []string{"dual", "list"},
	})
}

func (s *State) setErr(msg string) {
	s.mu.Lock()
	s.presenter.Err = msg
	s.mu.Unlock()
}

func (s *State) SyncCommands() { s.syncCmdButtons() }

// refreshFeatureAfterSet re-reads a feature after a SET so the status panel updates.
func (s *State) refreshFeatureAfterSet(fields []string) {
	if len(fields) < 2 || !strings.EqualFold(fields[1], "set") {
		return
	}
	feature := strings.ToLower(fields[0])
	switch feature {
	case "anc", "eq", "spatial", "lag", "dual":
	default:
		return
	}
	go func() {
		time.Sleep(350 * time.Millisecond)
		s.RunCommand(presenter.Command{
			Title:  feature + " get",
			Fields: []string{feature, "get"},
		})
	}()
}

// resetOnDisconnectLocked clears feature toggles and rebuilds the command list
// for an offline session. Caller must hold s.mu.
func (s *State) resetOnDisconnectLocked() {
	s.toggleBools = nil
	s.togglePending = nil
	s.toggleWant = nil
	s.presenter.Err = ""
	s.commands = presenter.BuildCommands(spp.DefaultModel(), nil, s.allowUnsafe)
	s.syncCmdButtons()
}
