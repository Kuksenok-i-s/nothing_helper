package session

import (
	"time"

	"tws_manager/internal/trace"
)

func (s *Session) publish(event Event) {
	if s.logger != nil && event.Trace.Direction == "" {
		s.logger.LogEvent(trace.Event{
			Direction:     string(event.Kind),
			Source:        event.Source,
			Trigger:       event.Trigger,
			DeviceMAC:     event.Device.MAC,
			DeviceName:    event.Device.Name,
			ModelCodename: s.Snapshot().Model.Codename,
			Summary:       eventSummary(event),
			Error:         errorString(event.Error),
		})
	}
	select {
	case s.events <- event:
	default:
	}
	s.mu.Lock()
	subscribers := append([]chan Event(nil), s.subscribers...)
	s.mu.Unlock()
	for _, ch := range subscribers {
		if isPriorityEvent(event.Kind) {
			select {
			case ch <- event:
			case <-time.After(2 * time.Second):
				// Subscriber slow; drop after brief wait rather than block forever.
			}
			continue
		}
		select {
		case ch <- event:
		default:
		}
	}
}

func isPriorityEvent(kind EventKind) bool {
	switch kind {
	case EventBattery, EventConnected, EventDisconnected:
		return true
	default:
		return false
	}
}

func eventSummary(event Event) string {
	if event.Trigger != "" {
		return event.Trigger
	}
	if event.Error != nil {
		return event.Error.Error()
	}
	return string(event.Kind)
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
