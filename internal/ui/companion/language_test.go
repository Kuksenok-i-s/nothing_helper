package companion

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLanguagePreferences(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nothing_helper", "interface.json")
	if got := loadLanguage(path); got != English {
		t.Fatalf("default = %q", got)
	}
	for _, lang := range []Language{Russian, English} {
		if err := saveLanguage(path, lang); err != nil {
			t.Fatal(err)
		}
		if got := loadLanguage(path); got != lang {
			t.Fatalf("loaded %q, want %q", got, lang)
		}
	}
	for _, data := range []string{`{"language":"fr"}`, `invalid json`} {
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		if got := loadLanguage(path); got != English {
			t.Fatalf("invalid preference fallback = %q", got)
		}
	}
	if err := saveLanguage(path, "fr"); err == nil {
		t.Fatal("invalid language accepted")
	}
}
func TestLanguageMessages(t *testing.T) {
	cases := map[string]string{
		"Звук": "Sound",
		"Поиск устройств · готово":  "Finding devices · done",
		"Сохранено: /tmp/Звук.json": "Saved: /tmp/Звук.json",
		"Сейчас активный источник — «Устройства». Переключить наушники на этот компьютер?": "The active source is “Устройства”. Switch the earbuds to this computer?",
		"наушник в ухе; выньте его перед поиском":                                          "earbud is in your ear; remove it before finding",
		"raw MAC AA:BB:CC:DD:EE:FF": "raw MAC AA:BB:CC:DD:EE:FF",
	}
	for source, want := range cases {
		if got := translateMessage(English, source); got != want {
			t.Errorf("%q: got %q want %q", source, got, want)
		}
		if got := translateMessage(Russian, source); got != source {
			t.Errorf("Russian message changed: %q", got)
		}
	}
}
