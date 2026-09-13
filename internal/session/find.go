package session

import (
	"context"
	"errors"
	"fmt"
	"time"

	"nothing_helper/internal/spp"
)

type EarbudState struct {
	spp.EarbudStatus
	UpdatedAt time.Time
}

func cloneEarbuds(src map[string]EarbudState) map[string]EarbudState {
	out := make(map[string]EarbudState, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}
func (s *Session) recordEarbuds(parsed spp.ParsedPacket) {
	if parsed.Earbuds == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.earbuds == nil {
		s.earbuds = map[string]EarbudState{}
	}
	now := time.Now()
	for side, state := range parsed.Earbuds {
		s.earbuds[side] = EarbudState{state, now}
	}
}

const findStatusMaxAge = 2 * time.Second

type findOperation struct {
	cancel context.CancelFunc
	done   chan struct{}
	err    error // published by closing done
}

// Explicit close must leave the transport open until FIND OFF completes.
// A stuck transport still gets torn down after the bounded grace period.
func (s *Session) stopFindLocked() error {
	active := s.activeFind
	if active == nil {
		return nil
	}
	select {
	case <-active.done:
		s.activeFind = nil
		return nil
	default:
	}
	active.cancel()
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	select {
	case <-active.done:
		s.activeFind = nil
		if active.err == context.Canceled {
			return nil
		}
		return active.err
	case <-timer.C:
		return fmt.Errorf("search stop timed out before transport close")
	}
}

// Always enforce these guards at the write boundary, even with --unsafe.
func (s *Session) validateFindLocked(payload []byte, now time.Time) error {
	if err := spp.ValidateFindPayload(payload); err != nil {
		return err
	}
	if payload[1] == 0 {
		return nil
	} // A stop must remain possible with missing sensor data.
	if !spp.SupportsFind(s.model) {
		return fmt.Errorf("поиск не поддерживается для этой модели")
	}
	side := "left"
	if payload[0] == 3 {
		side = "right"
	}
	state, ok := s.earbuds[side]
	if !ok || state.UpdatedAt.IsZero() || now.Sub(state.UpdatedAt) > findStatusMaxAge || now.Before(state.UpdatedAt) {
		return fmt.Errorf("нет свежих данных датчика; поиск не запущен")
	}
	if state.InEar {
		return fmt.Errorf("наушник в ухе; выньте его перед поиском")
	}
	if !state.Connected || state.InCase {
		return fmt.Errorf("наушник недоступен или находится в кейсе")
	}
	return nil
}

// FindEarbud plays one short search signal after a fresh sensor check.
// It stops on cancellation, insertion, stale data, or after ten seconds.
func (s *Session) FindEarbud(ctx context.Context, side string) (result error) {
	payload, err := spp.FindPayload(side, true)
	if err != nil {
		return err
	}
	s.mu.Lock()
	transport := s.transport
	supported := spp.SupportsFind(s.model)
	s.mu.Unlock()
	if transport == nil {
		return fmt.Errorf("наушники не подключены")
	}
	if !supported {
		return fmt.Errorf("поиск не поддерживается для этой модели")
	}
	// Keep reconnection/transport replacement out of the complete search sequence.
	s.connectMu.Lock()
	defer s.connectMu.Unlock()
	s.findMu.Lock()
	s.mu.Lock()
	same := s.transport == transport
	s.mu.Unlock()
	if !same {
		s.findMu.Unlock()
		return fmt.Errorf("соединение изменилось")
	}
	ctx, cancel := context.WithCancel(ctx)
	active := &findOperation{cancel: cancel, done: make(chan struct{})}
	s.activeFind = active
	s.findMu.Unlock()
	defer func() {
		cancel()
		active.err = result
		close(active.done)
	}()
	if err := ctx.Err(); err != nil {
		return err
	}
	requested := time.Now()
	if err := s.SendCommand(spp.CmdGetStatus, Meta{Source: "find", Trigger: "check in-ear sensor"}); err != nil {
		return err
	}
	check := time.NewTicker(50 * time.Millisecond)
	defer check.Stop()
	timeout := time.NewTimer(3 * time.Second)
	defer timeout.Stop()
	for {
		state := s.Snapshot().Earbuds[side]
		if !state.UpdatedAt.Before(requested) {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout.C:
			return fmt.Errorf("датчик не ответил; поиск не запущен")
		case <-check.C:
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	// Install stop before start: even a failed write may have reached the device.
	defer func() {
		s.mu.Lock()
		same := s.transport == transport
		s.mu.Unlock()
		if same {
			stop, _ := spp.FindPayload(side, false)
			if err := s.Send(spp.Packet{Cmd: spp.CmdFindEarbud, Payload: stop}, Meta{Source: "find", Trigger: "stop search"}); err != nil {
				result = errors.Join(result, fmt.Errorf("не удалось остановить сигнал: %w", err))
			}
		}
	}()
	if err := s.Send(spp.Packet{Cmd: spp.CmdFindEarbud, Payload: payload}, Meta{Source: "find", Trigger: "start search"}); err != nil {
		return err
	}
	duration := time.NewTimer(10 * time.Second)
	defer duration.Stop()
	poll := time.NewTicker(time.Second)
	defer poll.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-duration.C:
			return nil
		case <-poll.C:
			if err := s.SendCommand(spp.CmdGetStatus, Meta{Source: "find", Trigger: "monitor in-ear sensor"}); err != nil {
				return err
			}
		case <-check.C:
			s.mu.Lock()
			err := s.validateFindLocked(payload, time.Now())
			same := s.transport == transport
			s.mu.Unlock()
			if !same {
				return fmt.Errorf("соединение потеряно")
			}
			if err != nil {
				return err
			}
		}
	}
}
