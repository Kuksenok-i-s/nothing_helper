//go:build gio

package companion

import (
	"fmt"

	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

func (v *view) devices(c *Controller, s Snapshot) []layout.Widget {
	rows := []layout.Widget{
		func(gtx layout.Context) layout.Dimensions { return v.heading(gtx, 23, "Устройства") },
		func(gtx layout.Context) layout.Dimensions {
			return v.label(gtx, 12, "Сначала выполните сопряжение наушников в настройках Bluetooth системы.", true)
		},
		func(gtx layout.Context) layout.Dimensions {
			return v.button(gtx, "discover", "Найти устройства", true, !s.Busy, c.Discover)
		},
	}
	if len(s.Devices) == 0 {
		rows = append(rows, func(gtx layout.Context) layout.Dimensions {
			return v.label(gtx, 13, "Нажмите «Найти устройства», чтобы выбрать Nothing или CMF.", true)
		})
	}
	for _, dev := range s.Devices {
		rows = append(rows, func(gtx layout.Context) layout.Dimensions {
			name := dev.Name
			if name == "" {
				name = dev.MAC
			}
			selected := s.Session.Connected && s.Session.Device.MAC == dev.MAC
			sub := dev.MAC
			if selected {
				sub = "Подключены"
			}
			return surface(gtx, v.p.surface, 12, 12, func(gtx layout.Context) layout.Dimensions {
				return v.row(gtx, name, sub, func(gtx layout.Context) layout.Dimensions {
					label := "Выбрать"
					if selected {
						label = "Активны"
					}
					return v.button(gtx, "device-"+dev.MAC, label, selected, !selected && !s.Busy, func() { c.SetAuto(true); c.ConnectDevice(dev) })
				})
			})
		})
	}
	if s.Busy {
		rows = append(rows, func(gtx layout.Context) layout.Dimensions { return v.label(gtx, 12, s.Status, true) })
	}
	return rows
}
func (v *view) diagnostics(c *Controller, s Snapshot) []layout.Widget {
	rows := []layout.Widget{
		func(gtx layout.Context) layout.Dimensions { return v.heading(gtx, 23, "Диагностика") },
		func(gtx layout.Context) layout.Dimensions {
			return v.label(gtx, 12, fmt.Sprintf("%s\nMAC: %s\nКанал: %d", DeviceName(s.Session), s.Session.Device.MAC, s.Session.Device.Channel), true)
		},
		func(gtx layout.Context) layout.Dimensions { return v.label(gtx, 12, s.Status, true) },
		func(gtx layout.Context) layout.Dimensions {
			return v.button(gtx, "export", "Экспорт журнала", false, !s.Busy && len(s.Logs) > 0, c.Export)
		},
	}
	if len(s.Logs) == 0 {
		rows = append(rows, func(gtx layout.Context) layout.Dimensions {
			return v.label(gtx, 12, "События появятся после подключения устройства.", true)
		})
	}
	// Latest events first; the controller retains a bounded trace history.
	for i := len(s.Logs) - 1; i >= 0; i-- {
		line := s.Logs[i]
		rows = append(rows, func(gtx layout.Context) layout.Dimensions { return v.label(gtx, 11, line, true) })
	}
	return rows
}
func (v *view) passwordForm(gtx layout.Context, c *Controller, s Snapshot) layout.Dimensions {
	for {
		event, ok := v.password.Update(gtx)
		if !ok {
			break
		}
		if _, ok := event.(widget.SubmitEvent); ok {
			c.AnswerPassword(v.password.Text())
			v.password.SetText("")
		}
	}
	return surface(gtx, v.p.surface, 12, 14, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return v.heading(gtx, 14, "Доступ к Bluetooth") }), space(8),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return v.label(gtx, 12, s.PasswordPrompt, true) }), space(12),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				editor := material.Editor(v.th, &v.password, "Пароль sudo")
				editor.TextSize = 14
				return editor.Layout(gtx)
			}), space(12),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return v.button(gtx, "password-ok", "Разрешить", true, true, func() { c.AnswerPassword(v.password.Text()); v.password.SetText("") })
					}),
					layout.Rigid(layout.Spacer{Width: 8}.Layout), layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return v.button(gtx, "password-no", "Отмена", false, true, func() { c.AnswerPassword(""); v.password.SetText("") })
					}),
				)
			}),
		)
	})
}
func (v *view) dualPrompt(gtx layout.Context, c *Controller, s Snapshot) layout.Dimensions {
	return surface(gtx, v.p.surface, 12, 14, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return v.label(gtx, 13, s.DualPrompt, false) }), space(10),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return v.button(gtx, "dual-accept", "На компьютер", true, !s.Busy, func() { c.ResolveDual(true) })
					}), layout.Rigid(layout.Spacer{Width: 8}.Layout),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return v.button(gtx, "dual-decline", "Не сейчас", false, true, func() { c.ResolveDual(false) })
					}),
				)
			}),
		)
	})
}
