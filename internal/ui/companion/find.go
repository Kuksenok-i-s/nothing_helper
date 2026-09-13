package companion

import (
	"context"
	"fmt"
	"time"

	"tws_manager/internal/session"
	"tws_manager/internal/spp"
)

type statusReader interface{ Status() error }
type earbudFinder interface {
	Find(context.Context, string) error
}

func (b sessionBackend) Status() error {
	return b.s.SendCommand(spp.CmdGetStatus, session.Meta{Source: "companion", Trigger: "refresh wear sensor"})
}
func (b sessionBackend) Find(ctx context.Context, side string) error {
	return b.s.FindEarbud(ctx, side)
}
func (c *Controller) RefreshWear() {
	reader, ok := c.backend.(statusReader)
	if !ok {
		return
	}
	mac, generation := c.token()
	c.submit("Обновление датчиков", func() error {
		if !c.sameConnection(mac, generation) {
			return nil
		}
		if err := reader.Status(); err != nil {
			return err
		}
		if spp.ModelSupportsFeature(c.backend.Snapshot().Model, "walkie-talkie") {
			for _, feature := range []string{"super-mic", "walkie-talkie"} {
				if !c.sameConnection(mac, generation) {
					return nil
				}
				if err := c.backend.Execute([]string{feature, "get"}); err != nil {
					return err
				}
			}
		}
		return nil
	})
}
func (c *Controller) Find(side string) {
	if side != "left" && side != "right" {
		return
	}
	finder, ok := c.backend.(earbudFinder)
	if !ok {
		c.status("Поиск недоступен")
		return
	}
	mac, generation := c.token()
	ctx, cancel := context.WithCancel(c.ctx)
	c.mu.Lock()
	if c.findCancel != nil {
		c.mu.Unlock()
		cancel()
		return
	}
	c.findCancel = cancel
	c.findSide = side
	c.mu.Unlock()
	clear := func() {
		cancel()
		c.mu.Lock()
		c.findCancel = nil
		c.findSide = ""
		c.mu.Unlock()
		c.changedState()
	}
	if !c.submit("Поиск наушника", func() error {
		defer clear()
		if ctx.Err() != nil {
			return nil
		}
		if !c.sameConnection(mac, generation) {
			return fmt.Errorf("соединение изменилось; повторите поиск")
		}
		return finder.Find(ctx, side)
	}) {
		clear()
	}
}
func (c *Controller) StopFind() {
	c.mu.Lock()
	cancel := c.findCancel
	c.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// WearKnown deliberately distinguishes missing/stale readings from out-of-ear.
func WearKnown(s session.Snapshot, side string, now time.Time) (session.EarbudState, bool) {
	state, ok := s.Earbuds[side]
	age := now.Sub(state.UpdatedAt)
	return state, s.Connected && ok && state.Connected && !state.UpdatedAt.IsZero() && age >= 0 && age <= 30*time.Second
}
func CaseCharge(s session.Snapshot) (int, bool) {
	b, ok := s.Batteries["case"]
	return b.Percent, ok && s.Connected && b.Percent >= 0 && b.Percent <= 100
}
