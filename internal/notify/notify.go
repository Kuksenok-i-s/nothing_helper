// Package notify sends freedesktop desktop notifications (GNOME and other
// notification daemons) without pulling a D-Bus dependency. It shells out to
// gdbus (preferred, supports in-place replacement) or notify-send, and degrades
// to a no-op when neither is available.
package notify

import (
	"os/exec"
	"regexp"
	"strconv"
	"sync"
)

// Urgency maps to the freedesktop "urgency" hint.
type Urgency byte

const (
	UrgencyLow      Urgency = 0
	UrgencyNormal   Urgency = 1
	UrgencyCritical Urgency = 2
)

// Notifier posts desktop notifications. It is safe for concurrent use.
type Notifier struct {
	app  string
	icon string

	mu   sync.Mutex
	id   uint32 // replaces_id of the persistent (battery) notification
	send func(replaces uint32, urgency Urgency, title, body, icon string) uint32
}

// New returns a Notifier for appName. The default icon is "audio-headphones".
func New(appName, icon string) *Notifier {
	if appName == "" {
		appName = "tws_manager"
	}
	if icon == "" {
		icon = "audio-headphones"
	}
	n := &Notifier{app: appName, icon: icon}
	switch {
	case have("gdbus"):
		n.send = n.sendGdbus
	case have("notify-send"):
		n.send = n.sendNotifySend
	default:
		n.send = func(uint32, Urgency, string, string, string) uint32 { return 0 }
	}
	return n
}

// Available reports whether a notification backend was found.
func (n *Notifier) Available() bool {
	return have("gdbus") || have("notify-send")
}

// Update posts or replaces the persistent notification (e.g. battery levels),
// so repeated updates collapse into one entry instead of stacking.
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

var gdbusID = regexp.MustCompile(`uint32 (\d+)`)

func (n *Notifier) sendGdbus(replaces uint32, urgency Urgency, title, body, icon string) uint32 {
	hints := "@a{sv} {'urgency': <byte " + strconv.Itoa(int(urgency)) + ">}"
	cmd := exec.Command("gdbus", "call", "--session",
		"--dest", "org.freedesktop.Notifications",
		"--object-path", "/org/freedesktop/Notifications",
		"--method", "org.freedesktop.Notifications.Notify",
		n.app,
		strconv.FormatUint(uint64(replaces), 10),
		icon, title, body,
		"@as []", hints,
		"5000",
	)
	out, err := cmd.Output()
	if err != nil {
		return replaces
	}
	if m := gdbusID.FindSubmatch(out); m != nil {
		if v, err := strconv.ParseUint(string(m[1]), 10, 32); err == nil {
			return uint32(v)
		}
	}
	return replaces
}

func (n *Notifier) sendNotifySend(replaces uint32, urgency Urgency, title, body, icon string) uint32 {
	args := []string{"-a", n.app, "-i", icon, "-u", urgencyName(urgency)}
	// Coalesce repeated updates where the daemon supports the synchronous hint.
	args = append(args, "-h", "string:x-canonical-private-synchronous:tws_manager")
	args = append(args, title, body)
	_ = exec.Command("notify-send", args...).Run()
	return 0
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

func have(tool string) bool {
	_, err := exec.LookPath(tool)
	return err == nil
}
