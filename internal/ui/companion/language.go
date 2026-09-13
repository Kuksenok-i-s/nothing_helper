package companion

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Language string

const (
	English Language = "en"
	Russian Language = "ru"
)

func validLanguage(lang Language) bool { return lang == English || lang == Russian }
func languagePath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "nothing_helper", "interface.json")
}
func loadLanguage(path string) Language {
	var settings struct {
		Language Language `json:"language"`
	}
	data, err := os.ReadFile(path)
	if err != nil || json.Unmarshal(data, &settings) != nil || !validLanguage(settings.Language) {
		return English
	}
	return settings.Language
}
func saveLanguage(path string, lang Language) error {
	if !validLanguage(lang) {
		return fmt.Errorf("unsupported language: %q", lang)
	}
	if path == "" {
		return fmt.Errorf("user configuration directory is unavailable")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".interface-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	data, _ := json.Marshal(struct {
		Language Language `json:"language"`
	}{lang})
	if _, err := f.Write(append(data, '\n')); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
func translate(lang Language, text string) string {
	if lang == Russian {
		return text
	}
	if translated, ok := englishText[text]; ok {
		return translated
	}
	return text
}

type messageTranslation struct {
	pattern *regexp.Regexp
	parts   []string
}

var messageTranslations = func() []messageTranslation {
	var entries []messageTranslation
	for source, target := range englishText {
		if !strings.Contains(source, "%s") {
			continue
		}
		parts := strings.Split(source, "%s")
		for i := range parts {
			parts[i] = regexp.QuoteMeta(parts[i])
		}
		entries = append(entries, messageTranslation{regexp.MustCompile("(?s)^" + strings.Join(parts, "(.*?)") + "$"), strings.Split(target, "%s")})
	}
	return entries
}()

// Translate application messages at display time so existing status/errors switch
// immediately. Captured device names, paths and OS error details remain unchanged.
func translateMessage(lang Language, text string) string {
	if lang == Russian {
		if strings.HasPrefix(text, "Administrator password is required for ") {
			return "Требуется пароль администратора для " + strings.TrimPrefix(text, "Administrator password is required for ")
		}
		return text
	}
	if translated, ok := englishText[text]; ok {
		return translated
	}
	if strings.HasSuffix(text, " · готово") {
		return translate(lang, strings.TrimSuffix(text, " · готово")) + " · done"
	}
	if strings.HasPrefix(text, "Сохранено: ") {
		return "Saved: " + strings.TrimPrefix(text, "Сохранено: ")
	}
	for _, entry := range messageTranslations {
		captures := entry.pattern.FindStringSubmatch(text)
		if captures == nil {
			continue
		}
		var out strings.Builder
		for i, part := range entry.parts {
			out.WriteString(part)
			if i+1 < len(captures) {
				out.WriteString(captures[i+1])
			}
		}
		return out.String()
	}
	return text
}
