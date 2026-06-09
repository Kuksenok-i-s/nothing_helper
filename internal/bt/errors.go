package bt

import (
	"errors"
	"fmt"
)

var (
	ErrRFCOMMBindFailed      = errors.New("rfcomm bind failed")
	ErrRFCOMMWaitFailed      = errors.New("rfcomm device wait failed")
	ErrRFCOMMOpenFailed      = errors.New("rfcomm open failed")
	ErrRFCOMMReviveFailed    = errors.New("rfcomm revive failed")
	ErrRFCOMMPermission      = errors.New("rfcomm permission denied")
	ErrRFCOMMNoChannel       = errors.New("no working rfcomm channel")
	ErrBluetoothctlInfo      = errors.New("bluetoothctl info failed")
	ErrInvalidBluetoothMAC   = errors.New("invalid bluetooth mac")
)

func wrapRFCOMMBind(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %w", ErrRFCOMMBindFailed, err)
}

func wrapRFCOMMWait(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %w", ErrRFCOMMWaitFailed, err)
}

func wrapRFCOMMOpen(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %w", ErrRFCOMMOpenFailed, err)
}

func wrapRFCOMMRevive(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %w", ErrRFCOMMReviveFailed, err)
}

func wrapRFCOMMPermission(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %w", ErrRFCOMMPermission, err)
}
