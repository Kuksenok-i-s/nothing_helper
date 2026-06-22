//go:build darwin

package notify

import (
	"fmt"
	"os/exec"
)

// New returns a Notifier for appName using osascript on macOS.
func New(appName, icon string) *Notifier {
	if appName == "" {
		appName = "tws_manager"
	}
	if icon == "" {
		icon = "audio-headphones"
	}
	n := &Notifier{app: appName, icon: icon, backend: "osascript"}
	n.send = n.sendOsascript
	return n
}

func (n *Notifier) sendOsascript(replaces uint32, urgency Urgency, title, body, icon string) uint32 {
	_ = replaces
	_ = urgency
	_ = icon
	script := fmt.Sprintf(`display notification %q with title %q subtitle %q`,
		body, n.app, title)
	if err := exec.Command("osascript", "-e", script).Run(); err != nil {
		Warnf("osascript failed (%s): %v", title, err)
	}
	return 0
}
