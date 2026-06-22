package notify

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"tws_manager/internal/session"
	"tws_manager/internal/spp"
)

// lowLevel describes one low-battery threshold and the alert it raises.
type lowLevel struct {
	percent int
	urgency Urgency
	title   string
	bodyFmt string // receives component label and percent
}

// earbudLowLevels are the descending battery thresholds for the earbuds.
// Ordered high → low so a single pass picks the lowest threshold crossed.
var earbudLowLevels = []lowLevel{
	{percent: 20, urgency: UrgencyNormal, title: "Battery low", bodyFmt: "%s at %d%%"},
	{percent: 10, urgency: UrgencyCritical, title: "Charge required", bodyFmt: "%s at %d%% - please charge"},
	{percent: 5, urgency: UrgencyCritical, title: "Battery critical", bodyFmt: "%s at %d%% - about 3-5 minutes left, charge now"},
}

// caseLowLevels are the thresholds for the charging case (less urgent than the
// earbuds, which are the part actually in use).
var caseLowLevels = []lowLevel{
	{percent: 20, urgency: UrgencyNormal, title: "Case battery low", bodyFmt: "%s at %d%%"},
	{percent: 10, urgency: UrgencyCritical, title: "Charge the case", bodyFmt: "%s at %d%% - charge the case soon"},
}

// Options configures the notification watcher.
type Options struct {
	AppName string
	// EarbudLevels / CaseLevels override the default low-battery thresholds.
	EarbudLevels []lowLevel
	CaseLevels   []lowLevel
}

// Run subscribes to session events and emits desktop notifications for
// connect/disconnect and low-battery warnings. It returns when ctx is cancelled
// or the event stream closes.
func Run(ctx context.Context, s *session.Session, opts Options) {
	if opts.AppName == "" {
		opts.AppName = "tws_manager"
	}
	earbudLevels := opts.EarbudLevels
	if earbudLevels == nil {
		earbudLevels = earbudLowLevels
	}
	caseLevels := opts.CaseLevels
	if caseLevels == nil {
		caseLevels = caseLowLevels
	}
	sortLevels(earbudLevels)
	sortLevels(caseLevels)

	n := New(opts.AppName, "audio-headphones")
	if !n.Available() {
		Warnf("desktop notifications disabled: gdbus and notify-send not found in PATH")
		return
	}

	watch(ctx, s.Subscribe(), s, n, opts, earbudLevels, caseLevels)
}

type sessionView interface {
	Snapshot() session.Snapshot
}

func watch(ctx context.Context, events <-chan session.Event, sv sessionView, n *Notifier, opts Options, earbudLevels, caseLevels []lowLevel) {
	lowFired := map[string]int{}
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-events:
			if !ok {
				return
			}
			processEvent(ev, sv, n, opts, earbudLevels, caseLevels, lowFired)
		}
	}
}

func processEvent(ev session.Event, sv sessionView, n *Notifier, opts Options, earbudLevels, caseLevels []lowLevel, lowFired map[string]int) {
	switch ev.Kind {
	case session.EventConnected:
		n.Alert(UrgencyNormal, opts.AppName, "Connected to "+deviceName(sv, opts.AppName))
		checkLowBattery(n, sv.Snapshot().Batteries, earbudLevels, caseLevels, lowFired)
	case session.EventDisconnected:
		clear(lowFired)
		n.Alert(UrgencyNormal, opts.AppName, "Disconnected")
	case session.EventBattery:
		checkLowBattery(n, sv.Snapshot().Batteries, earbudLevels, caseLevels, lowFired)
	}
}

func sortLevels(levels []lowLevel) {
	sort.Slice(levels, func(i, j int) bool { return levels[i].percent > levels[j].percent })
}

func checkLowBattery(n *Notifier, data map[string]spp.Battery, earbudLevels, caseLevels []lowLevel, fired map[string]int) {
	for _, comp := range orderedComponents(data) {
		levels := earbudLevels
		if comp == "case" {
			levels = caseLevels
		}
		b := data[comp]
		if b.Charging {
			delete(fired, comp)
			continue
		}
		var hit *lowLevel
		for i := range levels {
			if b.Percent <= levels[i].percent {
				hit = &levels[i]
			}
		}
		if hit == nil {
			// Require recovery above the last fired threshold before alerting again
			// (avoids repeat alerts when percent oscillates around a threshold).
			if prev, ok := fired[comp]; ok && b.Percent > prev+5 {
				delete(fired, comp)
			}
			continue
		}
		if prev, ok := fired[comp]; ok && prev <= hit.percent {
			continue
		}
		fired[comp] = hit.percent
		n.Alert(hit.urgency, hit.title,
			fmt.Sprintf(hit.bodyFmt, componentLabel(comp), b.Percent))
	}
}

func deviceName(sv sessionView, fallback string) string {
	snap := sv.Snapshot()
	if snap.Device.Name != "" {
		return snap.Device.Name
	}
	if snap.Model.Product != "" {
		return snap.Model.Product
	}
	if snap.Device.MAC != "" {
		return snap.Device.MAC
	}
	return fallback
}

func orderedComponents(data map[string]spp.Battery) []string {
	order := []string{"left", "right", "case", "stereo", "tws", "watch"}
	seen := make(map[string]struct{}, len(data))
	out := make([]string, 0, len(data))
	for _, k := range order {
		if _, ok := data[k]; ok {
			out = append(out, k)
			seen[k] = struct{}{}
		}
	}
	extras := make([]string, 0, len(data))
	for k := range data {
		if _, ok := seen[k]; ok {
			continue
		}
		extras = append(extras, k)
	}
	sort.Strings(extras)
	return append(out, extras...)
}

func componentLabel(comp string) string {
	switch comp {
	case "left":
		return "Left earbud"
	case "right":
		return "Right earbud"
	case "case":
		return "Case"
	case "stereo":
		return "Headphones"
	case "tws":
		return "Earbuds"
	case "watch":
		return "Watch"
	default:
		if strings.HasPrefix(comp, "id_") {
			return "Device " + strings.TrimPrefix(comp, "id_")
		}
		return comp
	}
}

func formatBatteries(data map[string]spp.Battery) string {
	labels := map[string]string{
		"left": "Left", "right": "Right", "case": "Case",
		"stereo": "Headphones", "tws": "Earbuds", "watch": "Watch",
	}
	parts := make([]string, 0, len(data))
	for _, comp := range orderedComponents(data) {
		label := labels[comp]
		if label == "" {
			label = componentLabel(comp)
		}
		b := data[comp]
		s := fmt.Sprintf("%s %d%%", label, b.Percent)
		if b.Charging {
			s += " ⚡"
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, "   ")
}
