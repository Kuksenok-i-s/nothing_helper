// Package notify sends desktop notifications using platform-specific backends.
package notify

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
)

// Urgency maps to notification priority hints where supported.
type Urgency byte

const (
	UrgencyLow      Urgency = 0
	UrgencyNormal   Urgency = 1
	UrgencyCritical Urgency = 2
)

// Warnf logs non-fatal notification issues. Tests may replace it.
var Warnf = func(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "warning: notify: "+format+"\n", args...)
}

// execLookPath is exec.LookPath; tests override it to simulate missing tools.
var execLookPath = exec.LookPath

// Notifier posts desktop notifications. It is safe for concurrent use.
type Notifier struct {
	app     string
	icon    string
	backend string

	mu   sync.Mutex
	id   uint32
	send func(replaces uint32, urgency Urgency, title, body, icon string) uint32
}

// Available reports whether a notification backend was found.
func (n *Notifier) Available() bool {
	return n.backend != ""
}

// Update posts or replaces the persistent notification (e.g. battery levels).
func (n *Notifier) Update(title, body string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.id = n.send(n.id, UrgencyNormal, title, body, n.icon)
}

// Alert posts a standalone notification (connect/disconnect, low battery).
func (n *Notifier) Alert(urgency Urgency, title, body string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.send(0, urgency, title, body, n.icon)
}

func have(tool string) bool {
	_, err := execLookPath(tool)
	return err == nil
}

func urgencyName(u Urgency) string {
	switch u {
	case UrgencyLow:
		return "low"
	case UrgencyCritical:
		return "critical"
	default:
		return "normal"
	}
}
