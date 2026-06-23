//go:build darwin

package notify

/*
#cgo darwin LDFLAGS: -framework Foundation
#include <stdlib.h>
#include "notify_darwin.h"
*/
import "C"
import (
	"fmt"
	"os/exec"
	"unsafe"
)

// New returns a Notifier for appName. When the process runs from an application
// bundle it posts native banners attributed to the app; otherwise (e.g. a bare
// CLI binary) it falls back to osascript.
func New(appName, icon string) *Notifier {
	if appName == "" {
		appName = "tws_manager"
	}
	if icon == "" {
		icon = "audio-headphones"
	}
	n := &Notifier{app: appName, icon: icon}
	if C.notify_darwin_available() != 0 {
		n.backend = "nsusernotification"
		n.send = n.sendNative
	} else {
		n.backend = "osascript"
		n.send = n.sendOsascript
	}
	return n
}

// sendNative posts a banner via the native bridge. Mirrors the osascript layout:
// the app name is the title, the event title is the subtitle, and body is the
// informative text.
func (n *Notifier) sendNative(replaces uint32, urgency Urgency, title, body, icon string) uint32 {
	_ = replaces
	_ = urgency
	_ = icon
	ctitle := C.CString(n.app)
	csubtitle := C.CString(title)
	cbody := C.CString(body)
	defer C.free(unsafe.Pointer(ctitle))
	defer C.free(unsafe.Pointer(csubtitle))
	defer C.free(unsafe.Pointer(cbody))
	C.notify_darwin_post(ctitle, csubtitle, cbody)
	return 0
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
