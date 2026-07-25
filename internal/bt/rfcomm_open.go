//go:build linux

package bt

import (
	"fmt"
	"os"
	"time"

	"tws_manager/internal/security"
)

var (
	rfcommOpenFile          = openFileWithTimeout
	ensureRFCOMMOpenHook    func(string, int, os.FileMode) (*os.File, error)
	rfcommWaitForDevice     = waitForDevice
	rfcommEnsureAccess      = EnsureRFCOMMDeviceAccess
	rfcommPrivilegedBind    = privilegedRFCCOMMBind
	rfcommPrivilegedRelease = privilegedRFCOMMRelease
	rfcommReviveDevice      = ReviveRFCOMMDevice
	rfcommBindDevice        = BindRFCOMMDevice
)

// SetRFCOMMHooks overrides RFCOMM bind/open hooks in tests. Pass nil to restore defaults.
func SetRFCOMMHooks(openFn func(string, time.Duration) (*os.File, error), bindFn func(string, string, int) error) func() {
	oldOpen, oldBind := rfcommOpenFile, rfcommBindDevice
	if openFn != nil {
		rfcommOpenFile = openFn
	}
	if bindFn != nil {
		rfcommBindDevice = bindFn
	}
	return func() {
		rfcommOpenFile, rfcommBindDevice = oldOpen, oldBind
	}
}

func validateOpenRFCOMMParams(device, address string, channel int) (string, string, int, error) {
	devPath, err := security.ValidateRFCOMMDevice(device)
	if err != nil {
		return "", "", 0, err
	}
	if address != "" {
		if address, err = security.NormalizeMAC(address); err != nil {
			return "", "", 0, err
		}
	}
	if err := security.ValidateChannel(channel); err != nil {
		return "", "", 0, err
	}
	return devPath, address, channel, nil
}

func openRFCOMMAfterPermissionFix(device string, progress RFCOMMProgress) (*os.File, error) {
	report(progress, "fixing RFCOMM permissions")
	if accessErr := rfcommEnsureAccess(device); accessErr != nil {
		return nil, wrapRFCOMMPermission(accessErr)
	}
	f, retryErr := rfcommOpenFile(device, 5*time.Second)
	if retryErr != nil {
		return nil, wrapRFCOMMOpen(retryErr)
	}
	return f, nil
}

func openRFCOMMAfterRevive(device, address string, channel int, progress RFCOMMProgress, openErr error) (*os.File, error) {
	report(progress, "recovering stale RFCOMM device")
	if reviveErr := rfcommReviveDevice(device, address, channel, progress); reviveErr != nil {
		return nil, wrapRFCOMMRevive(fmt.Errorf("open %q: %w; revive failed: %w", device, openErr, reviveErr))
	}
	f, retryErr := rfcommOpenFile(device, 5*time.Second)
	if retryErr != nil {
		return nil, wrapRFCOMMOpen(fmt.Errorf("open %q after revive: %w", device, retryErr))
	}
	return f, nil
}

func createBindAndOpenRFCOMM(device, address string, channel int, progress RFCOMMProgress) (*os.File, error) {
	report(progress, "creating RFCOMM device")
	if err := rfcommBindDevice(device, address, channel); err != nil {
		return nil, wrapRFCOMMBind(fmt.Errorf("create %q: %w", device, err))
	}
	if err := rfcommWaitForDevice(device, 2*time.Second); err != nil {
		return nil, wrapRFCOMMWait(fmt.Errorf("wait for %q after rfcomm bind: %w", device, err))
	}
	f, err := rfcommOpenFile(device, 5*time.Second)
	if err != nil {
		return nil, wrapRFCOMMOpen(fmt.Errorf("open %q after rfcomm bind: %w", device, err))
	}
	return f, nil
}

func bindRFCOMMPrivileged(device, num, address string, channel int, plainErr error) error {
	if privErr := rfcommPrivilegedBind(num, address, channel); privErr != nil {
		return wrapRFCOMMBind(fmt.Errorf("%w; privileged fallback failed: %w", plainErr, privErr))
	}
	if err := rfcommWaitForDevice(device, 3*time.Second); err != nil {
		return wrapRFCOMMWait(fmt.Errorf("sudo rfcomm bind %s succeeded but %s was not created: %w", num, device, err))
	}
	if err := rfcommEnsureAccess(device); err != nil {
		return wrapRFCOMMPermission(err)
	}
	return nil
}
