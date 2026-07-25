//go:build linux

package bt

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/unix"

	"tws_manager/internal/security"
)

const (
	commandTimeout = 30 * time.Second
	sudoTimeout    = 60 * time.Second
)

var execCombinedOutput = func(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.CombinedOutput()
}

var (
	execCmdRun            = func(cmd *exec.Cmd) error { return cmd.Run() }
	execCmdCombinedOutput = func(cmd *exec.Cmd) ([]byte, error) { return cmd.CombinedOutput() }
	execSudoHook          func(args ...string) error
	bluetoothInfoFn       = BluetoothInfo
)

var (
	sudoMu          sync.Mutex
	sudoWarmupDone  bool
	sudoTicketValid bool
	sudoPasswordFn  func(prompt string) (string, error)
)

func Discover() ([]Device, error) {
	var warns []error
	connected, err := bluetoothDevices("Connected")
	if err != nil {
		warns = append(warns, err)
	}
	paired, err := bluetoothDevices("Paired")
	if err != nil {
		warns = append(warns, err)
	}
	devices := mergeBluetoothLists(paired, connected)
	out := make([]Device, 0, len(devices))
	for _, dev := range devices {
		info, infoErr := BluetoothInfo(dev.MAC)
		if infoErr != nil {
			warns = append(warns, infoErr)
		}
		dev = enrichDiscoveredDevice(dev, info)
		if isDiscoveredCandidate(dev) {
			out = append(out, dev)
		}
	}
	if len(warns) > 0 {
		return out, errors.Join(warns...)
	}
	return out, nil
}

func bluetoothDevices(kind string) ([]Device, error) {
	out, err := runCommand("bluetoothctl", "devices", kind)
	if err != nil {
		return nil, err
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
	out, err := runCommand("bluetoothctl", "info", mac)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// IsDeviceConnected reports whether bluetoothctl lists the device as Connected: yes.
// A missing BlueZ device ("not available") is treated as disconnected, not an error,
// so stale RFCOMM→MAC mappings can fall back to rediscovery.
func IsDeviceConnected(mac string) (bool, error) {
	info, err := BluetoothInfo(mac)
	if err != nil {
		if isBluetoothDeviceUnavailable(err) {
			return false, nil
		}
		return false, err
	}
	return deviceConnectedFromInfo(info), nil
}

func EnrichDeviceInfo(dev Device) Device {
	if dev.MAC == "" {
		return dev
	}
	info, err := bluetoothInfoFn(dev.MAC)
	if err != nil {
		return dev
	}
	dev.Info = info
	applyBluetoothInfo(&dev, info)
	return dev
}

// WarmupSudo prompts once for credentials (sudo -v) so later privileged ops can use sudo -n.
func WarmupSudo() (bool, error) {
	sudoMu.Lock()
	defer sudoMu.Unlock()
	if sudoWarmupDone {
		return sudoTicketValid, nil
	}
	sudoWarmupDone = true

	cmdCtx, cancel := context.WithTimeout(context.Background(), sudoTimeout)
	defer cancel()
	cmd := exec.CommandContext(cmdCtx, "sudo", "-v")
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

	_, err = runCommand("rfcomm", "bind", num, address, strconv.Itoa(channel))
	if err == nil && rfcommWaitForDevice(device, 1500*time.Millisecond) == nil {
		if accessErr := rfcommEnsureAccess(device); accessErr != nil {
			return wrapRFCOMMPermission(accessErr)
		}
		return nil
	}

	plainErr := wrapRFCOMMBind(err)
	if err == nil {
		plainErr = wrapRFCOMMBind(fmt.Errorf("rfcomm bind %s returned success but %s was not created", num, device))
	}
	return bindRFCOMMPrivileged(device, num, address, channel, plainErr)
}

// BindRFCOMMWithProbe binds an RFCOMM TTY, probing alternate channels when the
// preferred one fails. Returns the channel that worked.
func BindRFCOMMWithProbe(device, address string, channel int, progress RFCOMMProgress) (int, error) {
	device, address, channel, err := validateRFCCOMMBind(device, address, channel)
	if err != nil {
		return 0, err
	}
	channel = ResolveDeviceChannel(address, channel)
	usedChannel, err := probeRFCOMMChannels(channel, progress, "binding", func(ch, attempt int) error {
		if attempt > 0 {
			_ = ReleaseRFCOMMDevice(device)
		}
		if err := rfcommBindDevice(device, address, ch); err != nil {
			return err
		}
		f, err := rfcommOpenFile(device, 2*time.Second)
		if err != nil {
			return err
		}
		_ = f.Close()
		_ = RememberDeviceChannel(address, ch)
		return nil
	})
	if err != nil {
		return 0, err
	}
	return usedChannel, nil
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
	return releaseRFCOMMNumber(num)
}

func releaseRFCOMMNumber(num string) error {
	out, err := runCommand("rfcomm", "release", num)
	if err == nil || isRFCOMMNotBoundOutput(string(out)) || isRFCOMMNotBoundError(err) {
		return nil
	}
	if privErr := rfcommPrivilegedRelease(num); privErr != nil {
		if isRFCOMMNotBoundError(privErr) {
			return nil
		}
		return fmt.Errorf("%w; privileged fallback: %w", err, privErr)
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
		return wrapRFCOMMRevive(fmt.Errorf("bind %q: %w", device, err))
	}
	if err := rfcommWaitForDevice(device, 3*time.Second); err != nil {
		return wrapRFCOMMRevive(wrapRFCOMMWait(fmt.Errorf("wait for %q after bind: %w", device, err)))
	}

	report(progress, "granting RFCOMM device access")
	if err := rfcommEnsureAccess(device); err != nil {
		return err
	}

	report(progress, "verifying RFCOMM device")
	if _, err := rfcommOpenFile(device, 2*time.Second); err != nil {
		return wrapRFCOMMRevive(wrapRFCOMMOpen(fmt.Errorf("verify open %q: %w", device, err)))
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
		return "", "", 0, fmt.Errorf("%w: %w", ErrInvalidBluetoothMAC, err)
	}
	if err := security.ValidateChannel(channel); err != nil {
		return "", "", 0, err
	}
	return devPath, mac, channel, nil
}

func waitForDevice(device string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		_, err := os.Stat(device)
		if err == nil {
			return nil
		}
		lastErr = err
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
	openRFCOMM := ensureRFCOMMOpenHook
	if openRFCOMM == nil {
		openRFCOMM = os.OpenFile
	}
	if f, err := openRFCOMM(device, os.O_RDWR, 0); err == nil {
		return f.Close()
	}

	uid := strconv.Itoa(os.Getuid())
	gid := strconv.Itoa(os.Getgid())
	fmt.Fprintf(os.Stderr, "Granting current user access to %s via privileged helper\n", device)
	if err := privilegedEnsureRFCOMMAccess(device, uid+":"+gid); err != nil {
		return err
	}
	f, openErr := openRFCOMM(device, os.O_RDWR, 0)
	if openErr == nil {
		return f.Close()
	}
	return fmt.Errorf("open %q after chown/chmod: %w", device, openErr)
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

// execSudo runs `sudo args...` in one of three modes:
//   - cached ticket (after WarmupSudo): non-interactive `sudo -n`;
//   - GUI password provider: `sudo -S` with the password piped to stdin;
//   - interactive fallback: sudo inherits the terminal stdio.
//
// Errors are formatted uniformly via commandError with the original args
// (sudo flags excluded). A successful run marks sudo as available.
func execSudo(args ...string) error {
	if execSudoHook != nil {
		return execSudoHook(args...)
	}
	label := "sudo " + strings.Join(args, " ")
	sudoCtx, cancel := context.WithTimeout(context.Background(), sudoTimeout)
	defer cancel()
	cmd, interactive, err := buildSudoCommand(sudoCtx, label, args)
	if err != nil {
		return err
	}
	if err := runSudoCommand(label, cmd, interactive, args); err != nil {
		return err
	}
	markSudoAvailable()
	return nil
}

func buildSudoCommand(ctx context.Context, label string, args []string) (*exec.Cmd, bool, error) {
	var cmd *exec.Cmd
	interactive := false
	switch {
	case SudoAvailable():
		cmd = exec.CommandContext(ctx, "sudo", append([]string{"-n"}, args...)...)
	case sudoPasswordProvider() != nil:
		password, err := sudoPasswordProvider()("Administrator password is required for " + label)
		if err != nil {
			return nil, false, fmt.Errorf("%s: %w", label, err)
		}
		cmd = exec.CommandContext(ctx, "sudo", append([]string{"-S", "-p", ""}, args...)...)
		cmd.Stdin = bytes.NewBufferString(password + "\n")
	default:
		interactive = true
		cmd = exec.CommandContext(ctx, "sudo", args...)
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	}
	return cmd, interactive, nil
}

func runSudoCommand(label string, cmd *exec.Cmd, interactive bool, args []string) error {
	if interactive {
		if err := execCmdRun(cmd); err != nil {
			return fmt.Errorf("%s: %w", label, err)
		}
		return nil
	}
	if out, err := execCmdCombinedOutput(cmd); err != nil {
		return commandError("sudo", args, out, err)
	}
	return nil
}

// runCommand executes an external command and returns its combined output.
// On failure the error carries the full command line and trimmed output.
func runCommand(name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	out, err := execCombinedOutput(ctx, name, args...)
	if err != nil {
		return out, commandError(name, args, out, err)
	}
	return out, nil
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

func isRecoverableOpenMessage(msg string) bool {
	return strings.Contains(msg, "timed out") ||
		strings.Contains(msg, "input/output error") ||
		strings.Contains(msg, "no such device") ||
		strings.Contains(msg, "no such file or directory") ||
		strings.Contains(msg, "device not configured")
}

func isRecoverableSyscallErr(err error) bool {
	return errors.Is(err, syscall.EIO) ||
		errors.Is(err, syscall.ENXIO) ||
		errors.Is(err, syscall.ENODEV) ||
		errors.Is(err, syscall.ENOTCONN)
}

func isRecoverableRFCOMMOpenError(err error) bool {
	if err == nil {
		return false
	}
	if isRecoverableOpenMessage(strings.ToLower(err.Error())) {
		return true
	}
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		err = pathErr.Err
	}
	return isRecoverableSyscallErr(err)
}

// OpenTransport opens an RFCOMM tty, probing alternate channels when the
// preferred one fails. Returns the channel that worked.
func OpenTransport(device, address string, channel int, progress RFCOMMProgress) (Transport, int, error) {
	channel = ResolveDeviceChannel(address, channel)
	if address == "" {
		return openTransportWithoutAddress(device, address, channel, progress)
	}
	return openTransportWithAddress(device, address, channel, progress)
}

func openTransportWithoutAddress(device, address string, channel int, progress RFCOMMProgress) (Transport, int, error) {
	t, err := openRFCOMMOnChannel(device, address, channel, progress)
	if err != nil {
		return nil, channel, err
	}
	return newRWCTransport(t, "", channel, device), channel, nil
}

func openTransportWithAddress(device, address string, channel int, progress RFCOMMProgress) (Transport, int, error) {
	var opened *os.File
	usedChannel, err := probeRFCOMMChannels(channel, progress, "trying", func(ch, attempt int) error {
		if attempt > 0 {
			_ = ReleaseRFCOMMDevice(device)
		}
		f, openErr := openRFCOMMOnChannel(device, address, ch, progress)
		if openErr != nil {
			return openErr
		}
		opened = f
		_ = RememberDeviceChannel(address, ch)
		return nil
	})
	if err != nil {
		return nil, 0, err
	}
	mac := address
	if mac == "" {
		if m, ok := LookupDeviceMAC(device); ok {
			mac = m
		}
	}
	return newRWCTransport(opened, mac, usedChannel, device), usedChannel, nil
}

func openRFCOMMOnChannel(device, address string, channel int, progress RFCOMMProgress) (*os.File, error) {
	device, address, channel, err := validateOpenRFCOMMParams(device, address, channel)
	if err != nil {
		return nil, err
	}

	f, err := rfcommOpenFile(device, 5*time.Second)
	if err == nil {
		return f, nil
	}
	if os.IsPermission(err) {
		return openRFCOMMAfterPermissionFix(device, progress)
	}
	if isRecoverableRFCOMMOpenError(err) && address != "" {
		return openRFCOMMAfterRevive(device, address, channel, progress, err)
	}
	return openRFCOMMOnMissingDevice(device, address, channel, progress, err)
}

func openRFCOMMOnMissingDevice(device, address string, channel int, progress RFCOMMProgress, err error) (*os.File, error) {
	if !os.IsNotExist(err) {
		return nil, wrapRFCOMMOpen(fmt.Errorf("open %q: %w", device, err))
	}
	if address == "" {
		return nil, wrapRFCOMMOpen(fmt.Errorf("device %q does not exist; pass --addr or choose a device", device))
	}
	return createBindAndOpenRFCOMM(device, address, channel, progress)
}

// openFileWithTimeout opens an RFCOMM tty without ever blocking the caller.
//
// A plain blocking open() on /dev/rfcommN parks inside the kernel
// (tty_port_block_til_ready) until the RFCOMM link comes up, which previously
// required a watchdog goroutine that could leak forever. Instead we open with
// O_NONBLOCK (returns immediately) and poll the carrier bit (TIOCM_CD) — the
// same condition the blocking open waits for — until the deadline.
func openFileWithTimeout(device string, timeout time.Duration) (*os.File, error) {
	fd, err := unix.Open(device, unix.O_RDWR|unix.O_NOCTTY|unix.O_NONBLOCK, 0)
	if err != nil {
		return nil, &os.PathError{Op: "open", Path: device, Err: err}
	}

	if err := waitRFCOMMCarrier(fd, device, timeout); err != nil {
		_ = unix.Close(fd)
		return nil, err
	}

	if err := unix.SetNonblock(fd, false); err != nil {
		_ = unix.Close(fd)
		return nil, &os.PathError{Op: "open", Path: device, Err: err}
	}
	f := os.NewFile(uintptr(fd), device)
	setRawMode(f)
	return f, nil
}

func waitRFCOMMCarrier(fd int, device string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if bits, err := unix.IoctlGetInt(fd, unix.TIOCMGET); err != nil || bits&unix.TIOCM_CD != 0 {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return wrapRFCOMMOpen(fmt.Errorf("open %q timed out after %s", device, timeout))
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
