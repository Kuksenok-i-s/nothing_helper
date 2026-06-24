//go:build darwin

package bt

/*
#cgo darwin LDFLAGS: -framework IOBluetooth -framework Foundation
#include <stdlib.h>
#include "darwin_bridge.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"unsafe"

	"tws_manager/internal/security"
)

var darwinInitOnce sync.Once

func ensureDarwinBT() {
	darwinInitOnce.Do(func() {
		if C.bt_init() != 0 {
			panic("bt_init failed")
		}
	})
}

type darwinTransport struct {
	handle  int
	mac     string
	channel int
	label   string
	closed  atomic.Bool
}

func (t *darwinTransport) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	for {
		if t.closed.Load() {
			return 0, io.EOF
		}
		n := int(C.bt_transport_read(C.int(t.handle), (*C.uint8_t)(unsafe.Pointer(&p[0])), C.int(len(p)), 1000))
		if n < 0 {
			return 0, fmt.Errorf("darwin rfcomm read: %d", n)
		}
		if n > 0 {
			return n, nil
		}
	}
}

func (t *darwinTransport) Write(p []byte) (int, error) {
	if t.closed.Load() {
		return 0, fmt.Errorf("darwin rfcomm closed")
	}
	if len(p) == 0 {
		return 0, nil
	}
	n := int(C.bt_transport_write(C.int(t.handle), (*C.uint8_t)(unsafe.Pointer(&p[0])), C.int(len(p))))
	if n < 0 {
		return 0, fmt.Errorf("darwin rfcomm write: %d", n)
	}
	return n, nil
}

func (t *darwinTransport) Close() error {
	if t.closed.Swap(true) {
		return nil
	}
	if C.bt_transport_close(C.int(t.handle)) != 0 {
		return errors.New("darwin rfcomm close failed")
	}
	return nil
}

func (t *darwinTransport) RemoteMAC() string { return t.mac }
func (t *darwinTransport) Channel() int      { return t.channel }
func (t *darwinTransport) String() string    { return t.label }

// resolveRFCOMMChannel queries SDP for the Nothing SPP UUID (then serial-port 0x1101).
// Falls back to hint / cached / default when SDP is unavailable.
func resolveRFCOMMChannel(mac string, hint int) (int, error) {
	normMAC, err := security.NormalizeMAC(mac)
	if err != nil {
		return 0, err
	}
	if hint <= 0 {
		hint = ResolveDeviceChannel(normMAC, DefaultRFCOMMChannel)
	}
	ensureDarwinBT()
	cmac := C.CString(normMAC)
	defer C.free(unsafe.Pointer(cmac))
	var out C.int
	rc := C.bt_resolve_rfcomm_channel(cmac, C.int(hint), &out)
	if rc == 0 {
		return int(out), nil
	}
	if rc == -2 {
		return 0, fmt.Errorf("device not found: %s", normMAC)
	}
	return ResolveDeviceChannel(normMAC, hint), nil
}

func darwinDeviceChannel(mac string) int {
	return ResolveDeviceChannel(mac, DefaultRFCOMMChannel)
}

func openDarwinRFCOMM(mac string, channel int, progress RFCOMMProgress) (Transport, int, error) {
	if err := security.ValidateChannel(channel); err != nil {
		return nil, 0, err
	}
	cmac := C.CString(mac)
	defer C.free(unsafe.Pointer(cmac))
	var handle C.bt_transport_handle
	rc := C.bt_open_rfcomm(cmac, C.int(channel), &handle)
	if rc != 0 {
		if rc == -5 {
			return nil, 0, fmt.Errorf("open rfcomm channel %d: code %d (put earbuds out of case; disconnect phone Nothing app if open)", channel, rc)
		}
		return nil, 0, fmt.Errorf("open rfcomm channel %d: code %d", channel, rc)
	}
	label := TransportLabel(mac, channel)
	transport := &darwinTransport{
		handle:  int(handle.handle),
		mac:     mac,
		channel: channel,
		label:   label,
	}
	_ = RememberDeviceChannel(mac, channel)
	report(progress, fmt.Sprintf("opened %s on channel %d", transport.String(), channel))
	return transport, channel, nil
}

func Discover() ([]Device, error) {
	ensureDarwinBT()
	var records [C.BT_MAX_DEVICES]C.bt_device_record
	var count C.int
	if C.bt_discover_paired(&records[0], C.BT_MAX_DEVICES, &count) != 0 {
		return nil, errors.New("darwin discover failed")
	}
	out := make([]Device, 0, int(count))
	for i := 0; i < int(count); i++ {
		rec := records[i]
		mac := C.GoString(&rec.mac[0])
		dev := Device{
			MAC:       mac,
			Name:      C.GoString(&rec.name[0]),
			Alias:     C.GoString(&rec.alias[0]),
			Info:      C.GoString(&rec.info[0]),
			Connected: rec.connected != 0,
			Paired:    rec.paired != 0,
			SPP:       rec.spp != 0,
			Channel:   darwinDeviceChannel(mac),
		}
		if isCandidate(dev) || dev.SPP {
			out = append(out, dev)
		}
	}
	return out, nil
}

func BluetoothInfo(mac string) (string, error) {
	if err := security.ValidateMAC(mac); err != nil {
		return "", err
	}
	ensureDarwinBT()
	cmac := C.CString(mac)
	defer C.free(unsafe.Pointer(cmac))
	info := C.bt_device_info(cmac)
	if info == nil {
		return "", nil
	}
	defer C.free(unsafe.Pointer(info))
	return C.GoString(info), nil
}

func IsDeviceConnected(mac string) (bool, error) {
	if err := security.ValidateMAC(mac); err != nil {
		return false, err
	}
	ensureDarwinBT()
	cmac := C.CString(mac)
	defer C.free(unsafe.Pointer(cmac))
	return C.bt_is_connected(cmac) != 0, nil
}

func EnrichDeviceInfo(dev Device) Device {
	if dev.MAC == "" {
		return dev
	}
	info, err := BluetoothInfo(dev.MAC)
	if err != nil {
		return dev
	}
	dev.Info = info
	applyBluetoothInfo(&dev, info)
	cmac := C.CString(dev.MAC)
	defer C.free(unsafe.Pointer(cmac))
	if C.bt_device_has_spp(cmac) != 0 {
		dev.SPP = true
	}
	if dev.Channel <= 0 {
		dev.Channel = ResolveDeviceChannel(dev.MAC, DefaultRFCOMMChannel)
	}
	return dev
}

func WarmupSudo() (bool, error) { return false, nil }

func ConfigureSudoPasswordProvider(fn func(prompt string) (string, error)) {}

func SudoAvailable() bool { return false }

func BindRFCOMMDevice(device, address string, channel int) error {
	return nil
}

func ReleaseRFCOMMDevice(device string) error { return nil }

func ReviveRFCOMMDevice(device, address string, channel int, progress RFCOMMProgress) error {
	return errors.New("revive not supported on darwin")
}

func OpenTransport(transportRef, address string, channel int, progress RFCOMMProgress) (Transport, int, error) {
	ensureDarwinBT()
	mac := address
	if mac == "" {
		if m, ok := LookupDeviceMAC(transportRef); ok {
			mac = m
		} else if normalized, err := security.ValidateTransportRef(transportRef); err == nil {
			if m, err2 := security.NormalizeMAC(normalized); err2 == nil {
				mac = m
			}
		}
	}
	if mac == "" {
		return nil, 0, fmt.Errorf("device MAC is required on darwin")
	}
	normMAC, err := security.NormalizeMAC(mac)
	if err != nil {
		return nil, 0, err
	}
	hint := ResolveDeviceChannel(normMAC, channel)
	resolved := hint
	if ch, resolveErr := resolveRFCOMMChannel(normMAC, hint); resolveErr != nil {
		report(progress, fmt.Sprintf("SDP unavailable, using channel %d", resolved))
	} else {
		resolved = ch
		_ = RememberDeviceChannel(normMAC, ch)
		report(progress, fmt.Sprintf("SDP RFCOMM channel %d", resolved))
	}
	return openDarwinRFCOMM(normMAC, resolved, progress)
}

// OpenTransportExact opens a single RFCOMM channel without SDP re-resolve.
func OpenTransportExact(mac string, channel int, progress RFCOMMProgress) (Transport, int, error) {
	ensureDarwinBT()
	normMAC, err := security.NormalizeMAC(mac)
	if err != nil {
		return nil, 0, err
	}
	channel = ResolveDeviceChannel(normMAC, channel)
	return openDarwinRFCOMM(normMAC, channel, progress)
}

func isRecoverableRFCOMMOpenError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "code -3") || strings.Contains(msg, "code -4") || strings.Contains(msg, "code -5") || strings.Contains(msg, "timed out")
}

func HostAdapterMAC() (string, error) {
	ensureDarwinBT()
	mac := C.bt_host_adapter_mac()
	if mac == nil {
		return "", errors.New("host adapter MAC unavailable")
	}
	defer C.free(unsafe.Pointer(mac))
	s := C.GoString(mac)
	if s == "" {
		return "", errors.New("host adapter MAC empty")
	}
	return security.NormalizeMAC(s)
}
