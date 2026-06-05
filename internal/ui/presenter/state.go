package presenter

import (
	"fmt"
	"strings"
	"time"

	"tws_manager/internal/bt"
	"tws_manager/internal/session"
	"tws_manager/internal/spp"
	"tws_manager/internal/trace"
)

// State holds UI-agnostic presentation state derived from session events.
type State struct {
	LogLines      []string
	LastEvents    []trace.Event
	Status        string
	Err           string
	LogRaw        bool
	AutoReconnect bool
}

const (
	maxLogLines   = 300
	maxLastEvents = 200
)

// NewState creates an empty presenter state.
func NewState(logRaw bool) *State {
	return &State{LogRaw: logRaw}
}

// ApplyEvent updates status, log lines, and export buffer from a session event.
func (s *State) ApplyEvent(event session.Event, snap session.Snapshot) {
	line := time.Now().Format("15:04:05") + " " + string(event.Kind)
	if event.Parsed.Summary != "" {
		line += ": " + event.Parsed.Summary
	} else if event.Error != nil {
		line += ": " + event.Error.Error()
	} else if event.Trace.Summary != "" {
		line += ": " + event.Trace.Summary
	} else if event.Trigger != "" {
		line += ": " + event.Trigger
	}
	if event.Source != "" {
		line = fmt.Sprintf("[%s] %s", event.Source, line)
	}
	if s.LogRaw && len(event.Raw) > 0 {
		line += fmt.Sprintf(" raw=% x", event.Raw)
	}
	s.LogLines = append(s.LogLines, line)
	if len(s.LogLines) > maxLogLines {
		s.LogLines = s.LogLines[len(s.LogLines)-maxLogLines:]
	}
	if event.Trace.Direction != "" {
		tr := event.Trace
		if tr.Time == "" {
			tr.Time = time.Now().Format("15:04:05")
		}
		s.LastEvents = append(s.LastEvents, tr)
		if len(s.LastEvents) > maxLastEvents {
			s.LastEvents = s.LastEvents[len(s.LastEvents)-maxLastEvents:]
		}
	}
	switch event.Kind {
	case session.EventProgress:
		s.Status = event.Trigger
	case session.EventConnected:
		s.Err = ""
		if event.Device.Name != "" {
			s.Status = "connected to " + event.Device.Name
		} else {
			s.Status = "connected"
		}
	case session.EventPacketTX:
		s.Status = "sent " + spp.CommandLabel(event.Packet.Cmd)
	case session.EventPacketRX:
		s.Status = "received " + spp.CommandLabel(event.Packet.Cmd)
	case session.EventBattery:
		s.Status = "battery updated"
	case session.EventModel:
		model := snap.Model.Codename
		if model != "" {
			s.Status = "model detected: " + model
		} else {
			s.Status = event.Trigger
		}
	case session.EventError:
		if event.Error != nil {
			s.Err = event.Error.Error()
			s.Status = "error"
		}
	case session.EventDisconnected:
		s.Status = disconnectStatus(event.Device, s.AutoReconnect)
	}
}

func disconnectStatus(device bt.Device, autoReconnect bool) string {
	name := device.Name
	if name == "" {
		name = device.MAC
	}
	if autoReconnect {
		if name != "" {
			return fmt.Sprintf("disconnected from %s - reconnecting…", name)
		}
		return "disconnected - reconnecting…"
	}
	if name != "" {
		return fmt.Sprintf("disconnected from %s - tap Auto-connect", name)
	}
	return "disconnected - tap Auto-connect"
}

// LogText returns joined log lines for display.
func (s *State) LogText() string {
	return strings.Join(s.LogLines, "\n")
}

// FormatBatteries formats battery snapshot for display.
func FormatBatteries(data map[string]spp.Battery) string {
	if len(data) == 0 {
		return "n/a"
	}
	parts := make([]string, 0, len(data))
	for _, name := range []string{"left", "right", "case", "stereo"} {
		if b, ok := data[name]; ok {
			suffix := ""
			if b.Charging {
				suffix = "+"
			}
			parts = append(parts, fmt.Sprintf("%s=%d%%%s", name, b.Percent, suffix))
		}
	}
	return strings.Join(parts, " ")
}
