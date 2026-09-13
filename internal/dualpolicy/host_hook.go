package dualpolicy

var hostAdapterMACHook func() (string, error)

// SetHostAdapterMACHook overrides HostAdapterMAC in tests. Pass nil to restore default.
func SetHostAdapterMACHook(fn func() (string, error)) {
	hostAdapterMACHook = fn
}
