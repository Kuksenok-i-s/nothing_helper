# macOS packaging

Build a universal (Apple Silicon + Intel) `.app` bundle and DMG installer.

**Note:** On macOS the tray uses a native `NSStatusItem` (not `getlantern/systray`), because that library's `NSApplication` loop conflicts with Gio and crashes at startup. Linux uses AppIndicator via `getlantern/systray`.

## Prerequisites

- macOS 11+
- Xcode Command Line Tools (`xcode-select --install`)
- Go 1.26+ with `CGO_ENABLED=1` (required by systray)
- Paired Nothing/CMF device for live Bluetooth tests

## Quick build (recommended)

From the repo root:

```bash
make package-macos
```

Output:

| Artifact | Path |
|----------|------|
| Universal binary | `dist/build/macos/tws_manager-universal` |
| App bundle | `dist/tws_manager.app` |
| DMG installer | `dist/tws_manager-0.1.0-universal.dmg` |

Install: open the DMG and drag **tws_manager.app** to **Applications**.

Custom version:

```bash
VERSION=0.2.0 make package-macos
# -> dist/tws_manager-0.2.0-universal.dmg
```

## Scripts

| Script | Purpose |
|--------|---------|
| `build-universal.sh` | Build `arm64` + `amd64`, merge with `lipo` |
| `bundle.sh` | Assemble `.app`, generate `.icns`, ad-hoc sign |
| `mk-dmg.sh` | Create compressed DMG with Applications symlink |
| `package.sh` | Run all steps above |

Local dev bundle (native arch only, no DMG):

```bash
./packaging/macos/bundle.sh
open dist/tws_manager.app
```

## Architecture

The universal binary contains both slices:

```bash
lipo -info dist/build/macos/tws_manager-universal
# Architectures in the fat file: ... are: x86_64 arm64
```

Cross-compilation uses the Xcode SDK; build on a Mac with CLT installed.

## Hardware spike

Before relying on the GUI for connect, verify RFCOMM with:

```bash
go build -o bin/spp_spike ./cmd/spp_spike
./bin/spp_spike --addr AA:BB:CC:DD:EE:FF --channel 15
```

Pass criteria: parsed GET_BATTERY response within the timeout.

Scan all RFCOMM channels (1s timeout per channel):

```bash
chmod +x scripts/macos-rfcomm-scan.sh
./scripts/macos-rfcomm-scan.sh --addr AA:BB:CC:DD:EE:FF --build
./scripts/macos-rfcomm-scan.sh --addr AA:BB:CC:DD:EE:FF --from 14 --to 16 --probe battery
```

## Permissions

First Bluetooth API use triggers the macOS Privacy → Bluetooth prompt. Denied permission surfaces as connect errors in the UI/log.

## Notarization (optional)

For distribution outside your machine, sign with a Developer ID certificate and notarize via `notarytool`. Ad-hoc signing (`codesign -s -`) is sufficient for local development.

## CI release

On tag `v*` (or manual **Actions → Release macOS client**), GitHub Actions runs `make package-macos` on `macos-14` and publishes `dist/tws_manager-<version>-universal.dmg` to GitHub Releases. Version comes from [scripts/pkg-version.sh](../../scripts/pkg-version.sh).
