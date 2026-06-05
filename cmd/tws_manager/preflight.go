package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"tws_manager/internal/bt"
	"tws_manager/internal/connect"
)

func preflightRFCOMM(mgr *connect.Manager, address string) (bt.Device, bool, error) {
	devicePath := mgr.Options().RFCOMMPath
	channel := mgr.Options().Channel

	if address == "" {
		if mac, ok := bt.LookupDeviceMAC(devicePath); ok {
			address = mac
		}
	}

	exists, err := mgr.RFCOMMExists()
	if err != nil {
		return bt.Device{}, false, err
	}
	if exists {
		return mgr.DeviceForExistingRFCOMM(address), true, nil
	}

	fmt.Printf("%s does not exist.\n", devicePath)
	device, ok, err := selectDeviceForRFCOMM(address, channel)
	if err != nil || !ok {
		return bt.Device{}, false, err
	}

	if !confirm(fmt.Sprintf("Create %s for %s on channel %d?", devicePath, device.MAC, channel)) {
		fmt.Println("Skipping RFCOMM creation; TUI discovery will still start.")
		return bt.Device{}, false, nil
	}

	if err := mgr.Bind(context.Background(), device); err != nil {
		return bt.Device{}, false, err
	}

	return device, true, nil
}

func selectDeviceForRFCOMM(address string, channel int) (bt.Device, bool, error) {
	if address != "" {
		dev, err := connect.DeviceFromAddress(address, channel)
		if err != nil {
			return bt.Device{}, false, err
		}
		return dev, true, nil
	}

	devices, err := bt.Discover()
	if err != nil {
		fmt.Printf("Could not discover devices: %v\n", err)
		fmt.Println("Pass --addr XX:XX:XX:XX:XX:XX to create RFCOMM manually.")
		return bt.Device{}, false, nil
	}
	if len(devices) == 0 {
		fmt.Println("No Nothing/CMF devices found. Pass --addr XX:XX:XX:XX:XX:XX to create RFCOMM manually.")
		return bt.Device{}, false, nil
	}

	fmt.Println("Discovered devices:")
	for i, device := range devices {
		fmt.Printf("  %d) %s  %s connected=%t paired=%t spp=%t\n", i+1, device.MAC, device.Name, device.Connected, device.Paired, device.SPP)
	}
	fmt.Print("Select device number to bind, or press Enter to skip: ")

	line, err := readLine()
	if err != nil {
		return bt.Device{}, false, err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return bt.Device{}, false, nil
	}
	index, err := strconv.Atoi(line)
	if err != nil || index < 1 || index > len(devices) {
		return bt.Device{}, false, fmt.Errorf("invalid device selection %q", line)
	}

	device := devices[index-1]
	if device.Channel == 0 {
		device.Channel = channel
	}
	return device, true, nil
}

func confirm(prompt string) bool {
	fmt.Printf("%s [y/N] ", prompt)
	line, err := readLine()
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

func readLine() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	return reader.ReadString('\n')
}
