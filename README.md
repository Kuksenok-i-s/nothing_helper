English · **[Русская версия](README_RU.md)**

# Client for controlling Nothing earbuds

### It probably works with all supported models, but there is no absolute guarantee - bug reports are welcome

# tws_manager - Go SPP client

CLI/TUI and GUI client for **Nothing** and **CMF** earbuds and headsets over Bluetooth RFCOMM (SPP). Reads battery, status, ANC/EQ/spatial/dual, logs packets, and lets you explore the protocol safely without the official app.

This project is not affiliated with Nothing Technology Limited. Nothing/CMF names and related trademarks belong to their respective owners and are used here only to describe compatibility.

Runs **locally** on Linux: discovery via `bluetoothctl`, RFCOMM `/dev/rfcommN`, no network API.

Example

![main window](pics/main_control.png)

![logs and debug](pics/logs.png)

## Features

- **TUI** (Bubble Tea) - devices, control, packet log
- **GUI** (Gio, `-tags gio`) - same functionality in a graphical interface
- **System tray** (`-tags systray`) - status, battery, reconnect from the tray
- **Auto-connect** - discover Nothing/CMF devices and reconnect on link loss
- **Protocol parsing** - battery (L/R/case), status, identity, firmware, ANC, EQ, spatial, dual, lag
- **Tracing** - NDJSON TX/RX session log, JSON export (`captures/`)
- **Desktop notifications** - connect/disconnect, battery, low-battery (`--notify`)
- **Safe by default** - GET commands and a limited set of validated UI SET commands; raw scan and non-UI SET require `--unsafe`

## Requirements

| Component | Purpose |
|-----------|---------|
| Go **1.26+** | build |
| **BlueZ** (`bluetoothctl`, `rfcomm`) | Bluetooth |
| Privileged helper (`polkit`) or `sudo` | bind/release RFCOMM, chown/chmod |
| `vulkan-headers` (Linux) | Gio GUI only |
| `libayatana-appindicator` | system tray (Arch/Manjaro: `pacman -S libayatana-appindicator`) |

Headphones must be **paired** in the system. Default RFCOMM channel is **15** (typical for Nothing Ear).

## Quick start

```bash
# clone and build
git clone <repo-url> tws_manager && cd tws_manager
make build          # bin/tws_manager (TUI)
make test

# TUI - /dev/rfcomm0 already exists
make run ARGS="--device /dev/rfcomm0"

# TUI - create RFCOMM interactively (preflight)
make run

# explicit MAC
go run ./cmd/tws_manager --device /dev/rfcomm0 --addr AA:BB:CC:DD:EE:FF --channel 15

# auto-discover and connect
go run ./cmd/tws_manager --auto

# Gio GUI (tray enabled by default)
make run-gio

# Gio without tray (no libayatana-appindicator)
make run-gio-lite
```

On first run without a ready `/dev/rfcommN`, the TUI will offer to pick a device from discovery and create the RFCOMM node (behavior depends on `--privilege-helper`).

## Build

| Target | Command | Output |
|--------|---------|--------|
| TUI | `make build` | `bin/tws_manager` |
| TUI + tray | `make build-systray` | `bin/tws_manager` |
| Gio + tray | `make build-gio` | `bin/tws_manager_gio` |
| Gio without tray | `make build-gio-lite` | `bin/tws_manager_gio` |
| Tests | `make test` | `go test ./...` |

## CLI flags

Shared by `cmd/tws_manager` and `cmd/tws_manager_gio` (defaults may differ):

| Flag | Default | Description |
|------|---------|-------------|
| `--device` | `/dev/rfcomm0` | RFCOMM device (`/dev/rfcomm[0-9]+`) |
| `--addr` | - | Bluetooth MAC; skips discovery on bind |
| `--channel` | `15` | RFCOMM channel when creating `--device` |
| `--model` | - | codename, product name, or Fast Pair ID |
| `--log` | auto in `captures/` | NDJSON trace path |
| `--log-raw` | `false` | include raw bytes in log/export |
| `--capture-dir` | `captures` | JSON packet export directory |
| `--no-probe` | `false` | skip identity/battery probes after connect |
| `--query-every` | `0` | periodic GET_BATTERY, e.g. `30s` |
| `--unsafe` | `false` | allow SET and raw scan in UI |
| `--auto` | TUI: `false`, Gio: `true` | auto-discover and connect |
| `--notify` | TUI: `false`, Gio: `true` | desktop notifications |
| `--privilege-helper` | TUI: `sudo`, Gio: `auto` | backend for privileged ops: `sudo`, `polkit`, `auto`, `none` |
| `--privilege-helper-path` | - | path to `tws_manager_rfcomm_helper` for `polkit` |

Tracing example:

```bash
go run ./cmd/tws_manager --device /dev/rfcomm0 --log captures/session.ndjson --log-raw
```

## Supported devices

Model is detected from identity, Bluetooth name, or Fast Pair ID. Override explicitly: `--model EarThree`.

| Codename | Product | Main features |
|----------|---------|---------------|
| EarOne | Nothing ear (1) | anc, eq |
| EarTwo | Ear (2) | anc, eq, dual |
| EarTwos | Nothing Ear (2024) | anc, eq, spatial, dual |
| EarThree | Ear (3) | anc, eq, spatial, dual |
| EarStick | Ear (stick) | eq |
| EarColor | Nothing Ear (a) | anc, eq, spatial, dual |
| Flaffy | Nothing ear (open) | eq, dual |
| Elekid | Nothing Headphone (1) | anc, eq, spatial, dual |
| Forretress | Headphone Pro | anc, eq, spatial, headtrack |
| Crobat | CMF Neckband Pro | anc, eq, spatial |
| Corsola | CMF Buds Pro | anc, eq, spatial |
| Donphan | CMF Buds | eq |
| Espeon | CMF Buds Pro 2 | anc, eq, spatial |
| Girafarig, Gligar, … | codename models | see `internal/spp/spp.go` |

Feature commands in the UI: `anc`, `eq`, `spatial`, `lag`, `dual` - gated by each model’s capabilities.

## Interfaces

### TUI

Tabs: **Devices** (discovery, bind, connect), **Control** (GET/SET, toggles for lag/spatial/dual), **Log** (packet history, export). Keys and hints are shown in the bottom status line.

Validated UI SET commands (ANC/EQ/spatial/lag/dual) are available from Control without `--unsafe`. Raw scan requires `--unsafe` and **double Enter** confirmation.

### Gio GUI

Build: `make run-gio` or `go run -tags "gio systray" ./cmd/tws_manager_gio`. Sidebar: devices, control, log. Defaults: `--auto`, `--notify`, and `--privilege-helper=auto`; if the polkit helper is not installed, the GUI prompts for the sudo password in-window.

### System tray

Menu: status, battery, refresh, reconnect, disconnect, quit. On GNOME you may need an AppIndicator extension.

## Autostart and rootless

- Desktop autostart uses the XDG entry: `packaging/common/tws_manager-autostart.desktop`.
- The GUI defaults to `--privilege-helper=auto`: polkit helper first, then sudo fallback with an in-window password prompt.
- Rootless mode expects policy/rules and the helper:
  - `packaging/common/org.tws_manager.rfcomm.policy`
  - `packaging/common/90-tws_manager.rules`
  - `cmd/tws_manager_rfcomm_helper`
- The user must be in the `tws_manager` group (after adding the group, log out and back in).

## Security

- Default: **GET + validated UI SET** (ANC/EQ/spatial/lag/dual). Details: [SECURITY.md](SECURITY.md).
- `--unsafe` - non-UI SET and limited raw scan (`0xC0xx`, delay ≥ 200 ms, max 32 commands).
- Dangerous catalog commands (factory reset, debug mode) are **always blocked**, even with `--unsafe`.
- `polkit` helper is the recommended mode for GUI/autostart; `sudo` remains a fallback.
- Raw bytes in logs only with `--log-raw`.

## Open source readiness

- The public repository contains only project source code and documentation.
- For protocol verification, use your own logs and observations; do not include third-party source code in PRs.
- Project license: [LICENSE](LICENSE) (MIT).

## Packaging

Packaging artifacts live under `packaging/`:

- `packaging/debian` - Debian control/rules/install scripts
- `packaging/arch/PKGBUILD` - Arch/Manjaro package recipe
- `packaging/fedora/tws_manager.spec` - Fedora RPM spec
- `packaging/common` - shared desktop/polkit/sysusers files

Helper targets:

```bash
make build-helper
make build-gio-package
make package-deb
make package-arch
make package-rpm
```

### Post-install (rootless)

- **Debian/Ubuntu (`.deb`)**: `postinst` tries to create the `tws_manager` group and add the user automatically. On failure, it prints the manual `usermod` command.
- **Arch/Manjaro**: post-install hint; run `sudo usermod -aG tws_manager $USER`, then log out and back in.
- **Fedora/RPM**: group is created via `sysusers`; `%post` prints instructions to add the user to `tws_manager`.

## Project layout

```
cmd/tws_manager/          TUI entrypoint
cmd/tws_manager_gio/      Gio GUI entrypoint (-tags gio)
cmd/tws_manager_rfcomm_helper/ privileged helper for polkit
internal/
  app/                    flags, bootstrap, shutdown
  session/                RFCOMM session, read loop, probe
  spp/                    wire format, commands, parsers, models
  bt/                     bluetoothctl, rfcomm, discovery
  connect/                autodiscover, bind, reconnect
  notify/                 desktop notifications
  trace/                  NDJSON log, redaction
  security/               MAC, path, channel validation
  ui/tui/                 Bubble Tea
  ui/gio/                 Gio GUI
  ui/tray/                system tray
  ui/presenter/           shared command catalog for TUI/Gio
```

Agent map and invariants: [AGENTS.md](AGENTS.md).

## Development

```bash
go test ./...                              # all tests
go test ./internal/spp -run Test           # protocol
go test ./internal/session -run Test       # session
gofmt -w cmd internal && make test         # format + tests
```

After changes under `cmd/` or `internal/`, run the full test suite before committing.

## Limitations

- **Linux only** (depends on BlueZ and `/dev/rfcommN`).
- Not a replacement for the official app: OTA, find-my, gesture customization, and some features are not implemented.
- Protocol behavior is reconstructed from observed traffic; unverified models may differ.
