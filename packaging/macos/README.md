# Nothing_helper — macOS build and packaging

Run all commands from the repository root. The GUI uses Gio, Bluetooth uses IOBluetooth, and the menu bar uses native `NSStatusItem`. Linux libraries such as BlueZ and AppIndicator are not required on macOS.

## Prerequisites

- Go 1.26+ and Xcode Command Line Tools (`xcode-select --install`).
- A macOS version supported by the installed Go toolchain and Xcode SDK.
- CGO enabled: `go env CGO_ENABLED` should print `1`.
- Earbuds paired in system Bluetooth settings for live testing.

## Development

```bash
make help
make build                         # native executable: bin/nothing_helper
make run                           # GUI with menu bar
make run-lite                      # GUI without menu bar; exits on window close
make run ARGS="--addr AA:BB:CC:DD:EE:FF --channel 15"
make check-gui                     # vet + race tests with gio systray
```

Equivalent direct command: `go run -tags "gio systray" ./cmd/nothing_helper`.
The old TUI and separate Gio entrypoint were removed. `make run-gio` and `make build-gio` are compatibility aliases. The `nowayland` tag is only useful for selecting X11/XWayland on Linux; macOS uses native window decorations already.

## Native .app

```bash
make macos-app VERSION=1.2.0
open dist/Nothing_helper.app
```

This builds for the host architecture and assembles `dist/Nothing_helper.app`. Its executable is `Contents/MacOS/Nothing_helper`; the unbundled development executable is `bin/nothing_helper`.

## Universal installer (Apple Silicon + Intel)

```bash
make package-macos VERSION=1.2.0
open dist/Nothing_helper-1.2.0-universal.dmg
```

Outputs:

- `dist/build/macos/nothing_helper-universal` — combined arm64 and x86_64 binary.
- `dist/Nothing_helper.app` — universal application bundle.
- `dist/Nothing_helper-1.2.0-universal.dmg` — installer with an Applications shortcut.

Drag **Nothing_helper.app** to **Applications**. Always pass `VERSION` for local packaging; scripts default to `0.1.0`. In CI, the workflow resolves the tag or manual version input and passes `VERSION` to Make. These builds require macOS and the Xcode SDK; setting `GOOS=darwin` on Linux is insufficient for the native frameworks.

Verify the bundle:

```bash
lipo -info dist/Nothing_helper.app/Contents/MacOS/Nothing_helper
codesign --verify --deep --strict dist/Nothing_helper.app
```

## Scripts

- `build-universal.sh` builds arm64 and amd64, then merges them with `lipo`.
- `bundle.sh` assembles `.app`, generates `.icns`, and ad-hoc signs the app; used by `make macos-app`.
- `mk-dmg.sh` creates the compressed DMG.
- `package.sh` runs all universal packaging steps; used by `make package-macos`.

## Bluetooth diagnostics and profiling

Use the Devices and Diagnostics tabs in the application to check connection, battery responses, and export a session. For raw protocol tracing:

```bash
make run ARGS="--addr AA:BB:CC:DD:EE:FF --channel 15 --log captures/session.ndjson --log-raw"
```

For a running installed app, capture a macOS stack sample:

```bash
make sample-macos-app PROFILE_SECONDS=20
# For an unbundled development process instead:
make sample-macos-app MACOS_PROCESS=nothing_helper PROFILE_SECONDS=20
```

The old `cmd/spp_spike` utility is no longer present; use the application's diagnostics instead.

## Permissions and signing

Allow Bluetooth access when macOS prompts for it. Denied access appears as connection errors. The distributed app is ad-hoc signed, not notarized with an Apple Developer ID. Developer ID signing and notarization require separate credentials and are not performed by these Make targets.

## CI release

A new `v*` tag starts the Linux and macOS release workflows. The macOS workflow tests the GUI, builds the universal DMG, and attaches it plus a SHA-256 file to a draft GitHub Release. Publish the draft after both platforms finish and their artifacts are verified. Manual workflow runs upload Actions artifacts but do not publish a release.

After downloading both files into the same directory:

```bash
shasum -a 256 -c Nothing_helper-1.2.0-universal.dmg.sha256
```
