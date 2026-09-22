package companion

import (
	"reflect"
	"sync"
	"time"

	"nothing_helper/internal/session"
)

// redrawState compares displayed values, excluding background protocol traffic.
// The event listener and window loop access it from different goroutines.
type redrawState struct {
	mu   sync.Mutex
	last Snapshot
	page string
}

func displayedSnapshot(s Snapshot, page string, now time.Time) Snapshot {
	if page != "log" {
		s.Logs = nil
		if page != "devices" || !s.Busy {
			s.Status = ""
		}
	}
	// A renewed reading with the same value is not a visible change. Retain
	// freshness as a value so a stale reading becoming known still redraws.
	earbuds := make(map[string]session.EarbudState)
	for _, side := range []string{"left", "right"} {
		if state, known := WearKnown(s.Session, side, now); known {
			state.UpdatedAt = time.Time{}
			earbuds[side] = state
		}
	}
	s.Session.Earbuds = earbuds
	return s
}

func (r *redrawState) record(s Snapshot, page string, now time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.page = page
	r.last = displayedSnapshot(s, page, now)
}

func (r *redrawState) changed(s Snapshot, now time.Time) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return !reflect.DeepEqual(r.last, displayedSnapshot(s, r.page, now))
}
