//go:build gio

package companion

import (
	"fmt"
	"gioui.org/layout"
	"gioui.org/op"
)

func (v *view) tr(text string) string      { return translate(v.language, text) }
func (v *view) message(text string) string { return translateMessage(v.language, text) }
func (v *view) setLanguage(lang Language) {
	if !validLanguage(lang) {
		return
	}
	v.language = lang
	v.preferencesError = ""
	if err := saveLanguage(v.preferencesPath, lang); err != nil {
		v.preferencesError = fmt.Sprintf("Не удалось сохранить язык: %s", err)
	}
}
func (v *view) languageSettings(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return v.heading(gtx, 14, v.tr("Язык интерфейса"))
		}), space(8),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return v.button(gtx, "language-en", "English", v.language == English, true, func() { v.setLanguage(English); gtx.Execute(op.InvalidateCmd{}) })
				}),
				layout.Rigid(layout.Spacer{Width: 8}.Layout),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return v.button(gtx, "language-ru", "Русский", v.language == Russian, true, func() { v.setLanguage(Russian); gtx.Execute(op.InvalidateCmd{}) })
				}),
			)
		}),
	)
}
