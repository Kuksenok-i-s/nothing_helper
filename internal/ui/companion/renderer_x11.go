//go:build gio && linux && nowayland

package companion

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

const rendererRetried = "NOTHING_COMPANION_EGL_RETRIED"

// GLVND must select Mesa before GTK or Gio loads EGL, so recovery needs exec.
// This changes only the replacement process environment, not the desktop.
func retryRenderer(err error, shutdown func()) error {
	if !shouldRetryRenderer(err, os.Getenv(rendererRetried)) {
		return err
	}
	var mesa string
	for _, dir := range []string{"/usr/share/glvnd/egl_vendor.d", "/etc/glvnd/egl_vendor.d"} {
		matches, _ := filepath.Glob(filepath.Join(dir, "*mesa*.json"))
		if len(matches) > 0 {
			mesa = matches[0]
			break
		}
	}
	if mesa == "" {
		return fmt.Errorf("%w; Mesa EGL fallback unavailable (no Mesa GLVND manifest)", err)
	}
	executable, execErr := os.Executable()
	if execErr != nil {
		return fmt.Errorf("%w; cannot restart: %v", err, execErr)
	}
	env := softwareRendererEnv(os.Environ(), mesa)
	fmt.Fprintln(os.Stderr, "companion: X11 EGL failed; restarting once with Mesa software rendering (system window frame retained)")
	if shutdown != nil {
		shutdown()
	}
	if execErr := syscall.Exec(executable, os.Args, env); execErr != nil {
		return fmt.Errorf("%w; renderer restart failed: %v", err, execErr)
	}
	return err
}
func shouldRetryRenderer(err error, retried string) bool {
	return err != nil && retried != "1" && strings.Contains(err.Error(), "eglCreateWindowSurface failed")
}
func softwareRendererEnv(env []string, manifest string) []string {
	out := make([]string, 0, len(env)+4)
	for _, entry := range env {
		key, _, _ := strings.Cut(entry, "=")
		switch key {
		case rendererRetried, "__EGL_VENDOR_LIBRARY_FILENAMES", "LIBGL_ALWAYS_SOFTWARE", "EGL_PLATFORM":
			continue
		}
		out = append(out, entry)
	}
	return append(out, rendererRetried+"=1", "__EGL_VENDOR_LIBRARY_FILENAMES="+manifest, "LIBGL_ALWAYS_SOFTWARE=1", "EGL_PLATFORM=x11")
}
