# Overview

**For agents:** local Bluetooth client (Linux primary; macOS Gio experimental). No network API. Default safe mode. Read [agent-guide.md](agent-guide.md) and [invariants.md](invariants.md) before editing.

**tws_manager** talks to Nothing/CMF earbuds over RFCOMM SPP. On Linux: `/dev/rfcommN` and BlueZ. On macOS: IOBluetooth (experimental).

| Capability | Details |
|------------|---------|
| Discovery | Filter Nothing/CMF devices via `bluetoothctl` |
| RFCOMM | Bind/release/revive `/dev/rfcommN`, fix permissions |
| Protocol | Custom Nothing SPP framing (SOF/CRC/FSN) |
| Parsed data | Battery, status, identity, firmware, ANC, EQ, spatial, dual, lag |
| UIs | TUI (Bubble Tea), GUI (Gio), system tray |
| Safety | GET + validated UI SET by default; `--unsafe` for raw scan / other SET |
| Logging | NDJSON trace + JSON export (`captures/`) |

## Entry points

| Binary | Path | Build tags | Role |
|--------|------|------------|------|
| `tws_manager` | `cmd/tws_manager/main.go` | optional `systray` | TUI + stdin RFCOMM preflight |
| `tws_manager_gio` | `cmd/tws_manager_gio/main.go` | `gio`, optional `systray` | GUI |
| `tws_manager_rfcomm_helper` | `cmd/tws_manager_rfcomm_helper/main.go` | — | polkit helper |

Minimal entrypoint pattern:

```go
cfg, _ := app.ConfigFromFlags(app.RegisterFlags(fs, app.ProfileCLI))
app.Run(ctx, cfg, func(ctx context.Context, rt *app.Runtime) error {
    services, _ := app.WireServices(ctx, rt)
    return tui.Run(ctx, rt.Session, tui.Options{Manager: services.Manager, /* … */})
})
```

## Hub files (start search here)

| Area | File |
|------|------|
| Command catalog | `internal/spp/battery.go` |
| Session / TX-RX | `internal/session/session.go` |
| Event bus | `internal/session/events_publish.go` |
| Bluetooth | `internal/bt/bt.go`, `internal/bt/bluetoothctl.go` |
| UI commands | `internal/ui/presenter/commands.go` |

## Scale

~86 Go files, 137 import edges, 21 test files, 0 circular deps (SocratiCode graph).
