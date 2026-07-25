package session

import (
	"errors"
	"testing"
)

func TestEventSummary(t *testing.T) {
	if got := eventSummary(Event{Trigger: "opening"}); got != "opening" {
		t.Fatalf("summary=%q", got)
	}
	if got := eventSummary(Event{Error: errors.New("boom")}); got != "boom" {
		t.Fatalf("summary=%q", got)
	}
	if got := eventSummary(Event{Kind: EventProgress}); got != string(EventProgress) {
		t.Fatalf("summary=%q", got)
	}
}

func TestPublishNotifiesSubscribers(t *testing.T) {
	s := New(nil, false, false)
	ch := s.Subscribe()
	s.publish(Event{Kind: EventProgress, Trigger: "ready"})
	select {
	case ev := <-ch:
		if ev.Trigger != "ready" {
			t.Fatalf("event=%+v", ev)
		}
	default:
		t.Fatal("expected event on subscriber channel")
	}
}

func TestPublishPriorityEvent(t *testing.T) {
	s := New(nil, false, false)
	ch := make(chan Event, 1)
	ch <- Event{Kind: EventProgress, Trigger: "stale"}
	s.mu.Lock()
	s.subscribers = append(s.subscribers, ch)
	s.mu.Unlock()

	s.publish(Event{Kind: EventBattery, Trigger: "battery"})
	select {
	case ev := <-ch:
		if ev.Kind != EventBattery {
			t.Fatalf("event=%+v", ev)
		}
	default:
		t.Fatal("expected priority battery event")
	}
}
