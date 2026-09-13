//go:build gio && linux && nowayland && windowsmoke

package companion

import (
	"fmt"
	"image/color"
	"os"
	"testing"
	"time"

	"gioui.org/app"
	"gioui.org/op"
	"gioui.org/op/paint"
	"github.com/getlantern/systray"
)

// Opt-in real display check: no Bluetooth. Two frames are insufficient: EGL
// initialization can fail asynchronously after the first submitted frames.
func TestMain(m *testing.M) {
	if os.Getenv("COMPANION_WINDOW_SMOKE") != "1" {
		os.Exit(m.Run())
	}
	runNativeWindowSmoke()
}

func runNativeWindowSmoke() {
	go func() {
		time.Sleep(10 * time.Second)
		fmt.Fprintln(os.Stderr, "native window smoke timed out")
		os.Exit(2)
	}()
	ready := make(chan struct{})
	go systray.Run(func() { close(ready) }, func() {})
	<-ready
	go func() {
		w := new(app.Window)
		w.Option(app.Title("Nothing renderer check"), app.Decorated(true))
		var ops op.Ops
		frames := 0
		for {
			switch e := w.Event().(type) {
			case app.DestroyEvent:
				err := retryRenderer(e.Err, systray.Quit)
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			case app.FrameEvent:
				ops.Reset()
				paint.Fill(&ops, color.NRGBA{R: 30, G: 30, B: 30, A: 255})
				e.Frame(&ops)
				frames++
				if frames >= 60 {
					fmt.Println("PASS: 60 native frames; fallback=" + os.Getenv(rendererRetried))
					os.Exit(0)
				}
				w.Invalidate()
			}
		}
	}()
	app.Main()
}
