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
```

Requires **Go 1.26+**, BlueZ, Linux. Gio needs `vulkan-headers`; tray needs `libayatana-appindicator`.

**macOS (experimental):** Gio GUI and RFCOMM via IOBluetooth (CGO). No BlueZ/sudo. See [packaging/macos/README.md](packaging/macos/README.md) and [docs/agent/macos-adaptation.md](docs/agent/macos-adaptation.md).

```bash
# macOS — Gio GUI
go build -tags "gio systray" -o bin/tws_manager_gio ./cmd/tws_manager_gio
./packaging/macos/bundle.sh && open dist/tws_manager.app

# macOS — hardware spike (paired device required)
go build -o bin/spp_spike ./cmd/spp_spike
./bin/spp_spike --addr AA:BB:CC:DD:EE:FF --channel 15
```

macOS builds are supported experimentally (IOBluetooth RFCOMM, Gio GUI, systray bundle). See [macos-adaptation.md](macos-adaptation.md) and [packaging/macos/README.md](../../packaging/macos/README.md).

## Packaging

| Artifact | Path |
|----------|------|
| Debian | `packaging/debian/` |
| Arch | `packaging/arch/PKGBUILD` |
| Fedora | `packaging/fedora/tws_manager.spec` |
| Polkit policy | `packaging/common/org.tws_manager.rfcomm.policy` |
| Desktop entry | `packaging/common/*.desktop` |

Post-install: user may need `tws_manager` group membership for rootless RFCOMM (varies by distro — see [README.md](../../README.md)).
