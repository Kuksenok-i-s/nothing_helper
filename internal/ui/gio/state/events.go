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
			snap := s.session.Snapshot()
			s.mu.Lock()
			s.presenter.ApplyEvent(event, snap)
			if event.Kind == session.EventDisconnected {
				s.resetOnDisconnectLocked()
				s.clearDualPromptLocked()
			} else if eventUpdatesCommands(event) {
				s.commands = presenter.BuildCommands(snap.Model, snap.DualList, s.allowUnsafe)
				s.syncCmdButtons()
				s.evaluateDualPrompt(snap)
			}
			s.mu.Unlock()
			s.invalidate()
		}
	}
}

func eventUpdatesCommands(event session.Event) bool {
	switch event.Kind {
	case session.EventConnected, session.EventModel:
		return true
	default:
		return event.Parsed.DualList != nil
	}
}

// RunCommand enqueues a presenter command for async execution.
func (s *State) RunCommand(cmd presenter.Command) {
	s.enqueueCommand(cmd)
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
	s.RunCommand(presenter.Command{
		Title:  fmt.Sprintf("Dual: %s %s", action, name),
		Fields: []string{"dual", action, dev.MAC},
	})
}

// RefreshDualList requests the paired dual-connection device list.
func (s *State) RefreshDualList() {
	s.RunCommand(presenter.Command{
		Title:  "Dual: list",
		Fields: []string{"dual", "list"},
	})
}

func (s *State) setErr(msg string) {
	s.mu.Lock()
	changed := s.presenter.Err != msg
	s.presenter.Err = msg
	s.mu.Unlock()
	if changed {
		s.invalidate()
	}
}

func (s *State) SyncCommands() {
	s.mu.Lock()
	s.syncCmdButtons()
	s.mu.Unlock()
}

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
