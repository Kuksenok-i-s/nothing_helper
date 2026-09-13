English · **[Русская версия](README_RU.md)**

<h1 align="center">Nothing_helper</h1>
<p align="center"><strong>A compact desktop app for Nothing / CMF earbuds on Linux and macOS. Battery, ANC, EQ, earbud finding, and TALK button controls — from your computer.</strong></p>
<p align="center"><a href="https://github.com/Kuksenok-i-s/nothing_helper/releases/latest">Download the latest release</a> · <a href="LICENSE">MIT</a></p>

<p align="center"><strong>English</strong> · <strong>Русский</strong></p>
<p align="center">
  <img src="pics/companion-en-dark.png" width="350" alt="Nothing_helper — English interface, dark theme, language settings">
  <img src="pics/companion-ru-dark.png" width="350" alt="Nothing_helper — русский интерфейс, тёмная тема, настройки языка">
</p>

<details>
<summary>Light theme — English / Русский</summary>
<p align="center">
  <img src="pics/companion-en-light.png" width="350" alt="Nothing_helper — English interface, light theme">
  <img src="pics/companion-ru-light.png" width="350" alt="Nothing_helper — русский интерфейс, светлая тема">
</p>
</details>

English and Russian interfaces with the language selector inside All settings. Screenshots use demonstration data; expand the section above to see the light theme.

The executable is `nothing_helper`, the Debian package is `nothing-helper`, and the application is **Nothing_helper**.

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

## Requirements and setup

Run commands from the repository root. Install **Go 1.26+**, a C compiler, and enable CGO (`go env CGO_ENABLED` should print `1`). Pair the earbuds in system Bluetooth settings before connecting. Windows is not supported.

### Linux

Debian / Ubuntu:

```bash
make install-deps-debian
```

Arch / Manjaro:

```bash
make install-deps-arch
```

These targets install BlueZ, polkit, graphics headers, the AppIndicator tray library, and PipeWire tools. `make install-deps` is an alias for Debian/Ubuntu only. Install Go 1.26+ separately on Debian/Ubuntu. GNOME may require an AppIndicator extension to display the tray icon. Walkie Talkie detection uses PipeWire's `pw-dump`.

### macOS

Install Go 1.26+ and Xcode Command Line Tools (`xcode-select --install`). Bluetooth and the menu bar use native macOS frameworks; BlueZ, polkit, and AppIndicator are Linux-only dependencies. See [macOS packaging](packaging/macos/README.md) for `.app` and universal DMG builds.

## Quick start

```bash
make help
make build          # GUI + tray -> bin/nothing_helper
./bin/nothing_helper
# Or compile and run directly:
make run ARGS="--addr AA:BB:CC:DD:EE:FF --channel 15"
```

Select the earbuds on the Devices tab; connection and RFCOMM setup run in the background. The executable is `nothing_helper`; the window and macOS app are named **Nothing_helper**.

The GUI starts in English. Open **All settings → Interface language** to switch between English and Русский. The choice is saved in `nothing_helper/interface.json` under your system user configuration directory and applies immediately, even without connected earbuds. Technical logs retain their original text.

## Build variants

```bash
make run-native     # Linux X11/XWayland with system window decorations
make build-native   # same build -> bin/nothing_helper
make run-lite       # GUI without tray
make build-lite     # same build -> bin/nothing_helper
make build-package  # GUI + Linux RFCOMM helper -> bin/
```

`run` / `build` use `gio systray`; `run-native` / `build-native` add `nowayland` and require X11 or XWayland on Linux. The default build supports Wayland and X11. On macOS the standard `make run` uses native window decorations. Lite builds still require Gio graphics dependencies but omit AppIndicator on Linux. Each variant overwrites the same executable.

Equivalent direct Go commands:

```bash
go run -tags "gio systray" ./cmd/nothing_helper
go build -tags "gio systray nowayland" -o bin/nothing_helper ./cmd/nothing_helper
```

`make run-gio`, `make build-gio`, the `*-systray` and `*-gio-lite` targets, and `make build-gio-package` remain compatibility aliases. `GUI_TAGS` overrides tags for the standard GUI build and `check-gui`; for example, `make build GUI_TAGS="gio systray nowayland"`.

## CLI flags

Flags for `cmd/nothing_helper`:

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
| `--privilege-helper-path` | - | path to `nothing_helper_rfcomm_helper` for `polkit` |

Tracing example:

```bash
make run ARGS="--device /dev/rfcomm0 --log captures/session.ndjson --log-raw"
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

## Upgrade from v1.2.0

The published v1.2.0 packages were renamed for downloading only; their installed executable and package IDs remain the old ones. Subsequent builds install `nothing_helper` and `nothing_helper_rfcomm_helper`. Debian, Arch, and RPM recipes declare replacement of the old package. Update custom launch commands and use the `nothing_helper` group for passwordless polkit access; log out and back in after adding group membership.

The app reads the previous `tws_manager/devices.json` configuration when the new `nothing_helper/devices.json` is absent; subsequent saves use the new directory. macOS uses a new bundle identifier and may ask for Bluetooth permission again.

## Autostart and rootless

- Desktop autostart uses the XDG entry: `packaging/common/nothing_helper-autostart.desktop` (`--auto --notify --privilege-helper=polkit`).
- The GUI defaults to `--privilege-helper=auto`: polkit helper first, then sudo fallback with an in-window password prompt.
- Rootless mode expects policy/rules and the helper:
  - `packaging/common/org.nothing_helper.rfcomm.policy`
  - `packaging/common/90-nothing_helper.rules`
  - `cmd/nothing_helper_rfcomm_helper`
- The **`nothing_helper` group is required** for autostart without a polkit password: the polkit rule allows bind/release/chown only for group members. Without the group, every bind via `pkexec` will prompt for the administrator password.
- After `sudo usermod -aG nothing_helper $USER`, a **full logout/login** is required (restarting only the GUI is not enough).
- Verify: `groups | grep nothing_helper`, then `pkexec /usr/libexec/nothing_helper_rfcomm_helper bind --number 0 --addr <MAC> --channel 15 --owner $(id -u):$(id -g)` — no password dialog and `/dev/rfcomm0` should appear.

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
- `packaging/fedora/nothing_helper.spec` - Fedora RPM spec
- `packaging/macos` - universal `.app` + DMG (macOS only)
- `packaging/common` - shared desktop/polkit/sysusers files

Build packages on the target OS, from the repository root:

```bash
make package-deb PKG_VERSION=1.2.2
# dist/nothing-helper_1.2.2-1_<arch>.deb
make bundle-linux PKG_VERSION=1.2.2
# dist/Nothing_helper-1.2.2-linux-<arch>.tar.gz
make macos-app VERSION=1.2.2
# dist/Nothing_helper.app (native architecture)
make package-macos VERSION=1.2.2
# dist/Nothing_helper-1.2.2-universal.dmg (Intel + Apple Silicon)
```

Linux packages and bundles use the current Go toolchain architecture (`go env GOARCH`). `ARCH` does not enable cross-compilation; use an arm64 host with matching CGO dependencies for arm64 builds. The tarball contains only the GUI and helper, without system libraries or polkit installation. `make client-bundle-linux` remains an alias for `make bundle-linux`.

For `.deb`, first run `make install-deps-debian`. The v1.2.0 release packages were built on Ubuntu 24.04 and require the newer GTK/GLib `t64` libraries (Ubuntu 24.04 / Debian 13 generation).

For Arch, install dependencies with `make install-deps-arch`, set `pkgver` in `packaging/arch/PKGBUILD`, then run `make package-arch`. Output: `dist/nothing_helper-<pkgver>-1-<arch>.pkg.tar.zst`.

RPM is a separate Fedora recipe: `make package-rpm` requires a prepared `rpmbuild` tree, a version in `packaging/fedora/nothing_helper.spec`, and the matching `nothing_helper-<version>.tar.gz` source archive in its `SOURCES` directory. Outputs go to the rpmbuild tree's `RPMS` directory; v1.2.0 does not include an RPM asset.

Set local package versions explicitly as above. Without an override, Debian and Linux bundles use `APP_VERSION`, a tag from `GITHUB_REF` in CI, or `0.0.0~dev.<sha>`; checking out a tag locally does not set the version. macOS scripts default to `1.2.2` without `VERSION`. Arch/RPM versions come from their recipes.

### Post-install (rootless)

- **Debian/Ubuntu (`.deb`)**: `postinst` tries to create the `nothing_helper` group and add the user automatically. On failure, it prints the manual `usermod` command.
- **Arch/Manjaro**: the user is **not added to the group automatically** (unlike `.deb`). After installing the package, run `sudo usermod -aG nothing_helper $USER` and log out and back in — otherwise autostart will prompt for a polkit password and will not create `/dev/rfcomm0` for the SPP session.
- **Fedora/RPM**: group is created via `sysusers`; `%post` prints instructions to add the user to `nothing_helper`.

## Project layout

```
cmd/nothing_helper/          Companion entrypoint
cmd/nothing_helper_rfcomm_helper/ privileged helper for polkit
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
make fmt
make check          # vet + tests + race tests
make check-gui      # vet + race tests with gio systray
# Linux X11/XWayland:
make check-gui GUI_TAGS="gio systray nowayland"
go test ./internal/spp -run Test
go test ./internal/session -run Test
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
git tag -a v1.2.3 -m "Nothing_helper v1.2.3"  # example: choose an unused version
git push origin v1.2.3
```

Both release workflows attach artifacts to the same draft GitHub Release; publish it after verifying all artifacts (`Nothing_helper v<version>`). Manual runs: **Actions → Release Linux client / Release macOS client → Run workflow** (optional version override).

## Limitations

- **Linux** is the primary platform (BlueZ, `/dev/rfcommN`, polkit helper).
- **macOS** is experimental (IOBluetooth RFCOMM, Gio GUI).
- Not a replacement for the official app: OTA, gesture customization, and some features are not implemented.
- Protocol behavior is reconstructed from observed traffic; unverified models may differ.
