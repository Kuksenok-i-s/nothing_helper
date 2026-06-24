//go:build gio

package state

import (
	"fmt"
	"path/filepath"
	"time"

	"gioui.org/widget"

	"tws_manager/internal/bt"
	"tws_manager/internal/connect"
	"tws_manager/internal/session"
	"tws_manager/internal/spp"
	"tws_manager/internal/trace"
)

// Handle dispatches UI actions to async operations.
func (s *State) Handle(actions ...Action) {
	for _, action := range actions {
		switch action {
		case ActionDiscover:
			s.doDiscover()
		case ActionBind:
			s.doBind()
		case ActionConnect:
			s.doConnect()
		case ActionDisconnect:
			s.doDisconnect()
		case ActionBattery:
			s.doBattery()
		case ActionExport:
			s.doExport()
		case ActionAuto:
			s.doAuto()
		}
	}
	s.invalidate()
}

// runAutoReconnectLoop repeatedly discovers and connects while the link is down.
func (s *State) runAutoReconnectLoop() {
	s.manager.AutoConnect(s.ctx, connect.AutoOptions{
		Interval: 10 * time.Second,
		OnStatus: s.SetStatus,
	})
}

// doAuto searches for available headphones and connects to the best candidate
// in one click (discover → bind → connect), updating the device list as it goes.
func (s *State) doAuto() {
	go func() {
		setStatus := func(msg string) {
			s.mu.Lock()
			s.presenter.Status = msg
			s.presenter.Err = ""
			s.mu.Unlock()
			s.invalidate()
		}
		setStatus("auto: scanning for TWS devices…")

		if devs, err := s.manager.Discover(s.ctx); err == nil {
			s.mu.Lock()
			s.devices = devs
			s.devClicks = make([]widget.Clickable, len(devs))
			if s.selectedDev >= len(devs) {
				s.selectedDev = 0
			}
			s.mu.Unlock()
			s.invalidate()
		}

		err := s.manager.ConnectBest(s.ctx, setStatus)
		s.mu.Lock()
		if err != nil {
			s.presenter.Err = err.Error()
		} else {
			s.presenter.Status = "auto: connected"
			s.presenter.Err = ""
		}
		s.mu.Unlock()
		s.invalidate()
	}()
}

func (s *State) connectInitial(dev bt.Device) {
	name := dev.Name
	if name == "" {
		name = dev.MAC
	}
	s.SetStatus("connecting to " + name + "...")
	if exists, _ := s.manager.RFCOMMExists(); !exists && dev.MAC != "" {
		if err := s.manager.Bind(s.ctx, dev); err != nil {
			if s.ctx.Err() == nil {
				s.setErr("bind: " + err.Error())
			}
			return
		}
	}
	if err := s.manager.Connect(s.ctx, dev); err != nil {
		if s.ctx.Err() == nil {
			s.setErr("connect: " + err.Error())
		}
		return
	}
	s.SetStatus("connected to " + name)
}

// ConnectChosen connects to the discovered device at index i, rebinding RFCOMM
// if needed. Used by the device picker on the Control screen.
func (s *State) ConnectChosen(i int) {
	s.mu.Lock()
	if i < 0 || i >= len(s.devices) {
		s.mu.Unlock()
		return
	}
	dev := s.devices[i]
	s.selectedDev = i
	s.mu.Unlock()
	go func() {
		name := dev.Name
		if name == "" {
			name = dev.MAC
		}
		s.SetStatus("connecting to " + name + "…")
		if err := s.manager.SwitchTo(s.ctx, dev); err != nil {
			if s.ctx.Err() == nil {
				s.setErr(err.Error())
			}
			return
		}
		s.SetStatus("connected to " + name)
	}()
}

func (s *State) doDiscover() {
	go func() {
		devs, err := s.manager.Discover(s.ctx)
		s.mu.Lock()
		if err != nil {
			s.presenter.Err = err.Error()
		} else {
			s.devices = devs
			s.devClicks = make([]widget.Clickable, len(devs))
			if s.selectedDev >= len(devs) {
				s.selectedDev = 0
			}
			s.presenter.Status = fmt.Sprintf("found %d device(s)", len(devs))
			s.presenter.Err = ""
		}
		s.mu.Unlock()
		s.invalidate()
	}()
}

func (s *State) doBind() {
	dev := s.selectedDevice()
	if dev.MAC == "" {
		s.mu.Lock()
		s.presenter.Status = "select a device with MAC first"
		s.mu.Unlock()
		s.invalidate()
		return
	}
	go func() {
		err := s.manager.Bind(s.ctx, dev)
		s.mu.Lock()
		if err != nil {
			s.presenter.Err = err.Error()
		} else {
			s.presenter.Status = fmt.Sprintf("bound %s to %s", dev.MAC, s.manager.Options().RFCOMMPath)
			s.presenter.Err = ""
		}
		s.mu.Unlock()
		s.invalidate()
	}()
}

func (s *State) doConnect() {
	dev := s.selectedDevice()
	go func() {
		err := s.manager.Connect(s.ctx, dev)
		s.mu.Lock()
		if err != nil {
			s.presenter.Err = err.Error()
		} else {
			s.presenter.Status = "connect requested"
			s.presenter.Err = ""
		}
		s.mu.Unlock()
		s.invalidate()
	}()
}

func (s *State) doDisconnect() {
	go func() {
		err := s.manager.Disconnect()
		s.mu.Lock()
		if err != nil {
			s.presenter.Err = err.Error()
		} else {
			s.presenter.Status = "disconnected"
			s.presenter.Err = ""
		}
		s.mu.Unlock()
		s.invalidate()
	}()
}

func (s *State) doBattery() {
	go func() {
		err := s.session.SendCommand(spp.CmdGetBattery, session.Meta{Source: "gio", Trigger: "battery"})
		s.mu.Lock()
		if err != nil {
			s.presenter.Err = err.Error()
		}
		s.mu.Unlock()
		s.invalidate()
	}()
}

func (s *State) doExport() {
	go func() {
		s.mu.Lock()
		events := append([]trace.Event(nil), s.presenter.LastEvents...)
		dir := s.captureDir
		comment := s.comment
		// Raw inclusion is controlled live from the UI toggle, not a CLI flag.
		logRaw := s.RawHexToggle.Value
		s.mu.Unlock()
		if dir == "" {
			dir = "captures"
		}
		path := filepath.Join(dir, time.Now().Format("2006-01-02_15-04-05")+"_packets.json")
		s.mu.Lock()
		if len(events) == 0 {
			s.presenter.Err = "no packets to export"
		} else if err := trace.Export(path, events, comment, logRaw); err != nil {
			s.presenter.Err = err.Error()
		} else {
			s.presenter.Status = "exported " + path
			s.presenter.Err = ""
		}
		s.mu.Unlock()
		s.invalidate()
	}()
}

func (s *State) selectedDevice() bt.Device {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.devices) == 0 {
		return s.manager.DeviceForExistingRFCOMM("")
	}
	idx := s.selectedDev
	if idx < 0 || idx >= len(s.devices) {
		idx = 0
	}
	return s.devices[idx]
}
