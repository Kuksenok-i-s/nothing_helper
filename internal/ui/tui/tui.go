package tui

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"tws_manager/internal/bt"
	"tws_manager/internal/connect"
	"tws_manager/internal/dualpolicy"
	"tws_manager/internal/session"
	"tws_manager/internal/spp"
	"tws_manager/internal/trace"
	"tws_manager/internal/ui/actions"
	"tws_manager/internal/ui/dualprompt"
	"tws_manager/internal/ui/presenter"
)

type Options struct {
	Manager      *connect.Manager
	CaptureDir   string
	AllowUnsafe  bool
	LogRaw       bool
	AutoDiscover bool
	PCPrimary    dualpolicy.Mode
	Ctx          context.Context
}

type Model struct {
	session           *session.Session
	manager           *connect.Manager
	events            <-chan session.Event
	options           Options
	presenter         *presenter.State
	devices           []bt.Device
	commands          list.Model
	log               viewport.Model
	comment           textinput.Model
	activeTab         int
	commenting        bool
	pendingUnsafeItem *presenter.Command
	dualPrompt        dualprompt.Controller
	styles            styles
}

type styles struct {
	title  lipgloss.Style
	tab    lipgloss.Style
	ok     lipgloss.Style
	warn   lipgloss.Style
	panel  lipgloss.Style
	output lipgloss.Style
}

type eventMsg session.Event
type devicesMsg []bt.Device
type errMsg error

type commandItem struct {
	presenter.Command
}

func (i commandItem) Title() string       { return i.Command.Title }
func (i commandItem) Description() string { return i.Desc }
func (i commandItem) FilterValue() string { return i.Command.Title + " " + i.Desc }

func commandListItems(model spp.ModelInfo, dualDevices []spp.DualDevice, allowUnsafe bool) []list.Item {
	cmds := presenter.BuildCommands(model, dualDevices, allowUnsafe)
	items := make([]list.Item, len(cmds))
	for i, c := range cmds {
		items[i] = commandItem{Command: c}
	}
	return items
}

func New(s *session.Session, opts Options) Model {
	snap := s.Snapshot()
	items := commandListItems(snap.Model, snap.DualList, opts.AllowUnsafe)
	commands := list.New(items, list.NewDefaultDelegate(), 42, 14)
	commands.Title = "Commands"
	comment := textinput.New()
	comment.Placeholder = "comment for selected/last packet"
	p := presenter.NewState(opts.LogRaw)
	p.AutoReconnect = opts.AutoDiscover
	p.Status = "discovering devices..."
	return Model{
		session:    s,
		manager:    opts.Manager,
		events:     s.Subscribe(),
		options:    opts,
		presenter:  p,
		commands:   commands,
		log:        viewport.New(80, 16),
		comment:    comment,
		styles:     defaultStyles(),
		activeTab:  0,
		dualPrompt: dualprompt.Controller{Mode: opts.PCPrimary},
	}
}

func defaultStyles() styles {
	return styles{
		title:  lipgloss.NewStyle().Bold(true),
		tab:    lipgloss.NewStyle().Padding(0, 1),
		ok:     lipgloss.NewStyle().Foreground(lipgloss.Color("2")),
		warn:   lipgloss.NewStyle().Foreground(lipgloss.Color("3")),
		panel:  lipgloss.NewStyle().Width(62).PaddingRight(2),
		output: lipgloss.NewStyle().Width(78).Border(lipgloss.NormalBorder()).Padding(0, 1),
	}
}

func Run(ctx context.Context, s *session.Session, opts Options) error {
	defer func() { _ = s.Close() }()
	p := tea.NewProgram(New(s, opts), tea.WithAltScreen(), tea.WithContext(ctx))
	_, err := p.Run()
	return err
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(discoverCmd(m.manager), waitEventCmd(m.events))
}

func (m Model) View() string {
	snap := m.session.Snapshot()
	header := m.styles.title.Render("tws_manager") + "  " + m.presenter.Status
	if line := m.dualPromptLine(); line != "" {
		header += "\n" + m.styles.warn.Render(line)
	}
	if m.presenter.Err != "" {
		header += "  " + m.styles.warn.Render(m.presenter.Err)
	}
	battery := presenter.FormatBatteries(snap.Batteries)
	model := snap.Model.Codename
	if model == "" {
		model = "unknown"
	}
	body := fmt.Sprintf("Device: %s %s\nModel: %s\nBattery: %s\n\n", snap.Device.Name, snap.Device.MAC, model, battery)
	body += m.tabs()
	switch m.activeTab {
	case 0:
		body += m.devicesView()
	case 1:
		body += m.commands.View()
	case 2:
		body += m.log.View() + "\n\n" + m.comment.View()
	}
	output := m.styles.title.Render("Live Output") + "\n" + m.log.View()
	if m.commenting {
		output += "\n\n" + m.comment.View()
	}
	content := lipgloss.JoinHorizontal(lipgloss.Top, m.styles.panel.Render(body), m.styles.output.Render(output))
	return header + "\n" + content + "\n\nTab categories · Enter action · r refresh · c comment · s save · q close TUI · Ctrl+C quit"
}

func (m Model) tabs() string {
	names := []string{"Devices", "Control", "Log"}
	parts := make([]string, len(names))
	for i, name := range names {
		if i == m.activeTab {
			parts[i] = m.styles.title.Render("[" + name + "]")
		} else {
			parts[i] = m.styles.tab.Render(name)
		}
	}
	return strings.Join(parts, " ") + "\n\n"
}

func (m Model) devicesView() string {
	if len(m.devices) == 0 {
		return "No discovered devices yet. Use r to refresh or --addr to attach manually."
	}
	var b strings.Builder
	for i, dev := range m.devices {
		mark := " "
		if i == 0 {
			mark = ">"
		}
		fmt.Fprintf(&b, "%s %s  %s connected=%t spp=%t\n", mark, dev.MAC, dev.Name, dev.Connected, dev.SPP)
	}
	return b.String()
}

func (m *Model) applyEvent(event session.Event) {
	snap := m.session.Snapshot()
	m.presenter.ApplyEvent(event, snap)
	m.log.SetContent(m.presenter.LogText())
	m.log.GotoBottom()
	if event.Kind == session.EventDisconnected {
		m.dualPrompt.OnDisconnected()
		m.commands.SetItems(commandListItems(spp.DefaultModel(), nil, m.options.AllowUnsafe))
		return
	}
	m.evaluateDualPrompt(snap)
	m.commands.SetItems(commandListItems(snap.Model, snap.DualList, m.options.AllowUnsafe))
}

func (m Model) exportCmd() tea.Cmd {
	events := append([]trace.Event(nil), m.presenter.LastEvents...)
	comment := m.comment.Value()
	dir := m.options.CaptureDir
	if dir == "" {
		dir = "captures"
	}
	path := filepath.Join(dir, time.Now().Format("2006-01-02_15-04-05")+"_packets.json")
	return func() tea.Msg {
		if err := actions.ExportPackets(path, events, comment, m.options.LogRaw); err != nil {
			return errMsg(err)
		}
		return eventMsg(session.Event{Kind: session.EventPacketTX, Trace: trace.Event{Summary: "exported " + path}})
	}
}

func discoverCmd(mgr *connect.Manager) tea.Cmd {
	return func() tea.Msg {
		devices, err := mgr.Discover(context.Background())
		if err != nil {
			return errMsg(err)
		}
		return devicesMsg(devices)
	}
}

func connectCmd(mgr *connect.Manager, dev bt.Device) tea.Cmd {
	return func() tea.Msg {
		if err := mgr.Connect(context.Background(), dev); err != nil {
			return errMsg(err)
		}
		return eventMsg(session.Event{Kind: session.EventConnected, Device: dev})
	}
}

func waitEventCmd(events <-chan session.Event) tea.Cmd {
	return func() tea.Msg {
		event := <-events
		return eventMsg(event)
	}
}

func runScanCmd(ctx context.Context, s *session.Session, fields []string) tea.Cmd {
	return func() tea.Msg {
		if ctx == nil {
			ctx = context.Background()
		}
		if err := actions.ExecuteScan(ctx, s, fields); err != nil {
			return errMsg(err)
		}
		return eventMsg(session.Event{
			Kind:    session.EventProgress,
			Source:  "scan",
			Trigger: "scan finished " + strings.Join(fields[1:], " "),
		})
	}
}

func sendCommandCmd(s *session.Session, item presenter.Command, comment string) tea.Cmd {
	return func() tea.Msg {
		result := actions.Execute(s, item, actions.ExecOpts{
			Comment: comment,
			Source:  "tui",
		})
		if result.Err != nil {
			return errMsg(result.Err)
		}
		return nil
	}
}
