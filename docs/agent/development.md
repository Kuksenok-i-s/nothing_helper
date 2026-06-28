# Development

## Testing

```bash
go test ./...                              # all tests
go test ./internal/spp -run Test           # protocol
go test ./internal/session -run Test       # session
go test ./internal/bt -run Test            # bluetooth/rfcomm
go test ./internal/trace -run Test         # logging/redaction
gofmt -w cmd internal && make test         # format + tests
```

## Build tags

| Tag | Effect |
|-----|--------|
| `gio` | Gio GUI (`cmd/tws_manager_gio`, `internal/ui/gio/...`) |
| `systray` | Real tray (`internal/ui/tray/tray_systray.go`) vs no-op stub |

## Make targets

```bash
make run                              # TUI
make run-gio                          # Gio + systray
make run-gio-lite                     # Gio without tray
make build-helper                     # polkit helper binary
make package-deb                      # Debian .deb -> dist/
make package-arch                     # Arch pkg -> dist/
make package-macos                    # macOS universal DMG (macOS host)
make client-bundle-linux              # portable Linux tarball -> dist/
```

Requires **Go 1.26+**, BlueZ, Linux. Gio needs `vulkan-headers`; tray needs `libayatana-appindicator`.

**macOS (experimental):** Gio GUI and RFCOMM via IOBluetooth (CGO). No BlueZ/sudo. See [packaging/macos/README.md](../../packaging/macos/README.md) and [macos-adaptation.md](macos-adaptation.md).

```bash
# macOS — Gio GUI
go build -tags "gio systray" -o bin/tws_manager_gio ./cmd/tws_manager_gio
./packaging/macos/bundle.sh && open dist/tws_manager.app

# macOS — full release-style bundle + DMG
VERSION=0.2.0 make package-macos

# macOS — hardware spike (paired device required)
go build -o bin/spp_spike ./cmd/spp_spike
./bin/spp_spike --addr AA:BB:CC:DD:EE:FF --channel 15
```

macOS builds are supported experimentally (IOBluetooth RFCOMM, Gio GUI, systray bundle). See [macos-adaptation.md](macos-adaptation.md) and [packaging/macos/README.md](../../packaging/macos/README.md).

## CI / releases

Workflows in `.github/workflows/`:

| File | Trigger | Jobs |
|------|---------|------|
| `ci.yml` | push, PR | Linux vet/build/test-race; macOS vet/build/test; Debian package smoke (amd64) |
| `release-client-linux.yml` | tag `v*`, `workflow_dispatch` | `.deb` amd64/arm64, Arch x86_64, portable tarball |
| `release-client-macos.yml` | tag `v*`, `workflow_dispatch` | Universal macOS DMG |

Version resolution: [scripts/pkg-version.sh](../../scripts/pkg-version.sh) — git tag `v*`, `APP_VERSION` env, or `0.0.0~dev.<sha>`.

Tag release:

```bash
git tag v0.2.0 && git push origin v0.2.0
```

Linux and macOS release workflows both publish to the same GitHub Release (`append_body: true`).

When changing packaging or Make targets used by CI, verify locally then check the matching workflow job.

## Packaging

| Artifact | Path |
|----------|------|
| Debian | `packaging/debian/` (`make package-deb` symlinks `debian/` and copies `.deb` to `dist/`) |
| Arch | `packaging/arch/PKGBUILD` |
| Fedora | `packaging/fedora/tws_manager.spec` |
| macOS DMG | `packaging/macos/` → `dist/tws_manager-<version>-universal.dmg` |
| Linux portable | `make client-bundle-linux` → `dist/tws_manager-<version>-linux-amd64.tar.gz` |
| Polkit policy | `packaging/common/org.tws_manager.rfcomm.policy` |
| Desktop entry | `packaging/common/*.desktop` |

Post-install: user may need `tws_manager` group membership for rootless RFCOMM (varies by distro — see [README.md](../../README.md)).
