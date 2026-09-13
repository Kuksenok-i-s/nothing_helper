//go:build gio

package companion

import (
	"context"
	"image"
	"path/filepath"
	"testing"
	"time"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

func TestLanguageInsideSettingsWithoutConnection(t *testing.T) {
	v := newView()
	v.preferencesPath = filepath.Join(t.TempDir(), "interface.json")
	v.language = loadLanguage(v.preferencesPath)
	backend := &fakeBackend{}
	c := newController(context.Background(), Options{}, backend)
	var ops op.Ops
	render := func() {
		ops.Reset()
		v.Layout(layout.Context{Ops: &ops, Now: time.Now(), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Exact(image.Pt(430, 1600))}, c, Snapshot{})
	}
	render()
	if v.clicks["language-ru"] != nil {
		t.Fatal("language selector visible while settings collapsed")
	}
	v.details = true
	render()
	if v.clicks["language-ru"] == nil {
		t.Fatal("language selector unavailable when disconnected")
	}
	v.clicks["language-ru"].Click()
	render()
	if v.language != Russian || v.tr("Все настройки") != "Все настройки" {
		t.Fatal("Russian not applied immediately")
	}
	if got := loadLanguage(v.preferencesPath); got != Russian {
		t.Fatalf("saved language = %q", got)
	}
	if len(backend.calls) != 0 {
		t.Fatal("language switch sent a device command")
	}
	v.clicks["language-en"].Click()
	render()
	if v.language != English || v.tr("Все настройки") != "All settings" {
		t.Fatal("English not restored")
	}
}
