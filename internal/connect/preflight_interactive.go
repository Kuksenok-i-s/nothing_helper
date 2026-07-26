package connect

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"tws_manager/internal/bt"
)

var preflightTestHooksVar *preflightTestHooks

type preflightTestHooks struct {
	rfcommExists func(m *Manager) (bool, error)
	bind         func(m *Manager, ctx context.Context, dev bt.Device) error
}

func preflightRFCOMMExists(m *Manager) (bool, error) {
	if preflightTestHooksVar != nil && preflightTestHooksVar.rfcommExists != nil {
		return preflightTestHooksVar.rfcommExists(m)
	}
	return m.RFCOMMExists()
}

func preflightBind(m *Manager, ctx context.Context, dev bt.Device) error {
	if preflightTestHooksVar != nil && preflightTestHooksVar.bind != nil {
		return preflightTestHooksVar.bind(m, ctx, dev)
	}
	return m.Bind(ctx, dev)
}

type PreflightIO struct {
	Discover func() ([]bt.Device, error)
	ReadLine func() (string, error)
	Confirm  func(prompt string) bool
	Printf   func(format string, args ...any)
}

// DefaultPreflightIO wires stdin/stdout discovery for CLI usage.
func DefaultPreflightIO() PreflightIO {
	return PreflightIO{
		Discover: bt.Discover,
		ReadLine: defaultReadLine,
		Confirm:  defaultConfirm,
		Printf:   func(format string, args ...any) { fmt.Printf(format, args...) },
	}
}

func defaultReadLine() (string, error) {
	if defaultReadLineHook != nil {
		return defaultReadLineHook()
	}
	reader := bufio.NewReader(os.Stdin)
	return reader.ReadString('\n')
}

var defaultReadLineHook func() (string, error)

func defaultConfirm(prompt string) bool {
	fmt.Printf("%s [y/N] ", prompt)
	line, err := defaultReadLine()
	if err != nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes", "д", "да":
		return true
	default:
		return false
	}
}

// SelectDeviceForRFCOMM resolves a device from --addr or an interactive list.
func SelectDeviceForRFCOMM(address string, channel int, io PreflightIO) (bt.Device, bool, error) {
	if dev, ok, err := DeviceFromPreflightAddress(address, channel); ok || err != nil {
		return dev, ok, err
	}
	return selectDiscoveredDevice(channel, io)
}

func selectDiscoveredDevice(channel int, io PreflightIO) (bt.Device, bool, error) {
	if io.Discover == nil {
		return bt.Device{}, false, fmt.Errorf("discover is required")
	}
	if io.Printf == nil {
		io.Printf = func(string, ...any) {}
	}

	devices, ok := discoverPreflightDevices(io)
	if !ok {
		return bt.Device{}, false, nil
	}
	printDiscoveredDeviceList(io, devices)
	return chooseDiscoveredDevice(io, devices, channel)
}

func discoverPreflightDevices(io PreflightIO) ([]bt.Device, bool) {
	devices, err := io.Discover()
	if err != nil {
		io.Printf("Could not discover devices: %v\n", err)
		io.Printf("Pass --addr XX:XX:XX:XX:XX:XX to create RFCOMM manually.\n")
		return nil, false
	}
	if len(devices) == 0 {
		io.Printf("No Nothing/CMF devices found. Pass --addr XX:XX:XX:XX:XX:XX to create RFCOMM manually.\n")
		return nil, false
	}
	return devices, true
}

func printDiscoveredDeviceList(io PreflightIO, devices []bt.Device) {
	io.Printf("Discovered devices:\n")
	for i, device := range devices {
		io.Printf("  %d) %s  %s connected=%t paired=%t spp=%t\n", i+1, device.MAC, device.Name, device.Connected, device.Paired, device.SPP)
	}
	io.Printf("Select device number to bind, or press Enter to skip: ")
}

func chooseDiscoveredDevice(io PreflightIO, devices []bt.Device, channel int) (bt.Device, bool, error) {
	if io.ReadLine == nil {
		return bt.Device{}, false, fmt.Errorf("read line is required")
	}
	line, err := io.ReadLine()
	if err != nil {
		return bt.Device{}, false, err
	}
	index, err := ParseDeviceSelection(line, len(devices))
	if err != nil {
		return bt.Device{}, false, err
	}
	if index == 0 {
		return bt.Device{}, false, nil
	}
	device, err := DeviceAtSelection(index, devices, channel)
	if err != nil {
		return bt.Device{}, false, err
	}
	return device, true, nil
}

// PreflightRFCOMM ensures an RFCOMM node exists, optionally creating it interactively.
func PreflightRFCOMM(ctx context.Context, mgr *Manager, address string, io PreflightIO) (bt.Device, bool, error) {
	devicePath := mgr.Options().RFCOMMPath
	channel := mgr.Options().Channel
	address = ResolvePreflightAddress(address, devicePath)

	exists, err := preflightRFCOMMExists(mgr)
	if err != nil {
		return bt.Device{}, false, err
	}
	if exists {
		return mgr.DeviceForExistingRFCOMM(address), true, nil
	}
	return createPreflightRFCOMM(ctx, mgr, address, channel, devicePath, io)
}

func createPreflightRFCOMM(ctx context.Context, mgr *Manager, address string, channel int, devicePath string, io PreflightIO) (bt.Device, bool, error) {
	if io.Printf == nil {
		io.Printf = func(string, ...any) {}
	}
	io.Printf("%s does not exist.\n", devicePath)
	device, ok, err := SelectDeviceForRFCOMM(address, channel, io)
	if err != nil || !ok {
		return bt.Device{}, false, err
	}

	if io.Confirm == nil {
		return bt.Device{}, false, fmt.Errorf("confirm is required")
	}
	if !io.Confirm(fmt.Sprintf("Create %s for %s on channel %d?", devicePath, device.MAC, channel)) {
		io.Printf("Skipping RFCOMM creation; TUI discovery will still start.\n")
		return bt.Device{}, false, nil
	}

	if err := preflightBind(mgr, ctx, device); err != nil {
		return bt.Device{}, false, err
	}
	return device, true, nil
}

func runPreflightConnect(ctx context.Context, device bt.Device, connect func(context.Context, bt.Device) error) {
	go func() {
		select {
		case <-ctx.Done():
			return
		default:
		}
		if err := connect(ctx, device); err != nil && ctx.Err() == nil {
			fmt.Fprintf(os.Stderr, "connect %s: %v\n", device.MAC, err)
		}
	}()
}

// RunPreflightConnect performs preflight and optionally connects in the background.
func RunPreflightConnect(ctx context.Context, mgr *Manager, autoDiscover bool, address string, io PreflightIO, connect func(context.Context, bt.Device) error) error {
	if autoDiscover && address == "" {
		return nil
	}
	device, ok, err := PreflightRFCOMM(ctx, mgr, address, io)
	if err != nil {
		return fmt.Errorf("rfcomm preflight: %w", err)
	}
	if !ok || connect == nil {
		return nil
	}
	runPreflightConnect(ctx, device, connect)
	return nil
}

func reportAutoConnectErr(report func(string), err error) {
	switch {
	case errors.Is(err, errWaitingForBluetooth):
		report("auto: waiting for headphones (Bluetooth disconnected)")
	case errors.Is(err, errWaitingForAudioOutput):
		report("auto: waiting for bluetooth audio profile (A2DP)")
	case errors.Is(err, errNoCandidate):
		report("auto: no compatible TWS device found")
	}
}
