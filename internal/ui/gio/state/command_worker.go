//go:build gio

package state

import (
	"tws_manager/internal/ui/actions"
	"tws_manager/internal/ui/presenter"
)

func (s *State) enqueueCommand(cmd presenter.Command) {
	select {
	case s.cmdQueue <- cmd:
	default:
		s.setErr("command queue full")
	}
}

func (s *State) commandWorker() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case cmd, ok := <-s.cmdQueue:
			if !ok {
				return
			}
			result := actions.Execute(s.session, cmd, actions.ExecOpts{
				Source: "gio",
			})
			if result.Err != nil {
				s.setErr(result.Err.Error())
				continue
			}
			if len(cmd.Fields) > 0 {
				s.refreshFeatureAfterSet(cmd.Fields)
			}
		}
	}
}
