//go:build gio

package companion

import (
	"strings"

	"gioui.org/font/gofont"
	"gioui.org/layout"
	"gioui.org/op/paint"
	"gioui.org/text"

	"gioui.org/widget"
	"gioui.org/widget/material"

	"nothing_helper/internal/spp"
	"nothing_helper/internal/ui/presenter"
)

type view struct {
	language         Language
	preferencesPath  string
	preferencesError string
	th               *material.Theme
	p                palette
	light, details   bool
	page             string
	clicks           map[string]*widget.Clickable
	list             widget.List
	password         widget.Editor
}

func newView() *view {
	v := &view{th: material.NewTheme(), page: "sound", clicks: map[string]*widget.Clickable{}}
	v.preferencesPath = languagePath()
	v.language = loadLanguage(v.preferencesPath)
	v.th.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))
	v.list.Axis = layout.Vertical
	v.password.SingleLine = true
	v.password.Submit = true
	v.password.Mask = '•'
	return v
}
func (v *view) Layout(gtx layout.Context, c *Controller, s Snapshot) layout.Dimensions {
	v.p = colors(v.light)
	v.th.Palette = material.Palette{Bg: v.p.bg, Fg: v.p.fg, ContrastBg: v.p.accent, ContrastFg: v.p.onAccent}
	paint.Fill(gtx.Ops, v.p.bg)
	rows := []layout.Widget{
		func(gtx layout.Context) layout.Dimensions { return v.toolbar(gtx, c, s) },
	}
	if s.PasswordPrompt != "" {
		rows = append(rows, func(gtx layout.Context) layout.Dimensions { return v.passwordForm(gtx, c, s) })
	}
	if s.DualPrompt != "" {
		rows = append(rows, func(gtx layout.Context) layout.Dimensions { return v.dualPrompt(gtx, c, s) })
	}
	if v.preferencesError != "" {
		rows = append(rows, func(gtx layout.Context) layout.Dimensions {
			return v.label(gtx, 12, v.message(v.preferencesError), false)
		})
	}
	if s.Error != "" {
		rows = append(rows, func(gtx layout.Context) layout.Dimensions {
			return surface(gtx, v.p.surface, 10, 12, func(gtx layout.Context) layout.Dimensions { return v.label(gtx, 12, v.message(s.Error), false) })
		})
	}
	switch v.page {
	case "devices":
		rows = append(rows, v.devices(c, s)...)
	case "log":
		rows = append(rows, v.diagnostics(c, s)...)
	default:
		rows = append(rows, v.sound(c, s)...)
	}
	return layout.UniformInset(20).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return material.List(v.th, &v.list).Layout(gtx, len(rows), func(gtx layout.Context, i int) layout.Dimensions {
			return layout.Inset{Bottom: 14}.Layout(gtx, rows[i])
		})
	})
}
func (v *view) toolbar(gtx layout.Context, c *Controller, s Snapshot) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return v.heading(gtx, 12, "NOTHING_HELPER") }),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					label := v.tr("Светлая")
					if v.light {
						label = v.tr("Тёмная")
					}
					return v.button(gtx, "theme", label, false, true, func() { v.light = !v.light })
				}),
			)
		}), space(14), layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			tabs := []struct{ id, label string }{{"sound", v.tr("Звук")}, {"devices", v.tr("Устройства")}, {"log", v.tr("Диагностика")}}
			items := []layout.FlexChild{}
			for i, tab := range tabs {
				if i > 0 {
					items = append(items, layout.Rigid(layout.Spacer{Width: 6}.Layout))
				}
				items = append(items, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return v.button(gtx, "tab-"+tab.id, tab.label, v.page == tab.id, true, func() { v.page = tab.id; v.list.Position = layout.Position{}; c.Interaction() })
				}))
			}
			return layout.Flex{}.Layout(gtx, items...)
		}),
	)
}
func (v *view) sound(c *Controller, s Snapshot) []layout.Widget {
	snap := s.Session
	enabled := snap.Connected && !s.Busy
	rows := []layout.Widget{
		func(gtx layout.Context) layout.Dimensions { return v.deviceArt(gtx, c, s) },
		func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, layout.Rigid(func(gtx layout.Context) layout.Dimensions { return v.heading(gtx, 25, DeviceName(snap)) }), space(5), layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				text := v.tr("Не подключены")
				if snap.Connected {
					text = v.tr("Подключены · Bluetooth")
				}
				if s.Busy {
					text = v.tr("Выполняется операция…")
				}
				if s.FindSide != "" {
					text = v.tr("Поиск левого · нажмите на него ещё раз для остановки")
					if s.FindSide == "right" {
						text = v.tr("Поиск правого · нажмите на него ещё раз для остановки")
					}
				}
				return v.label(gtx, 12, text, true)
			}))
		},
	}
	if !snap.Connected {
		rows = append(rows, func(gtx layout.Context) layout.Dimensions {
			return v.button(gtx, "connect", v.tr("Найти и подключить"), true, !s.Busy, func() { c.SetAuto(true); c.Connect() })
		})
	}
	if spp.ModelSupportsFeature(snap.Model, "anc") && snap.Connected {
		rows = append(rows, func(gtx layout.Context) layout.Dimensions {
			return v.heading(gtx, 14, v.tr("Управление шумом"))
		}, func(gtx layout.Context) layout.Dimensions { return v.anc(gtx, c, s, enabled) })
	}
	if snap.Connected && spp.ModelSupportsFeature(snap.Model, "lag") {
		rows = append(rows, func(gtx layout.Context) layout.Dimensions {
			return v.featureToggle(gtx, c, s, "lag", v.tr("Низкая задержка"), enabled)
		})
	}
	if snap.Connected && spp.ModelSupportsFeature(snap.Model, "walkie-talkie") {
		rows = append(rows, func(gtx layout.Context) layout.Dimensions {
			return surface(gtx, v.p.surface, 14, 14, func(gtx layout.Context) layout.Dimensions {
				return v.featureToggle(gtx, c, s, "walkie-talkie", v.tr("Walkie Talkie · кнопка TALK"), enabled)
			})
		})
	}

	rows = append(rows, func(gtx layout.Context) layout.Dimensions {
		label := v.tr("Все настройки")
		if v.details {
			label = v.tr("Свернуть настройки")
		}
		return v.button(gtx, "details", label, false, true, func() { v.details = !v.details; c.Interaction() })
	})
	if v.details {
		rows = append(rows, v.languageSettings)
	}
	if v.details && snap.Connected {
		if spp.ModelSupportsFeature(snap.Model, "anc") {
			rows = append(rows, func(gtx layout.Context) layout.Dimensions { return v.ancLevel(gtx, c, s, enabled) })
		}
		if spp.ModelSupportsFeature(snap.Model, "eq") {
			rows = append(rows, func(gtx layout.Context) layout.Dimensions { return v.eq(gtx, c, s, enabled) })
		}
		for _, tf := range presenter.ToggleFeatures(presenter.BuildCommands(snap.Model, snap.DualList, false)) {
			if tf.Feature == "lag" {
				continue
			}
			rows = append(rows, func(gtx layout.Context) layout.Dimensions {
				labels := map[string]string{"lag": v.tr("Низкая задержка"), "spatial": v.tr("Пространственное аудио"), "dual": v.tr("Два устройства")}
				return v.featureToggle(gtx, c, s, tf.Feature, labels[tf.Feature], enabled)
			})
		}
		if spp.ModelSupportsFeature(snap.Model, "dual") {
			rows = append(rows, func(gtx layout.Context) layout.Dimensions {
				return v.row(gtx, v.tr("Источники звука"), v.tr("Подключённые к наушникам устройства"), func(gtx layout.Context) layout.Dimensions {
					return v.button(gtx, "peers", v.tr("Обновить"), false, enabled, c.RefreshPeers)
				})
			})
			for _, peer := range snap.DualList {
				rows = append(rows, func(gtx layout.Context) layout.Dimensions {
					name := peer.Name
					if name == "" {
						name = peer.MAC
					}
					sub := v.tr("Не подключено")
					if peer.Connected {
						sub = v.tr("Подключено")
					}
					if peer.Owner {
						sub += v.tr(" · активный источник")
					}
					return v.row(gtx, name, sub, func(gtx layout.Context) layout.Dimensions {
						label := v.tr("Подключить")
						if peer.Connected {
							label = v.tr("Отключить")
						}
						return v.button(gtx, "peer-"+peer.MAC, label, false, enabled, func() { c.Interaction(); c.DualPeer(peer) })
					})
				})
			}
		}
	}
	rows = append(rows, func(gtx layout.Context) layout.Dimensions { return v.line(gtx) }, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return v.button(gtx, "refresh", v.tr("Обновить"), false, enabled, c.Refresh)
		}), layout.Rigid(layout.Spacer{Width: 8}.Layout), layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return v.button(gtx, "disconnect", v.tr("Отключить"), false, enabled, c.Disconnect)
		}))
	})
	return rows
}
func (v *view) anc(gtx layout.Context, c *Controller, s Snapshot, enabled bool) layout.Dimensions {
	choices := []struct{ id, label string }{{"off", v.tr("Выкл.")}, {"transparency", v.tr("Прозрачность")}, {"strong", v.tr("Подавление")}}
	current := ANCMode(s.Session.Config["anc"])
	items := []layout.FlexChild{}
	for i, item := range choices {
		if i > 0 {
			items = append(items, layout.Rigid(layout.Spacer{Width: 5}.Layout))
		}
		items = append(items, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			label := item.label
			if item.id == "transparency" && gtx.Constraints.Max.X < gtx.Dp(112) {
				label = v.tr("Прозрачн.")
			}
			return v.button(gtx, "anc-"+item.id, label, item.id == current, enabled, func() { c.Interaction(); c.Feature("anc", item.id) })
		}))
	}
	return layout.Flex{}.Layout(gtx, items...)
}
func (v *view) ancLevel(gtx layout.Context, c *Controller, s Snapshot, enabled bool) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return v.label(gtx, 12, v.tr("Интенсивность ANC"), true)
	}), space(8), layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		items := []layout.FlexChild{}
		for i, item := range []struct{ id, label string }{{"weak", v.tr("Низкая")}, {"medium", v.tr("Средняя")}, {"strong", v.tr("Высокая")}, {"adaptive", v.tr("Авто")}} {
			if i > 0 {
				items = append(items, layout.Rigid(layout.Spacer{Width: 4}.Layout))
			}
			items = append(items, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				label := item.label
				if item.id == "medium" && gtx.Constraints.Max.X < gtx.Dp(90) {
					label = v.tr("Сред.")
				}
				return v.button(gtx, "level-"+item.id, label, strings.Contains(s.Session.Config["anc"], "mode="+map[string]string{"weak": "low", "medium": "mid", "strong": "high", "adaptive": "adaptive"}[item.id]), enabled, func() { c.Feature("anc", item.id) })
			}))
		}
		return layout.Flex{}.Layout(gtx, items...)
	}))
}
func (v *view) eq(gtx layout.Context, c *Controller, s Snapshot, enabled bool) layout.Dimensions {
	values := []struct{ id, name, label string }{{"0", "balanced", v.tr("Баланс")}, {"3", "more_bass", v.tr("Бас")}, {"2", "more_treble", v.tr("Высокие")}, {"1", "voice", v.tr("Голос")}}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, layout.Rigid(func(gtx layout.Context) layout.Dimensions { return v.heading(gtx, 14, v.tr("Эквалайзер")) }), space(9), layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		items := []layout.FlexChild{}
		for i, item := range values {
			if i > 0 {
				items = append(items, layout.Rigid(layout.Spacer{Width: 4}.Layout))
			}
			items = append(items, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				label := item.label
				if item.id == "0" && v.language != Russian && gtx.Constraints.Max.X < gtx.Dp(90) {
					label = "Flat"
				}
				return v.button(gtx, "eq-"+item.id, label, s.Session.Config["eq"] == item.name, enabled, func() { c.Feature("eq", item.id) })
			}))
		}
		return layout.Flex{}.Layout(gtx, items...)
	}))
}
func (v *view) featureToggle(gtx layout.Context, c *Controller, s Snapshot, feature, title string, enabled bool) layout.Dimensions {
	value, known := s.Session.Config[feature]
	on := presenter.ToggleStateOn(feature, value)
	sub := v.tr("Состояние не получено")
	if known {
		sub = v.tr("Выключено")
		if on {
			sub = v.tr("Включено")
		}
	}
	return v.row(gtx, title, sub, func(gtx layout.Context) layout.Dimensions {
		title := v.tr("Включить")
		if on {
			title = v.tr("Выключить")
		}
		return v.button(gtx, feature, title, on, enabled, func() {
			c.Interaction()
			value := "on"
			if on {
				value = "off"
			}
			c.Feature(feature, value)
		})
	})
}
