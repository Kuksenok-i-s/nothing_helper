// Package companion implements the compact Nothing desktop companion.
// Device operations are independent of the rendering toolkit and serialized.
package companion

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"nothing_helper/internal/bt"
	"nothing_helper/internal/connect"
	"nothing_helper/internal/dualpolicy"
	"nothing_helper/internal/session"
	"nothing_helper/internal/spp"
	"nothing_helper/internal/trace"
	"nothing_helper/internal/ui/actions"
	"nothing_helper/internal/ui/dualprompt"
	"nothing_helper/internal/ui/presenter"
)

type Options struct {
	Controller    *Controller
	Manager       *connect.Manager
	CaptureDir    string
	LogRaw        bool
	AutoConnect   bool
	InitialDevice bt.Device
	PCPrimary     dualpolicy.Mode
	HideToTray    bool
	ShowCh        <-chan struct{}
	OnQuit        func()
}

// Backend keeps all writes behind the session's feature and safety checks.
type Backend interface {
	Snapshot() session.Snapshot
	Subscribe() <-chan session.Event
	Execute([]string) error
	Battery() error
}
type sessionBackend struct{ s *session.Session }

func (b sessionBackend) Snapshot() session.Snapshot      { return b.s.Snapshot() }
func (b sessionBackend) Subscribe() <-chan session.Event { return b.s.Subscribe() }
func (b sessionBackend) Execute(fields []string) error {
	return actions.Execute(b.s, presenter.Command{Fields: fields}, actions.ExecOpts{Source: "companion"}).Err
}
func (b sessionBackend) Battery() error {
	return b.s.SendCommand(spp.CmdGetBattery, session.Meta{Source: "companion", Trigger: "refresh battery"})
}

type Snapshot struct {
	Session                                   session.Snapshot
	Devices                                   []bt.Device
	Status, Error, PasswordPrompt, DualPrompt string
	FindSide                                  string
	Busy                                      bool
	Logs                                      []string
}
type work struct {
	label string
	run   func() error
}
type Controller struct {
	mu             sync.Mutex
	ctx            context.Context
	opts           Options
	backend        Backend
	queue          chan work
	changed        chan struct{}
	presenter      *presenter.State
	devices        []bt.Device
	pending        int
	generation     uint64
	auto           bool
	target         bt.Device
	passwordPrompt string
	passwordReply  chan string
	dual           dualprompt.Controller
	findCancel     context.CancelFunc
	findSide       string
}

func NewController(ctx context.Context, opts Options) *Controller {
	return newController(ctx, opts, sessionBackend{opts.Manager.Session()})
}
func newController(ctx context.Context, opts Options, backend Backend) *Controller {
	return &Controller{ctx: ctx, opts: opts, backend: backend, queue: make(chan work, 8), changed: make(chan struct{}, 1), presenter: presenter.NewState(opts.LogRaw), auto: opts.AutoConnect, target: opts.InitialDevice, dual: dualprompt.Controller{Mode: opts.PCPrimary, HostLoaded: true}}
}
func (c *Controller) Changed() <-chan struct{} { return c.changed }
func (c *Controller) changedState() {
	select {
	case c.changed <- struct{}{}:
	default:
	}
}
func (c *Controller) Snapshot() Snapshot {
	snap := c.backend.Snapshot()
	c.mu.Lock()
	defer c.mu.Unlock()
	return Snapshot{FindSide: c.findSide, Session: snap, Devices: append([]bt.Device(nil), c.devices...), Status: c.presenter.Status, Error: c.presenter.Err, Busy: c.pending > 0, Logs: append([]string(nil), c.presenter.LogLines...), PasswordPrompt: c.passwordPrompt, DualPrompt: c.dual.PromptLine()}
}
func (c *Controller) Start() {
	events := c.backend.Subscribe()
	configurePasswordProvider(c)
	go c.worker()
	go func() {
		mac, err := dualpolicy.HostAdapterMAC()
		c.mu.Lock()
		c.dual.HostLoaded = true
		c.dual.HostMAC = mac
		if err != nil {
			c.dual.HostErr = err.Error()
		}
		c.mu.Unlock()
		for {
			select {
			case <-c.ctx.Done():
				return
			case ev, ok := <-events:
				if !ok {
					return
				}
				snap := c.backend.Snapshot()
				c.mu.Lock()
				c.presenter.ApplyEvent(ev, snap)
				if ev.Kind == session.EventConnected || ev.Kind == session.EventDisconnected {
					c.generation++
				}
				c.dual.OnSnapshot(snap)
				c.mu.Unlock()
				c.changedState()
			}
		}
	}()
	if c.opts.InitialDevice.MAC != "" {
		c.ConnectDevice(c.opts.InitialDevice)
	} else if c.opts.AutoConnect {
		c.Connect()
	}
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-c.ctx.Done():
				return
			case <-ticker.C:
				c.mu.Lock()
				auto, idle := c.auto, c.pending == 0
				c.mu.Unlock()
				if auto && idle && !c.backend.Snapshot().Connected {
					c.Connect()
				} else if idle && c.backend.Snapshot().Connected {
					c.RefreshWear()
				}
			}
		}
	}()
}
func (c *Controller) submit(label string, fn func() error) bool {
	c.mu.Lock()
	if c.ctx.Err() != nil {
		c.mu.Unlock()
		return false
	}
	c.pending++
	select {
	case c.queue <- work{label, fn}:
		c.mu.Unlock()
		c.changedState()
		return true
	default:
		c.pending--
		c.presenter.Err = "Операция уже выполняется. Повторите позже."
		c.mu.Unlock()
		c.changedState()
		return false
	}
}
func (c *Controller) worker() {
	for {
		select {
		case <-c.ctx.Done():
			return
		case job := <-c.queue:
			if c.ctx.Err() != nil {
				return
			}
			c.mu.Lock()
			c.presenter.Status = job.label
			c.presenter.Err = ""
			c.mu.Unlock()
			c.changedState()
			err := job.run()
			c.mu.Lock()
			c.pending--
			if err != nil {
				c.presenter.Err = err.Error()
			} else if c.presenter.Status == job.label {
				c.presenter.Status = job.label + " · готово"
			}
			c.mu.Unlock()
			c.changedState()
		}
	}
}
func (c *Controller) status(s string) {
	c.mu.Lock()
	c.presenter.Status = s
	c.mu.Unlock()
	c.changedState()
}
func (c *Controller) Discover() {
	c.submit("Поиск устройств", func() error {
		devs, err := c.opts.Manager.Discover(c.ctx)
		if err != nil {
			return err
		}
		c.mu.Lock()
		c.devices = devs
		c.mu.Unlock()
		return nil
	})
}
func (c *Controller) Connect() {
	c.mu.Lock()
	target := c.target
	c.mu.Unlock()
	c.submit("Подключение", func() error {
		if target.MAC != "" {
			return c.opts.Manager.SwitchTo(c.ctx, target)
		}
		return c.opts.Manager.ConnectBest(c.ctx, c.status)
	})
}
func (c *Controller) ConnectDevice(dev bt.Device) {
	c.mu.Lock()
	c.target = dev
	c.mu.Unlock()
	c.Connect()
}
func (c *Controller) Disconnect() {
	c.StopFind()
	c.mu.Lock()
	c.auto = false
	c.mu.Unlock()
	c.submit("Отключение", c.opts.Manager.Disconnect)
}
func (c *Controller) SetAuto(on bool) { c.mu.Lock(); c.auto = on; c.mu.Unlock() }
func (c *Controller) token() (string, uint64) {
	snap := c.backend.Snapshot()
	c.mu.Lock()
	defer c.mu.Unlock()
	return snap.Device.MAC, c.generation
}
func (c *Controller) sameConnection(mac string, generation uint64) bool {
	snap := c.backend.Snapshot()
	c.mu.Lock()
	defer c.mu.Unlock()
	return snap.Connected && snap.Device.MAC == mac && generation == c.generation
}
func (c *Controller) Feature(feature, value string) bool {
	if feature == "walkie-talkie" {
		return c.WalkieTalkie(value)
	}
	switch feature {
	case "anc", "eq", "lag", "spatial", "dual":
	default:
		return false
	}
	snap := c.backend.Snapshot()
	if !snap.Connected || !spp.ModelSupportsFeature(snap.Model, feature) {
		return false
	}
	mac, generation := c.token()
	return c.submit("Изменение настройки", func() error {
		if !c.sameConnection(mac, generation) {
			return fmt.Errorf("соединение изменилось; повторите действие")
		}
		return c.executeAndReadback(mac, generation, []string{feature, "set", value}, []string{feature, "get"})
	})
}
func (c *Controller) Refresh() {
	mac, generation := c.token()
	c.submit("Обновление состояния", func() error {
		if !c.sameConnection(mac, generation) {
			return nil
		}
		if reader, ok := c.backend.(statusReader); ok {
			if err := reader.Status(); err != nil {
				return err
			}
		}
		if err := c.backend.Battery(); err != nil {
			return err
		}
		for _, feature := range []string{"anc", "eq", "lag", "spatial", "dual", "super-mic", "walkie-talkie"} {
			if !c.sameConnection(mac, generation) {
				return nil
			}
			if spp.ModelSupportsFeature(c.backend.Snapshot().Model, feature) {
				if err := c.backend.Execute([]string{feature, "get"}); err != nil {
					return err
				}
			}
		}
		return nil
	})
}
func (c *Controller) DualPeer(dev spp.DualDevice) {
	mac, generation := c.token()
	c.submit("Переключение источника", func() error {
		if !c.sameConnection(mac, generation) {
			return fmt.Errorf("соединение изменилось")
		}
		action := "connect"
		if dev.Connected {
			action = "disconnect"
		}
		return c.executeAndReadback(mac, generation, []string{"dual", action, dev.MAC}, []string{"dual", "list"})
	})
}
func (c *Controller) RefreshPeers() {
	mac, generation := c.token()
	c.submit("Обновление источников", func() error {
		if !c.sameConnection(mac, generation) {
			return nil
		}
		return c.backend.Execute([]string{"dual", "list"})
	})
}

// A follow-up GET is connection-scoped and cancellable; the view continues to
// show confirmed state while the device processes the SET.
func (c *Controller) executeAndReadback(mac string, generation uint64, write, read []string) error {
	if err := c.backend.Execute(write); err != nil {
		return err
	}
	timer := time.NewTimer(350 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-c.ctx.Done():
		return c.ctx.Err()
	case <-timer.C:
	}
	if !c.sameConnection(mac, generation) {
		return nil
	}
	return c.backend.Execute(read)
}
func (c *Controller) Export() {
	c.submit("Экспорт журнала", func() error {
		c.mu.Lock()
		events := append([]trace.Event(nil), c.presenter.LastEvents...)
		c.mu.Unlock()
		dir := c.opts.CaptureDir
		if dir == "" {
			dir = "captures"
		}
		path := filepath.Join(dir, time.Now().Format("2006-01-02_15-04-05.000000000")+"_packets.json")
		if err := actions.ExportPackets(path, events, "Nothing companion", c.opts.LogRaw); err != nil {
			return err
		}
		c.status("Сохранено: " + path)
		return nil
	})
}
func (c *Controller) Interaction() { c.mu.Lock(); c.dual.OnInteraction(); c.mu.Unlock() }
func (c *Controller) ResolveDual(accept bool) {
	c.mu.Lock()
	if !accept {
		c.dual.Decline()
		c.mu.Unlock()
		c.changedState()
		return
	}
	fields, err := c.dual.AcceptFields()
	c.mu.Unlock()
	if err != nil {
		c.status(err.Error())
		return
	}
	mac, generation := c.token()
	c.submit("Переключение на компьютер", func() error {
		if !c.sameConnection(mac, generation) {
			return fmt.Errorf("соединение изменилось")
		}
		return c.executeAndReadback(mac, generation, fields, []string{"dual", "list"})
	})
}
func (c *Controller) Password(prompt string) (string, error) {
	reply := make(chan string, 1)
	c.mu.Lock()
	c.passwordPrompt = prompt
	c.passwordReply = reply
	c.mu.Unlock()
	c.changedState()
	defer func() { c.mu.Lock(); c.passwordPrompt = ""; c.passwordReply = nil; c.mu.Unlock(); c.changedState() }()
	select {
	case <-c.ctx.Done():
		return "", c.ctx.Err()
	case password := <-reply:
		if password == "" {
			return "", fmt.Errorf("авторизация отменена")
		}
		return password, nil
	}
}
func (c *Controller) AnswerPassword(password string) {
	c.mu.Lock()
	reply := c.passwordReply
	c.mu.Unlock()
	if reply != nil {
		select {
		case reply <- password:
		default:
		}
	}
}
func DeviceName(s session.Snapshot) string {
	if s.Device.Name != "" {
		return s.Device.Name
	}
	if s.Connected && s.Model.Product != "" {
		return s.Model.Product
	}
	return "Nothing / CMF"
}
func ANCMode(config string) string {
	switch {
	case strings.Contains(config, "transparency"):
		return "transparency"
	case strings.Contains(config, "off"):
		return "off"
	case strings.Contains(config, "anc") || strings.Contains(config, "adaptive") || strings.Contains(config, "mode=high") || strings.Contains(config, "mode=mid") || strings.Contains(config, "mode=low"):
		return "strong"
	}
	return ""
}
