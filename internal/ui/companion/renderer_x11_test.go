//go:build gio && linux && nowayland

package companion

import (
	"errors"
	"strings"
	"testing"
)

func TestRendererRetryIsBoundedAndSpecific(t *testing.T) {
	err := errors.New("newContext: eglCreateWindowSurface failed 0x3005 (sRGB=false)")
	if !shouldRetryRenderer(err, "") {
		t.Fatal("EGL failure not recoverable")
	}
	if shouldRetryRenderer(err, "1") {
		t.Fatal("restart loop possible")
	}
	if shouldRetryRenderer(errors.New("Bluetooth failed"), "") || shouldRetryRenderer(nil, "") {
		t.Fatal("unrelated error retried")
	}
}
func TestSoftwareRendererEnvironment(t *testing.T) {
	original := []string{"DISPLAY=:0", "EGL_PLATFORM=wayland", "LIBGL_ALWAYS_SOFTWARE=0", "__EGL_VENDOR_LIBRARY_FILENAMES=nvidia.json", rendererRetried + "=0"}
	env := softwareRendererEnv(original, "/mesa.json")
	values := map[string]string{}
	for _, entry := range env {
		k, v, _ := strings.Cut(entry, "=")
		if _, ok := values[k]; ok {
			t.Fatalf("duplicate %s", k)
		}
		values[k] = v
	}
	if values["DISPLAY"] != ":0" || values["EGL_PLATFORM"] != "x11" || values["LIBGL_ALWAYS_SOFTWARE"] != "1" || values["__EGL_VENDOR_LIBRARY_FILENAMES"] != "/mesa.json" || values[rendererRetried] != "1" {
		t.Fatalf("wrong environment: %v", values)
	}
	if original[1] != "EGL_PLATFORM=wayland" {
		t.Fatal("caller environment mutated")
	}
}
