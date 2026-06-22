//go:build gio && linux

package state

import "tws_manager/internal/bt"

func (s *State) configurePlatformHooks() {
	bt.ConfigureSudoPasswordProvider(s.promptSudoPassword)
}
