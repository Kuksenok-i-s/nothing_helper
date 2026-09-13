package companion

import (
	"context"
	"fmt"
	"time"

	"nothing_helper/internal/audio"
	"nothing_helper/internal/spp"
)

type captureChecker interface {
	ActiveCapture(context.Context, string) (bool, error)
}

func (b sessionBackend) ActiveCapture(ctx context.Context, mac string) (bool, error) {
	return audio.HasActiveCaptureForMAC(ctx, mac)
}

func (c *Controller) requireCapture(mac string, generation uint64) error {
	if !c.sameConnection(mac, generation) {
		return fmt.Errorf("соединение изменилось; повторите действие")
	}
	checker, ok := c.backend.(captureChecker)
	if !ok {
		return fmt.Errorf("проверка активного микрофона недоступна")
	}
	ctx, cancel := context.WithTimeout(c.ctx, 2*time.Second)
	defer cancel()
	active, err := checker.ActiveCapture(ctx, mac)
	if err != nil {
		return fmt.Errorf("не удалось проверить микрофон: %w; начните звонок или запись с микрофоном Ear (3)", err)
	}
	if !active {
		return fmt.Errorf("сначала начните звонок или запись с микрофоном Ear (3), затем включите Walkie Talkie")
	}
	return nil
}

func (c *Controller) WalkieTalkie(value string) bool {
	snap := c.backend.Snapshot()
	if !snap.Connected || !spp.ModelSupportsFeature(snap.Model, "walkie-talkie") || (value != "on" && value != "off") {
		return false
	}
	mac, generation := c.token()
	return c.submit("Переключение Walkie Talkie", func() error {
		if value == "on" {
			if err := c.requireCapture(mac, generation); err != nil {
				return err
			}
			if err := c.writeMicMode(mac, generation, "super-mic", "on"); err != nil {
				return err
			}
			if err := c.requireCapture(mac, generation); err != nil {
				return err
			}
		}
		return c.writeMicMode(mac, generation, "walkie-talkie", value)
	})
}

func (c *Controller) writeMicMode(mac string, generation uint64, feature, value string) error {
	if !c.sameConnection(mac, generation) {
		return fmt.Errorf("соединение изменилось; повторите действие")
	}
	if err := c.executeAndReadback(mac, generation, []string{feature, "set", value}, []string{feature, "get"}); err != nil {
		return err
	}
	// Session invalidates the old value before sending a GET/SET. Only a
	// decoded response/event restores it; acknowledgement is not state.
	want := fmt.Sprintf("enabled=%t", value == "on")
	timeout := time.NewTimer(3 * time.Second)
	defer timeout.Stop()
	poll := time.NewTicker(50 * time.Millisecond)
	defer poll.Stop()
	for {
		if !c.sameConnection(mac, generation) {
			return fmt.Errorf("соединение изменилось; состояние режима не подтверждено")
		}
		if c.backend.Snapshot().Config[feature] == want {
			return nil
		}
		select {
		case <-c.ctx.Done():
			return c.ctx.Err()
		case <-timeout.C:
			return fmt.Errorf("%s: наушники не подтвердили переключение; обновите состояние", feature)
		case <-poll.C:
		}
	}
}
