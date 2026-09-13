English · **[Русская версия](README_RU.md)**

<h1 align="center">Nothing_helper</h1>
<p align="center"><strong>A compact desktop app for **Nothing / CMF** earbuds on Linux and macOS. Battery, ANC, EQ, earbud finding, and TALK button controls — from your computer.</strong></p>
<p align="center"><a href="https://github.com/Kuksenok-i-s/nothing_helper/releases/latest">Download the latest release</a> · <a href="LICENSE">MIT</a></p>

<p align="center">
  <img src="pics/companion.png" width="350" alt="Nothing_helper — dark theme">
  <img src="pics/companion-light.png" width="350" alt="Nothing_helper — light theme">
</p>

Dark and light themes. Interface screenshots use demonstration data.

Linux package identifiers and the `tws_manager` command are retained for upgrade compatibility.

An independent community project, not affiliated with Nothing Technology Limited. macOS support is experimental; features vary by model.

## Features

- **Nothing_helper** (Gio, `-tags gio`) — compact charge/ANC controls, expanded sound settings, device picker, diagnostics
- **Walkie Talkie (Ear (3))** — enable case TALK controls after starting a Bluetooth headset microphone stream in a call or recorder; Super Mic is enabled first. The app does not record audio.
- **Find earbuds** — click an earbud silhouette to ring it; wear-state checks and an automatic stop limit protect against ringing in your ear
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
make build          # bin/tws_manager — GUI + tray
make run            # compact companion
make run-gio-lite   # GUI without system tray
make run ARGS="--addr AA:BB:CC:DD:EE:FF --channel 15"
make test
```

Pair the headphones in system Bluetooth settings first. Use Devices to discover and select a device; connection and RFCOMM setup run in the background.

## Build

| Target | Command | Output |
|--------|---------|--------|
| Gio + tray | `make build-gio` | `bin/tws_manager` |
| Gio without tray | `make build-gio-lite` | `bin/tws_manager` |
| Tests | `make test` | `go test ./...` |

## CLI flags

Flags for `cmd/tws_manager`:

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
| `--query-every` | `0` (or `60s` when `--notify`) | periodic GET_BATTERY, e.g. `30s` |
| `--unsafe` | `false` | enable non-UI unsafe protocol operations (no raw scan UI) |
| `--auto` | `true` | auto-discover and connect |
| `--notify` | `true` | desktop notifications (needs `gdbus` or `libnotify` / `notify-send`) |
| `--privilege-helper` | `auto` | backend for privileged ops: `sudo`, `polkit`, `auto`, `none` |
| `--privilege-helper-path` | - | path to `tws_manager_rfcomm_helper` for `polkit` |

Tracing example:

```bash
go run -tags gio ./cmd/tws_manager --device /dev/rfcomm0 --log captures/session.ndjson --log-raw
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
| Girafarig, Gligar, … | codename models | see `internal/spp/models.go` |

Feature commands in the UI: `anc`, `eq`, `spatial`, `lag`, `dual` - gated by each model’s capabilities.

## Interfaces

### Nothing_helper

`make run` opens the new compact GUI. Sound shows confirmed battery and ANC state; All settings reveals EQ, ANC intensity, spatial audio, low latency, and dual-device controls when supported by the model. Devices handles discovery and connection; Diagnostics contains technical state and JSON export. Walkie Talkie controls are available on Ear (3) while a Bluetooth headset microphone stream is active. On Linux, this check requires PipeWire and `pw-dump`; on macOS it uses CoreAudio.

Dark and light appearances are available. Closing a tray-enabled build hides the window; Open companion restores it, and Quit exits. Builds without tray support exit on close. The old Bubble Tea TUI and previous Gio screens have been removed.

### System tray

Menu: status, battery, refresh, reconnect, disconnect, quit. On GNOME you may need an AppIndicator extension.

## Autostart and rootless

- Desktop autostart uses the XDG entry: `packaging/common/tws_manager-autostart.desktop` (`--auto --notify --privilege-helper=polkit`).
- The GUI defaults to `--privilege-helper=auto`: polkit helper first, then sudo fallback with an in-window password prompt.
- Rootless mode expects policy/rules and the helper:
  - `packaging/common/org.tws_manager.rfcomm.policy`
  - `packaging/common/90-tws_manager.rules`
  - `cmd/tws_manager_rfcomm_helper`
- The **`tws_manager` group is required** for autostart without a polkit password: the polkit rule allows bind/release/chown only for group members. Without the group, every bind via `pkexec` will prompt for the administrator password.
- After `sudo usermod -aG tws_manager $USER`, a **full logout/login** is required (restarting only the GUI is not enough).
- Verify: `groups | grep tws_manager`, then `pkexec /usr/libexec/tws_manager_rfcomm_helper bind --number 0 --addr <MAC> --channel 15 --owner $(id -u):$(id -g)` — no password dialog and `/dev/rfcomm0` should appear.

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
- `packaging/macos` - universal `.app` + DMG (macOS only)
- `packaging/common` - shared desktop/polkit/sysusers files

Helper targets:

```bash
make build-helper
make build-gio-package
make package-deb              # .deb -> dist/tws-manager_<version>-1_<arch>.deb
make package-arch             # Arch pkg -> dist/
make package-rpm
make package-macos            # macOS: dist/Nothing_helper-<version>-universal.dmg
make client-bundle-linux      # portable tarball -> dist/Nothing_helper-<version>-linux-amd64.tar.gz
```

Release version for local packaging (optional):

```bash
./scripts/pkg-version.sh      # tag / APP_VERSION / 0.0.0~dev.<sha>
VERSION=0.2.0 make package-macos
PKG_VERSION=0.2.0 make client-bundle-linux
```

### Post-install (rootless)

- **Debian/Ubuntu (`.deb`)**: `postinst` tries to create the `tws_manager` group and add the user automatically. On failure, it prints the manual `usermod` command.
- **Arch/Manjaro**: the user is **not added to the group automatically** (unlike `.deb`). After installing the package, run `sudo usermod -aG tws_manager $USER` and log out and back in — otherwise autostart will prompt for a polkit password and will not create `/dev/rfcomm0` for the SPP session.
- **Fedora/RPM**: group is created via `sysusers`; `%post` prints instructions to add the user to `tws_manager`.

## Project layout

```
cmd/tws_manager/          Companion entrypoint
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
  ui/companion/                 Gio GUI
  ui/tray/                system tray
  ui/presenter/           shared command catalog for Companion
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

## CI / releases

GitHub Actions workflows under `.github/workflows/`:

| Workflow | Trigger | What it does |
|----------|---------|--------------|
| `ci.yml` | push, pull_request | Linux vet/build/test-race; macOS vet/build/test; Debian `.deb` smoke build (amd64) |
| `release-client-linux.yml` | tag `v*`, manual | `.deb` (amd64/arm64), Arch pkg (x86_64), portable tarball; publishes to GitHub Releases |
| `release-client-macos.yml` | tag `v*`, manual | Universal macOS DMG; publishes to GitHub Releases |

Cut a release:

```bash
git tag v0.2.0 && git push origin v0.2.0
```

Both release workflows attach artifacts to the same draft GitHub Release; publish it after verifying all artifacts (`Nothing_helper v<version>`). Manual runs: **Actions → Release Linux client / Release macOS client → Run workflow** (optional version override).

## Limitations

- **Linux** is the primary platform (BlueZ, `/dev/rfcommN`, polkit helper).
- **macOS** is experimental (IOBluetooth RFCOMM, Gio GUI).
- Not a replacement for the official app: OTA, gesture customization, and some features are not implemented.
- Protocol behavior is reconstructed from observed traffic; unverified models may differ.
