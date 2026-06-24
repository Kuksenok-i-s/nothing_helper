//go:build gio

package state

import (
	"time"

	"tws_manager/internal/session"
	"tws_manager/internal/spp"
)

const (
	guiDeviceCheckInterval = time.Second
	guiStatusRequestEvery  = 10 * time.Second
)

func (s *State) pollDeviceStatus() {
	var lastRequest time.Time
	for {
		if !s.guiActive() {
			if !s.waitForGUIActivityChange() {
				return
			}
			continue
		}

		ticker := time.NewTicker(guiDeviceCheckInterval)
		active := true
		for active {
			select {
			case <-s.ctx.Done():
				ticker.Stop()
				return
			case <-s.windowChanged:
				active = s.guiActive()
			case now := <-ticker.C:
				snap := s.session.Snapshot()
				if !statusPollDue(now, lastRequest, true, snap.Connected) {
					continue
				}
				lastRequest = now
				_ = s.session.SendCommand(spp.CmdGetStatus, session.Meta{
					Source:  "gio_status",
					Trigger: "status refresh",
				})
			}
		}
		ticker.Stop()
	}
}

func (s *State) guiActive() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.window != nil
}

func statusPollDue(now, lastRequest time.Time, active, connected bool) bool {
	if !active || !connected {
		return false
	}
	return lastRequest.IsZero() || now.Sub(lastRequest) >= guiStatusRequestEvery
}

func (s *State) waitForGUIActivityChange() bool {
	if s.windowChanged == nil {
		<-s.ctx.Done()
		return false
	}
	select {
	case <-s.ctx.Done():
		return false
	case <-s.windowChanged:
		return true
	}
}
