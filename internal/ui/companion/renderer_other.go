//go:build gio && (!linux || !nowayland)

package companion

func retryRenderer(err error, _ func()) error { return err }
