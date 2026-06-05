package bt

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/unix"

	"tws_manager/internal/security"
)

const (
	NothingSPPUUID  = "AEAC4A03-DFF5-498F-843A-34487CF133EB"
	privateDirPerm  = 0o700
	privateFilePerm = 0o600
)

type Device struct {
	MAC       string `json:"mac"`
	Name      string `json:"name"`
	Alias     string `json:"alias,omitempty"`
	Info      string `json:"info,omitempty"`
	Connected bool   `json:"connected"`
	Paired    bool   `json:"paired"`
	SPP       bool   `json:"spp"`
	Channel   int    `json:"channel"`
}

type Config struct {
	Devices  map[string]string `json:"devices"`
	Channels map[string]int    `json:"channels,omitempty"`
}

// RFCOMMProgress reports recovery steps (release, bind, reopen).
type RFCOMMProgress func(step string)

var (
	sudoMu          sync.Mutex
	sudoWarmupDone  bool
	sudoTicketValid bool
	sudoPasswordFn  func(prompt string) (string, error)
)

func Discover() ([]Device, error) {
	devices := map[string]Device{}
	connected, _ := bluetoothDevices("Connected")
	paired, _ := bluetoothDevices("Paired")
	for _, dev := range paired {
		dev.Paired = true
		devices[dev.MAC] = dev
	}
	for _, dev := range connected {
		cur := devices[dev.MAC]
		if cur.MAC == "" {
			cur = dev
		}
		cur.Connected = true
		if cur.Name == "" {
			cur.Name = dev.Name
		}
		devices[dev.MAC] = cur
	}
	out := make([]Device, 0, len(devices))
	for _, dev := range devices {
		info, _ := BluetoothInfo(dev.MAC)
		dev.Info = info
		applyBluetoothInfo(&dev, info)
		dev.SPP = strings.Contains(strings.ToUpper(info), NothingSPPUUID)
		dev.Channel = ResolveDeviceChannel(dev.MAC, DefaultRFCOMMChannel)
		if isCandidate(dev) || dev.SPP {
			out = append(out, dev)
		}
	}
	return out, nil
}

func bluetoothDevices(kind string) ([]Device, error) {
	out, err := exec.Command("bluetoothctl", "devices", kind).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("bluetoothctl devices %s: %w: %s", kind, err, strings.TrimSpace(string(out)))
	}
	var devices []Device
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[0] != "Device" {
			continue
		}
		devices = append(devices, Device{MAC: fields[1], Name: strings.Join(fields[2:], " ")})
	}
	return devices, nil
}

func BluetoothInfo(mac string) (string, error) {
	if err := security.ValidateMAC(mac); err != nil {
		return "", err
	}
	out, err := exec.Command("bluetoothctl", "info", mac).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("bluetoothctl info %s: %w: %s", mac, err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
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
	return dev
}

func applyBluetoothInfo(dev *Device, info string) {
	for _, line := range strings.Split(info, "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), ":")
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "name":
			if dev.Name == "" || strings.EqualFold(dev.Name, dev.MAC) {
				dev.Name = value
			}
		case "alias":
			dev.Alias = value
		}
	}
}

func isCandidate(dev Device) bool {
	name := strings.ToLower(dev.Name)
	alias := strings.ToLower(dev.Alias)
	for _, token := range []string{"nothing", "cmf", "ear", "headphone", "neckband"} {
		if strings.Contains(name, token) || strings.Contains(alias, token) {
			return true
		}
	}
	return false
}

// WarmupSudo prompts once for credentials (sudo -v) so later privileged ops can use sudo -n.
func WarmupSudo() (bool, error) {
	sudoMu.Lock()
	defer sudoMu.Unlock()
	if sudoWarmupDone {
		return sudoTicketValid, nil
	}
	sudoWarmupDone = true

	cmd := exec.Command("sudo", "-v")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		sudoTicketValid = false
		return false, fmt.Errorf("sudo authentication failed: %w", err)
	}
	sudoTicketValid = true
	return true, nil
}

// ConfigureSudoPasswordProvider installs an optional GUI password provider for
// sudo fallback. Passing nil restores terminal/stdin sudo behaviour.
func ConfigureSudoPasswordProvider(fn func(prompt string) (string, error)) {
	sudoMu.Lock()
	defer sudoMu.Unlock()
	sudoPasswordFn = fn
}

// SudoAvailable reports whether a non-interactive sudo ticket is active after WarmupSudo.
func SudoAvailable() bool {
	sudoMu.Lock()
	defer sudoMu.Unlock()
	return sudoTicketValid
}

func sudoPasswordProvider() func(prompt string) (string, error) {
	sudoMu.Lock()
	defer sudoMu.Unlock()
	return sudoPasswordFn
}

func markSudoAvailable() {
	sudoMu.Lock()
	defer sudoMu.Unlock()
	sudoWarmupDone = true
	sudoTicketValid = true
}

func BindRFCOMMDevice(device, address string, channel int) error {
	device, address, channel, err := validateRFCCOMMBind(device, address, channel)
	if err != nil {
		return err
	}
	num, err := security.RFCOMMNumber(device)
	if err != nil {
		return err
	}

	out, err := exec.Command("rfcomm", "bind", num, address, strconv.Itoa(channel)).CombinedOutput()
	if err == nil && waitForDevice(device, 1500*time.Millisecond) == nil {
		return EnsureRFCOMMDeviceAccess(device)
	}

	args := []string{"bind", num, address, strconv.Itoa(channel)}
	plainErr := commandError("rfcomm", args, out, err)
	if err == nil {
		plainErr = fmt.Errorf("rfcomm %s returned success but %s was not created", strings.Join(args, " "), device)
	}
	if privErr := privilegedRFCCOMMBind(num, address, channel); privErr != nil {
		return fmt.Errorf("%w; privileged fallback failed: %w", plainErr, privErr)
	}
	if err := waitForDevice(device, 3*time.Second); err != nil {
		return fmt.Errorf("sudo rfcomm bind %s succeeded but %s was not created: %w", num, device, err)
	}

	return EnsureRFCOMMDeviceAccess(device)
}

// ReleaseRFCOMMDevice unbinds an RFCOMM TTY. "Not bound" is treated as success.
func ReleaseRFCOMMDevice(device string) error {
	device, err := security.ValidateRFCOMMDevice(device)
	if err != nil {
		return err
	}
	num, err := security.RFCOMMNumber(device)
	if err != nil {
		return err
	}

	out, err := exec.Command("rfcomm", "release", num).CombinedOutput()
	if err == nil || isRFCOMMNotBoundOutput(string(out)) || isRFCOMMNotBoundError(err) {
		return nil
	}
	if privErr := privilegedRFCOMMRelease(num); privErr != nil {
		if isRFCOMMNotBoundOutput(string(out)) || isRFCOMMNotBoundError(err) || isRFCOMMNotBoundError(privErr) {
			return nil
		}
		return fmt.Errorf("rfcomm release %s: %w; privileged fallback: %w", num, commandError("rfcomm", []string{"release", num}, out, err), privErr)
	}
	return nil
}

// ReviveRFCOMMDevice releases and re-binds a stale RFCOMM device node.
func ReviveRFCOMMDevice(device, address string, channel int, progress RFCOMMProgress) error {
	device, address, channel, err := validateRFCCOMMBind(device, address, channel)
	if err != nil {
		return err
	}
	report(progress, "releasing RFCOMM device")
	if err := ReleaseRFCOMMDevice(device); err != nil {
		return fmt.Errorf("release %q: %w", device, err)
	}
	time.Sleep(200 * time.Millisecond)

	report(progress, "binding RFCOMM device")
	if err := BindRFCOMMDevice(device, address, channel); err != nil {
		return fmt.Errorf("bind %q: %w", device, err)
	}
	if err := waitForDevice(device, 3*time.Second); err != nil {
		return fmt.Errorf("wait for %q after bind: %w", device, err)
	}

	report(progress, "granting RFCOMM device access")
	if err := EnsureRFCOMMDeviceAccess(device); err != nil {
		return err
	}

	report(progress, "verifying RFCOMM device")
	if _, err := openFileWithTimeout(device, 2*time.Second); err != nil {
		return fmt.Errorf("verify open %q: %w", device, err)
	}
	return nil
}

func validateRFCCOMMBind(device, address string, channel int) (string, string, int, error) {
	devPath, err := security.ValidateRFCOMMDevice(device)
	if err != nil {
		return "", "", 0, err
	}
	mac, err := security.NormalizeMAC(address)
	if err != nil {
		return "", "", 0, err
	}
	if err := security.ValidateChannel(channel); err != nil {
		return "", "", 0, err
	}
	return devPath, mac, channel, nil
}

func report(progress RFCOMMProgress, step string) {
	if progress != nil {
		progress(step)
	}
}

func waitForDevice(device string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		if _, err := os.Stat(device); err == nil {
			return nil
		} else {
			lastErr = err
		}
		time.Sleep(100 * time.Millisecond)
	}
	if lastErr == nil {
		lastErr = os.ErrNotExist
	}
	return lastErr
}

func EnsureRFCOMMDeviceAccess(device string) error {
	device, err := security.ValidateRFCOMMDevice(device)
	if err != nil {
		return err
	}
	if f, err := os.OpenFile(device, os.O_RDWR, 0); err == nil {
		return f.Close()
	}

	uid := strconv.Itoa(os.Getuid())
	gid := strconv.Itoa(os.Getgid())
	fmt.Fprintf(os.Stderr, "Granting current user access to %s via privileged helper\n", device)
	if err := privilegedEnsureRFCOMMAccess(device, uid+":"+gid); err != nil {
		return err
	}
	if f, err := os.OpenFile(device, os.O_RDWR, 0); err == nil {
		return f.Close()
	} else {
		return fmt.Errorf("open %q after chown/chmod: %w", device, err)
	}
}

func sudoRFCCOMMRelease(num string) error {
	if err := security.ValidateRFCOMMNumber(num); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Running: sudo rfcomm release %s\n", num)
	return execSudo("rfcomm", "release", num)
}

func sudoRFCCOMMBind(num, mac string, channel int) error {
	if err := security.ValidateRFCOMMNumber(num); err != nil {
		return err
	}
	if _, err := security.NormalizeMAC(mac); err != nil {
		return err
	}
	if err := security.ValidateChannel(channel); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Creating RFCOMM device requires privileges; running: sudo rfcomm bind %s %s %d\n", num, mac, channel)
	return execSudo("rfcomm", "bind", num, mac, strconv.Itoa(channel))
}

func sudoDeviceChown(device, owner string) error {
	device, err := security.ValidateRFCOMMDevice(device)
	if err != nil {
		return err
	}
	if owner == "" || strings.Contains(owner, " ") {
		return fmt.Errorf("invalid chown owner %q", owner)
	}
	return execSudo("chown", owner, device)
}

func sudoDeviceChmod(device string) error {
	device, err := security.ValidateRFCOMMDevice(device)
	if err != nil {
		return err
	}
	return execSudo("chmod", "u+rw", device)
}

func execSudo(args ...string) error {
	cmdArgs := args
	if SudoAvailable() {
		cmdArgs = append([]string{"-n"}, args...)
	}
	if provider := sudoPasswordProvider(); provider != nil && !SudoAvailable() {
		password, err := provider("Administrator password is required for sudo " + strings.Join(args, " "))
		if err != nil {
			return fmt.Errorf("sudo %s: %w", strings.Join(args, " "), err)
		}
		cmdArgs = append([]string{"-S", "-p", ""}, args...)
		cmd := exec.Command("sudo", cmdArgs...)
		cmd.Stdin = bytes.NewBufferString(password + "\n")
		out, err := cmd.CombinedOutput()
		if err != nil {
			output := strings.TrimSpace(string(out))
			if output != "" {
				return fmt.Errorf("sudo %s: %w: %s", strings.Join(args, " "), err, output)
			}
			return fmt.Errorf("sudo %s: %w", strings.Join(args, " "), err)
		}
		markSudoAvailable()
		return nil
	}
	cmd := exec.Command("sudo", cmdArgs...)
	if !SudoAvailable() {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("sudo %s: %w", strings.Join(args, " "), err)
		}
		markSudoAvailable()
		return nil
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		output := strings.TrimSpace(string(out))
		if output != "" {
			return fmt.Errorf("sudo %s: %w: %s", strings.Join(args, " "), err, output)
		}
		return fmt.Errorf("sudo %s: %w", strings.Join(args, " "), err)
	}
	return nil
}

func commandError(name string, args []string, out []byte, err error) error {
	output := strings.TrimSpace(string(out))
	if output != "" {
		return fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, output)
	}
	return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
}

func isRFCOMMNotBoundOutput(output string) bool {
	lower := strings.ToLower(output)
	return strings.Contains(lower, "not bound") ||
		strings.Contains(lower, "no such device") ||
		strings.Contains(lower, "can't release")
}

func isRFCOMMNotBoundError(err error) bool {
	if err == nil {
		return false
	}
	return isRFCOMMNotBoundOutput(err.Error())
}

func isRecoverableRFCOMMOpenError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "timed out") ||
		strings.Contains(msg, "input/output error") ||
		strings.Contains(msg, "no such device") ||
		strings.Contains(msg, "no such file or directory") ||
		strings.Contains(msg, "device not configured") {
		return true
	}
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		err = pathErr.Err
	}
	return errors.Is(err, syscall.EIO) ||
		errors.Is(err, syscall.ENXIO) ||
		errors.Is(err, syscall.ENODEV) ||
		errors.Is(err, syscall.ENOTCONN)
}

// OpenRFCOMMDevice opens an RFCOMM tty, probing alternate channels when the
// preferred one fails. Returns the channel that worked.
func OpenRFCOMMDevice(device, address string, channel int, progress RFCOMMProgress) (*os.File, int, error) {
	channel = ResolveDeviceChannel(address, channel)
	if address == "" {
		f, err := openRFCOMMOnChannel(device, address, channel, progress)
		return f, channel, err
	}
	var lastErr error
	for i, ch := range channelCandidates(channel) {
		if i > 0 {
			report(progress, fmt.Sprintf("trying RFCOMM channel %d", ch))
			_ = ReleaseRFCOMMDevice(device)
		}
		f, err := openRFCOMMOnChannel(device, address, ch, progress)
		if err == nil {
			_ = RememberDeviceChannel(address, ch)
			return f, ch, nil
		}
		lastErr = err
		if !shouldProbeNextChannel(err) {
			return nil, 0, err
		}
	}
	return nil, 0, fmt.Errorf("no working RFCOMM channel for %s: %w", address, lastErr)
}

func openRFCOMMOnChannel(device, address string, channel int, progress RFCOMMProgress) (*os.File, error) {
	devPath, err := security.ValidateRFCOMMDevice(device)
	if err != nil {
		return nil, err
	}
	if address != "" {
		if address, err = security.NormalizeMAC(address); err != nil {
			return nil, err
		}
	}
	if err := security.ValidateChannel(channel); err != nil {
		return nil, err
	}
	device = devPath

	f, err := openFileWithTimeout(device, 5*time.Second)
	if err == nil {
		return f, nil
	}
	if os.IsPermission(err) {
		report(progress, "fixing RFCOMM permissions")
		if accessErr := EnsureRFCOMMDeviceAccess(device); accessErr != nil {
			return nil, accessErr
		}
		return openFileWithTimeout(device, 5*time.Second)
	}
	if isRecoverableRFCOMMOpenError(err) && address != "" {
		report(progress, "recovering stale RFCOMM device")
		if reviveErr := ReviveRFCOMMDevice(device, address, channel, progress); reviveErr != nil {
			return nil, fmt.Errorf("open %q: %w; revive failed: %w", device, err, reviveErr)
		}
		f, retryErr := openFileWithTimeout(device, 5*time.Second)
		if retryErr != nil {
			return nil, fmt.Errorf("open %q after revive: %w", device, retryErr)
		}
		return f, nil
	}
	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("open %q: %w", device, err)
	}
	if address == "" {
		return nil, fmt.Errorf("device %q does not exist; pass --addr or choose a device", device)
	}
	report(progress, "creating RFCOMM device")
	if err := BindRFCOMMDevice(device, address, channel); err != nil {
		return nil, fmt.Errorf("create %q: %w", device, err)
	}
	if err := waitForDevice(device, 2*time.Second); err != nil {
		return nil, fmt.Errorf("wait for %q after rfcomm bind: %w", device, err)
	}
	f, err = openFileWithTimeout(device, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("open %q after rfcomm bind: %w", device, err)
	}
	return f, nil
}

func openFileWithTimeout(device string, timeout time.Duration) (*os.File, error) {
	type result struct {
		file *os.File
		err  error
	}
	ch := make(chan result, 1)
	timedOut := make(chan struct{})
	go func() {
		f, err := os.OpenFile(device, os.O_RDWR|syscall.O_NOCTTY, 0)
		select {
		case <-timedOut:
			if f != nil {
				_ = f.Close()
			}
			return
		default:
		}
		if err == nil && f != nil {
			setRawMode(f)
		}
		ch <- result{file: f, err: err}
	}()
	select {
	case res := <-ch:
		return res.file, res.err
	case <-time.After(timeout):
		close(timedOut)
		return nil, fmt.Errorf("open %q timed out after %s", device, timeout)
	}
}

// setRawMode puts an RFCOMM tty into raw mode so binary SPP frames are passed
// through byte-for-byte and reads return immediately (VMIN=1, VTIME=0) instead
// of being line-buffered by the canonical line discipline. This is the
// in-process equivalent of `stty -F <dev> raw -echo -icanon min 1 time 0`, but
// applied to our own fd (no root needed). Best-effort: a no-op if the node is
// not a tty (e.g. a regular file in tests).
func setRawMode(f *os.File) {
	fd := int(f.Fd())
	t, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if err != nil {
		return
	}
	t.Iflag &^= unix.IGNBRK | unix.BRKINT | unix.PARMRK | unix.ISTRIP |
		unix.INLCR | unix.IGNCR | unix.ICRNL | unix.IXON
	t.Oflag &^= unix.OPOST
	t.Lflag &^= unix.ECHO | unix.ECHONL | unix.ICANON | unix.ISIG | unix.IEXTEN
	t.Cflag &^= unix.CSIZE | unix.PARENB
	t.Cflag |= unix.CS8
	t.Cc[unix.VMIN] = 1
	t.Cc[unix.VTIME] = 0
	_ = unix.IoctlSetTermios(fd, unix.TCSETS, t)
}

func ConfigPath() string {
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, "tws_manager", "devices.json")
	}
	return filepath.Join(".", "devices.json")
}

func LoadConfig(path string) (Config, error) {
	cfg := Config{Devices: map[string]string{}, Channels: map[string]int{}}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return sanitizeConfig(cfg), nil
}

func sanitizeConfig(cfg Config) Config {
	out := Config{Devices: map[string]string{}, Channels: map[string]int{}}
	if cfg.Devices != nil {
		for path, mac := range cfg.Devices {
			devPath, err := security.ValidateRFCOMMDevice(path)
			if err != nil {
				continue
			}
			normMAC, err := security.NormalizeMAC(mac)
			if err != nil {
				continue
			}
			out.Devices[devPath] = normMAC
		}
	}
	if cfg.Channels != nil {
		for mac, channel := range cfg.Channels {
			normMAC, err := security.NormalizeMAC(mac)
			if err != nil {
				continue
			}
			if err := security.ValidateChannel(channel); err != nil {
				continue
			}
			out.Channels[normMAC] = channel
		}
	}
	return out
}

func SaveConfig(path string, cfg Config) error {
	cfg = sanitizeConfig(cfg)
	if err := os.MkdirAll(filepath.Dir(path), privateDirPerm); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), privateFilePerm)
}

// LookupDeviceMAC returns a saved MAC for an RFCOMM device path.
func LookupDeviceMAC(devicePath string) (string, bool) {
	devPath, err := security.ValidateRFCOMMDevice(devicePath)
	if err != nil {
		return "", false
	}
	cfg, err := LoadConfig(ConfigPath())
	if err != nil {
		return "", false
	}
	mac, ok := cfg.Devices[devPath]
	return mac, ok && mac != ""
}

// RememberDeviceMAC persists devicePath -> MAC for later revive without --addr.
func RememberDeviceMAC(devicePath, mac string) error {
	devPath, err := security.ValidateRFCOMMDevice(devicePath)
	if err != nil {
		return err
	}
	normMAC, err := security.NormalizeMAC(mac)
	if err != nil {
		return err
	}
	path := ConfigPath()
	cfg, err := LoadConfig(path)
	if err != nil {
		return err
	}
	cfg.Devices[devPath] = normMAC
	return SaveConfig(path, cfg)
}
