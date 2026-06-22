//go:build darwin && systray

package macosnative

import (
	"unsafe"
)

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa
#include <stdlib.h>

void tray_darwin_schedule_init(const char *tooltip, const void *iconData, int iconLen,
                               int showWindow, int showReconnect);
void tray_darwin_set_status(const char *title);
void tray_darwin_set_battery(const char *title);
void tray_darwin_set_tooltip(const char *tooltip);
*/
import "C"

// MenuClickHandler receives tray menu item tag clicks from Cocoa.
var MenuClickHandler func(tag int32)

//export tray_go_menu_click
func tray_go_menu_click(tag int32) {
	if MenuClickHandler != nil {
		MenuClickHandler(tag)
	}
}

func ScheduleInit(tooltip string, icon []byte, showWindow, showReconnect bool) {
	tip := C.CString(tooltip)
	defer C.free(unsafe.Pointer(tip))
	var iconPtr unsafe.Pointer
	iconLen := 0
	if len(icon) > 0 {
		iconPtr = unsafe.Pointer(&icon[0])
		iconLen = len(icon)
	}
	sw := 0
	if showWindow {
		sw = 1
	}
	sr := 0
	if showReconnect {
		sr = 1
	}
	C.tray_darwin_schedule_init(tip, iconPtr, C.int(iconLen), C.int(sw), C.int(sr))
}

func SetStatus(title string) {
	s := C.CString(title)
	defer C.free(unsafe.Pointer(s))
	C.tray_darwin_set_status(s)
}

func SetBattery(title string) {
	s := C.CString(title)
	defer C.free(unsafe.Pointer(s))
	C.tray_darwin_set_battery(s)
}

func SetTooltip(tooltip string) {
	s := C.CString(tooltip)
	defer C.free(unsafe.Pointer(s))
	C.tray_darwin_set_tooltip(s)
}
