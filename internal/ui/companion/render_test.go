//go:build gio && render

package companion

import (
	"context"
	"image"
	"image/png"
	"os"
	"testing"
	"time"

	"gioui.org/gpu/headless"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"nothing_helper/internal/bt"
	"nothing_helper/internal/session"
	"nothing_helper/internal/spp"
)

type previewBackend struct{}

func (previewBackend) Snapshot() session.Snapshot      { return session.Snapshot{} }
func (previewBackend) Subscribe() <-chan session.Event { return nil }
func (previewBackend) Execute([]string) error          { return nil }
func (previewBackend) Battery() error                  { return nil }

// Opt-in rendering uses software EGL on Linux; it never opens Bluetooth.
func TestRenderCompanion(t *testing.T) {
	path := os.Getenv("COMPANION_RENDER_PATH")
	if path == "" {
		t.Skip("set COMPANION_RENDER_PATH to save a preview")
	}
	model, ok := spp.ResolveModelInfo("EarThree")
	if !ok {
		t.Fatal("model unavailable")
	}
	s := Snapshot{Session: session.Snapshot{Connected: true, Device: bt.Device{Name: "Nothing Ear (3)"}, Model: model, Batteries: map[string]spp.Battery{"left": {Percent: 82}, "right": {Percent: 78}, "case": {Percent: 64}}, Config: map[string]string{"anc": "mode=high", "eq": "balanced", "dual": "dual=on", "lag": "low_latency=off", "spatial": "spatial=off", "walkie-talkie": "enabled=false", "super-mic": "enabled=false"}}}
	s.Session.Earbuds = map[string]session.EarbudState{
		"left":  {EarbudStatus: spp.EarbudStatus{InEar: true, Connected: true}, UpdatedAt: time.Now()},
		"right": {EarbudStatus: spp.EarbudStatus{Connected: true}, UpdatedAt: time.Now()},
	}
	if os.Getenv("COMPANION_RENDER_NO_CASE") == "1" {
		delete(s.Session.Batteries, "case")
	}
	var ops op.Ops
	size := image.Pt(430, 780)
	v := newView()
	v.language = English
	if os.Getenv("COMPANION_RENDER_LANGUAGE") == "ru" {
		v.language = Russian
	}
	v.light = os.Getenv("COMPANION_RENDER_THEME") == "light"
	v.details = os.Getenv("COMPANION_RENDER_DETAILS") == "1"
	if os.Getenv("COMPANION_RENDER_NARROW") == "1" {
		size = image.Pt(340, 1000)
	}
	c := newController(context.Background(), Options{}, previewBackend{})
	gtx := layout.Context{Ops: &ops, Now: time.Now(), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Exact(size)}
	v.Layout(gtx, c, s)
	window, err := headless.NewWindow(size.X, size.Y)
	if err != nil {
		t.Fatal(err)
	}
	defer window.Release()
	if err := window.Frame(&ops); err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rectangle{Max: size})
	if err := window.Screenshot(img); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}
