//go:build linux

package notify

import (
	"os/exec"
	"regexp"
	"strconv"
)

var gdbusID = regexp.MustCompile(`uint32 (\d+)`)

// New returns a Notifier for appName using gdbus or notify-send.
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
		n.backend = "gdbus"
		n.send = n.sendGdbus
	case have("notify-send"):
		n.backend = "notify-send"
		n.send = n.sendNotifySend
	default:
		n.send = func(uint32, Urgency, string, string, string) uint32 { return 0 }
	}
	return n
}

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
		Warnf("gdbus Notify failed (%s): %v", title, err)
		if replaces == 0 && have("notify-send") {
			return n.sendNotifySendDirect(urgency, title, body, icon)
		}
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
	_ = replaces
	return n.sendNotifySendDirect(urgency, title, body, icon)
}

func (n *Notifier) sendNotifySendDirect(urgency Urgency, title, body, icon string) uint32 {
	args := []string{"-a", n.app, "-i", icon, "-u", urgencyName(urgency)}
	args = append(args, "-h", "string:x-canonical-private-synchronous:tws_manager")
	args = append(args, title, body)
	if err := exec.Command("notify-send", args...).Run(); err != nil {
		Warnf("notify-send failed (%s): %v", title, err)
	}
	return 0
}
