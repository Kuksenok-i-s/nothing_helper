//go:build gio && darwin

package state

// configurePlatformHooks applies Darwin-specific UI hooks.
func (s *State) configurePlatformHooks() {
	// No sudo password modal on macOS; privileged RFCOMM is unavailable.
}
