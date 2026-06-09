//go:build gio

package state

import (
	"context"
	"errors"
	"sync"

	"gioui.org/app"
	"gioui.org/widget"

	"tws_manager/internal/bt"
	"tws_manager/internal/connect"
	"tws_manager/internal/dualpolicy"
	"tws_manager/internal/session"
	"tws_manager/internal/spp"
	"tws_manager/internal/trace"
	"tws_manager/internal/ui/gio/config"
	"tws_manager/internal/ui/presenter"
)

// Tab identifies the active content panel.
type Tab int

const (
	TabDevices Tab = iota
	TabControl
	TabLog
)

func (t Tab) String() string {
	switch t {
	case TabDevices:
		return "Devices"
	case TabControl:
		return "Control"
	default:
		return "Log"
	}
}

// Action identifies a UI action.
type Action string

const (
	ActionDiscover   Action = "discover"
	ActionBind       Action = "bind"
	ActionConnect    Action = "connect"
	ActionDisconnect Action = "disconnect"
	ActionBattery    Action = "battery"
	ActionExport     Action = "export"
	ActionAuto       Action = "auto"
)

// State holds Gio application state and widget handles.
type State struct {
	ctx         context.Context
	window      *app.Window
	manager     *connect.Manager
	session     *session.Session
	presenter   *presenter.State
	events      <-chan session.Event
	allowUnsafe   bool
	autoReconnect bool
	captureDir    string
	logRaw        bool

	mu          sync.Mutex
	devices     []bt.Device
	selectedDev int
	activeTab   Tab
	commands    []presenter.Command
	comment     string
	devClicks   []widget.Clickable
	dualClicks  []widget.Clickable
	sudoPrompt  string
	sudoReply   chan sudoPasswordResult

	pcPrimaryMode     dualpolicy.Mode
	hostMAC           string
	hostMACLoaded     bool
	hostMACErr        string
	dualPending       spp.DualDevice
	dualPendingOK     bool
	dualPromptVisible bool
	userInteracted    bool

	toggleBools   map[string]*widget.Bool
	togglePending map[string]bool
	toggleWant    map[string]bool

	DiscoverBtn    widget.Clickable
	AutoBtn        widget.Clickable
	BindBtn        widget.Clickable
	ConnectBtn     widget.Clickable
	DisconnectBtn  widget.Clickable
	ExportBtn      widget.Clickable
	CopyBtn        widget.Clickable
	RefreshBattery widget.Clickable
	DualRefreshBtn widget.Clickable
	StatusList     widget.List
	GetList        widget.List
	SetList        widget.List
	HumanList      widget.List
	ExportList     widget.List
	RawList        widget.List
	RawHexList     widget.List
	RawHexToggle   widget.Bool
	SudoPassword   widget.Editor
	SudoSubmit     widget.Clickable
	SudoCancel     widget.Clickable
	DualAcceptBtn  widget.Clickable
	DualDeclineBtn widget.Clickable
	TabDevices     widget.Clickable
	TabControl     widget.Clickable
	TabLog         widget.Clickable
	CmdButtons     []widget.Clickable
}

// Snapshot is an immutable view of State for rendering.
type Snapshot struct {
	Session    session.Snapshot
	Status     string
	ErrText    string
	LogText    string
	Devices    []bt.Device
	Selected   int
	Tab        Tab
	Commands   []presenter.Command
	Clicks     []widget.Clickable
	Packets    int
	Recent     []trace.Event
	RawPackets []trace.Event
	SudoPrompt      string
	DualPrompt      string
	DualPromptShown bool
}

type sudoPasswordResult struct {
	password string
	err      error
}

// New creates application state from options.
func New(ctx context.Context, w *app.Window, opts config.Options, sess *session.Session) *State {
	s := &State{
		ctx:           ctx,
		window:        w,
		manager:       opts.Manager,
		session:       sess,
		presenter:     presenter.NewState(opts.LogRaw),
		events:        sess.Subscribe(),
		allowUnsafe:   opts.AllowUnsafe,
		autoReconnect: opts.AutoConnect,
		captureDir:    opts.CaptureDir,
		logRaw:        opts.LogRaw,
		activeTab:     TabControl,
		pcPrimaryMode: opts.PCPrimary,
	}
	s.presenter.AutoReconnect = opts.AutoConnect
	s.SudoPassword.SingleLine = true
	s.SudoPassword.Submit = true
	s.SudoPassword.Mask = '*'
	bt.ConfigureSudoPasswordProvider(s.promptSudoPassword)
	snap := sess.Snapshot()
	s.commands = presenter.BuildCommands(snap.Model, snap.DualList, opts.AllowUnsafe)
	s.syncCmdButtons()
	if opts.InitialDevice.MAC != "" {
		go s.connectInitial(opts.InitialDevice)
	} else if opts.AutoConnect {
		go s.runAutoReconnectLoop()
	}
	return s
}

func (s *State) promptSudoPassword(prompt string) (string, error) {
	if prompt == "" {
		prompt = "Administrator password is required."
	}
	reply := make(chan sudoPasswordResult, 1)
	s.mu.Lock()
	if s.sudoReply != nil {
		s.mu.Unlock()
		return "", errors.New("sudo password prompt already active")
	}
	s.sudoPrompt = prompt
	s.sudoReply = reply
	s.SudoPassword.SetText("")
	s.mu.Unlock()
	s.invalidate()

	select {
	case result := <-reply:
		return result.password, result.err
	case <-s.ctx.Done():
		return "", s.ctx.Err()
	}
}

func (s *State) SubmitSudoPassword() {
	s.mu.Lock()
	reply := s.sudoReply
	password := s.SudoPassword.Text()
	s.sudoPrompt = ""
	s.sudoReply = nil
	s.SudoPassword.SetText("")
	s.mu.Unlock()
	if reply != nil {
		reply <- sudoPasswordResult{password: password}
	}
	s.invalidate()
}

func (s *State) CancelSudoPassword() {
	s.mu.Lock()
	reply := s.sudoReply
	s.sudoPrompt = ""
	s.sudoReply = nil
	s.SudoPassword.SetText("")
	s.mu.Unlock()
	if reply != nil {
		reply <- sudoPasswordResult{err: errors.New("sudo password prompt cancelled")}
	}
	s.invalidate()
}

func (s *State) Snapshot() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return Snapshot{
		Session:    s.session.Snapshot(),
		Status:     s.presenter.Status,
		ErrText:    s.presenter.Err,
		LogText:    s.presenter.LogText(),
		Devices:    append([]bt.Device(nil), s.devices...),
		Selected:   s.selectedDev,
		Tab:        s.activeTab,
		Commands:   append([]presenter.Command(nil), s.commands...),
		Clicks:     s.devClicks,
		Packets:    len(s.presenter.LastEvents),
		Recent:     recentEvents(s.presenter.LastEvents, 3),
		RawPackets: chronologicalEvents(s.presenter.LastEvents, 16),
		SudoPrompt:      s.sudoPrompt,
		DualPromptShown: s.dualPromptVisible && s.dualPendingOK,
		DualPrompt: func() string {
			if !s.dualPromptVisible || !s.dualPendingOK {
				return ""
			}
			return dualpolicy.PromptText(s.dualPending)
		}(),
	}
}

// chronologicalEvents returns up to n most-recent trace events, oldest first,
// so TX→RX pairs read naturally top to bottom.
func chronologicalEvents(events []trace.Event, n int) []trace.Event {
	if len(events) > n {
		events = events[len(events)-n:]
	}
	return append([]trace.Event(nil), events...)
}

// recentEvents returns up to n most-recent trace events, newest first.
func recentEvents(events []trace.Event, n int) []trace.Event {
	if len(events) > n {
		events = events[len(events)-n:]
	}
	out := make([]trace.Event, 0, len(events))
	for i := len(events) - 1; i >= 0; i-- {
		out = append(out, events[i])
	}
	return out
}

func (s *State) SetTab(tab Tab) {
	s.activeTab = tab
	s.invalidate()
}

func (s *State) SelectDevice(index int) {
	s.mu.Lock()
	s.selectedDev = index
	s.mu.Unlock()
	s.invalidate()
}

// DeviceClick returns a stable clickable for device list row i.
func (s *State) DeviceClick(i int) *widget.Clickable {
	s.mu.Lock()
	defer s.mu.Unlock()
	for len(s.devClicks) <= i {
		s.devClicks = append(s.devClicks, widget.Clickable{})
	}
	return &s.devClicks[i]
}

// DualClick returns a stable clickable for dual-device row i. The slice only
// grows so handles stay valid across frames even as the dual list changes.
func (s *State) DualClick(i int) *widget.Clickable {
	s.mu.Lock()
	defer s.mu.Unlock()
	for len(s.dualClicks) <= i {
		s.dualClicks = append(s.dualClicks, widget.Clickable{})
	}
	return &s.dualClicks[i]
}

// ToggleBool returns a stable widget.Bool handle for an on/off feature switch.
func (s *State) ToggleBool(feature string) *widget.Bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.toggleBools == nil {
		s.toggleBools = map[string]*widget.Bool{}
	}
	if s.toggleBools[feature] == nil {
		s.toggleBools[feature] = &widget.Bool{}
	}
	return s.toggleBools[feature]
}

// SyncToggle reconciles a switch with the live device state. While a user toggle
// is in flight (pending), it holds the requested value until the device confirms
// it via a fresh GET; otherwise it mirrors the reported state (indicator role).
// It returns the value the switch should display this frame.
func (s *State) SyncToggle(feature string, deviceOn bool) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.togglePending == nil {
		s.togglePending = map[string]bool{}
		s.toggleWant = map[string]bool{}
	}
	if s.togglePending[feature] {
		if deviceOn == s.toggleWant[feature] {
			s.togglePending[feature] = false
			return deviceOn
		}
		return s.toggleWant[feature]
	}
	return deviceOn
}

// MarkTogglePending records a user's requested switch value until the device
// reports it back, preventing the indicator from snapping back mid-flight.
func (s *State) MarkTogglePending(feature string, want bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.togglePending == nil {
		s.togglePending = map[string]bool{}
		s.toggleWant = map[string]bool{}
	}
	s.togglePending[feature] = true
	s.toggleWant[feature] = want
}

// SetStatus updates the transient status line (used for UI feedback).
func (s *State) SetStatus(msg string) {
	s.mu.Lock()
	s.presenter.Status = msg
	s.mu.Unlock()
	s.invalidate()
}

func (s *State) syncCmdButtons() {
	if len(s.CmdButtons) != len(s.commands) {
		s.CmdButtons = make([]widget.Clickable, len(s.commands))
	}
}

// SetWindow rebinds the Gio window after hide-to-tray recreation.
func (s *State) SetWindow(w *app.Window) {
	s.mu.Lock()
	s.window = w
	s.mu.Unlock()
	s.invalidate()
}

func (s *State) invalidate() {
	if s.window != nil {
		s.window.Invalidate()
	}
}
