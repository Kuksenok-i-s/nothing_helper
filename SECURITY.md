# Security

tws_manager is a **local** Bluetooth RFCOMM client. It runs on your machine, talks to paired earbuds over `/dev/rfcommN`, and uses either a `polkit` helper (recommended) or `sudo` fallback for privileged RFCOMM operations.

Project status: unofficial client. Nothing/CMF names and logos are trademarks of their respective owners.

## Threat model

| Asset | Risk |
|-------|------|
| Root via helper/pkexec | Misconfigured policy may grant too much; mitigated by narrow helper CLI and path validation |
| Root via `sudo` fallback | Broad session ticket after `sudo -v`; mitigated by allowlisted `sudo` subcommands only |
| RFCOMM device node | Must be `/dev/rfcomm<N>`; no arbitrary paths for bind/chown/chmod |
| Earbuds firmware | SET commands can change ANC/EQ/spatial; gated by `--unsafe` + TUI double-confirm |
| Trace/capture files | May contain MAC, model, summaries; raw bytes only with `--log-raw` |

This tool does **not** expose a network API. Attack surface is CLI flags, config file, and local Bluetooth stack.

## Privileged operations and modes

Privileged operations are:

- `rfcomm bind|release <N> …`
- `chown <uid:gid> /dev/rfcomm<N>`
- `chmod u+rw /dev/rfcomm<N>`

Supported runtime modes (`--privilege-helper`):

- `polkit`: execute `/usr/libexec/tws_manager_rfcomm_helper` via `pkexec`
- `sudo` (default in TUI): keeps legacy behavior with `sudo -v` warmup
- `auto` (default in Gio): try `polkit`, fallback to `sudo`; Gio asks for the sudo password in the window when needed
- `none`: disable privileged fallback completely

Arbitrary root commands are not exposed from application code.

### Polkit helper hardening

Helper command: `cmd/tws_manager_rfcomm_helper`.

Allowed actions only:

- `bind --number/--device --addr --channel --owner`
- `release --number/--device`
- `fix-perms --device --owner`

All action arguments are validated (`/dev/rfcommN`, MAC format, channel range, owner `uid:gid`).

Packaging artifacts:

- `packaging/common/org.tws_manager.rfcomm.policy`
- `packaging/common/90-tws_manager.rules`
- `packaging/common/tws_manager.sysusers`

### Optional legacy sudoers (NOPASSWD)

Replace `YOUR_USER` and adjust device index if needed:

```
YOUR_USER ALL=(root) NOPASSWD: /usr/bin/rfcomm bind 0 *, /usr/bin/rfcomm release 0
YOUR_USER ALL=(root) NOPASSWD: /usr/bin/chown * /dev/rfcomm0, /usr/bin/chmod u+rw /dev/rfcomm0
```

A broad `NOPASSWD: ALL` is discouraged. Even with NOPASSWD, the app only invokes the allowlisted commands above. Prefer polkit helper mode for desktop autostart.

## Input validation

- `--device` must be `/dev/rfcomm[0-9]+`
- `--addr` must be a Bluetooth MAC `AA:BB:CC:DD:EE:FF`
- `--channel` must be 1–30
- `--log` and `--capture-dir` must be absolute-safe paths (no `..` segments)

Saved mapping `~/.config/tws_manager/devices.json` is sanitized on load: invalid keys/MACs are dropped.

## File permissions

| Path | Mode |
|------|------|
| `~/.config/tws_manager/` | `0700` |
| `devices.json`, NDJSON logs, exports | `0600` |
| `captures/` parent dirs | `0700` |

## Device safety (earbuds firmware)

The client is designed to avoid sending dangerous or arbitrary packets to your headphones.

### Default behaviour

- After connect (unless `--no-probe`): one `GET identity`, then one `GET battery` with a **3s** gap (identity probe still waits **2s** after connect).
- TUI/Gio expose **GET** commands plus a small allowlist of validated UI writes
  (ANC, EQ, spatial, low latency, dual enable, dual peer connect/disconnect).
- Catalog commands marked unsafe (`restore_factory_setting`, debug test mode, etc.) are **always blocked** in `Send`, even with `--unsafe`.

### `--no-probe`

Skip automatic identity and battery probes on connect. Use when you want zero extra SPP traffic after opening RFCOMM.

### Validated UI SET commands

These commands are available from the UI without `--unsafe`; payload builders
validate their accepted values before anything is sent.

| Feature | Allowed values |
|---------|----------------|
| ANC | `off`, `strong`, `medium`, `weak`, `smart`, `transparency`, `comfortable`, or numeric `0-4`, `252`, `253`, `254`, `255`. Arbitrary bytes outside the allowlist are rejected. |
| EQ | Preset index `0..N` where `N` is `3` by default or `7` if the model supports `advance_eq`. Pass `--model` for correct limits. |
| Spatial | `on`/`off` only (unchanged). |
| Low latency | `on`/`off` only. |
| Dual | enable/disable and peer connect/disconnect actions built from the decoded device list. |

All other SET commands still require `--unsafe`; catalog entries marked unsafe
remain blocked even with `--unsafe`.

### Query scan (`--unsafe`)

Enter in the comment field (press `c` first), then select **Advanced: raw scan** and confirm twice:

```text
scan c001 c020 500ms
```

Rules:

- Only **GET** opcodes (`0xC0xx`) in the range
- Delay at least **200ms** between commands
- At most **32** commands per scan

### Recommendations

- Use the app **without** `--unsafe` for everyday battery/status monitoring.
- Set `--model` to your ear codename so EQ limits and feature warnings match the device.
- Do not use scan or SET while updating firmware in the vendor app.

## Device writes (`--unsafe`)

By default only **GET** commands and validated UI SET commands are available.
Passing `--unsafe`:

- Enables raw scan in the UI
- Allows non-UI SET commands that are not permanently blocked
- Requires **two Enter presses** for raw scan confirmation

Prefer running without `--unsafe` for everyday monitoring and normal UI controls.

## Logging privacy

By default trace logs omit `raw_hex`, `raw_without_crc`, and `crc`. Use `--log-raw` only when you need packet dumps for debugging.

Exports from the TUI omit raw bytes unless `--log-raw` was enabled for the session.

## Reporting issues

If you find a way to execute commands outside the allowlist or write to non-RFCOMM paths, please report it with steps to reproduce.
