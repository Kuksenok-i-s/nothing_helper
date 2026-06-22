# Package reference

**For agents:** use this to pick the right package. Code patterns in [examples.md](examples.md).

---

## `internal/session` — orchestration hub

Owns RFCOMM fd, device state, TX/RX.

| API | Agent notes |
|-----|-------------|
| `New(logger, allowUnsafe, probeEnabled)` | `allowUnsafe` mirrors `--unsafe` |
| `Connect(device, rfcommPath, channel)` | Starts `readLoop` + optional `initialProbe`; idempotent same-MAC |
| `Send` / `SendCommand` | Always through `authorizeCommand`; assigns FSN |
| `FeaturePacket(fields)` | Delegates to `spp.FeatureCommandPacket` |
| `RunQueryScan(ctx, start, end, delay)` | Unsafe GET scan only |
| `Events()` / `Subscribe()` | UI + notify subscribe here |
| `Snapshot()` | Read batteries, model, config, connected flag |

```go
err := sess.SendCommand(spp.CmdGetBattery, session.Meta{Source: "ui", Trigger: "Info: battery"})
snap := sess.Snapshot() // snap.Batteries, snap.Model, snap.Config["anc"]
```

**Event kinds:** `packet_rx`, `packet_tx`, `battery`, `connected`, `disconnected`, `error`, `model`, `progress`.

**Config keys in `Snapshot.Config`:** `anc`, `lag`, `dual`, `eq`, `spatial`.

Files: `session.go`, `events_publish.go`, `livestate.go`.

---

## `internal/spp` — wire format and protocol

| File | Edit when… |
|------|------------|
| `commands.go` | New opcode constants |
| `battery.go` | `commandCatalog` metadata |
| `frame.go` | Frame layout, SOF, CRC, FSN |
| `wire.go` | Decode/read stream |
| `parse_dispatch.go` | New parsers, `ParsePacket` routing |
| `models.go` | Device models, features |
| `feature_commands.go` | UI feature GET/SET builders |
| `device_safety.go` | SET validation, scan limits |
| `scan.go` | Scan comment parsing |

Frame: `[SOF][control LE][cmd LE][len LE][fsn][payload][crc LE]`

Response match: FSN echo, else `0x40xx` → request `0xC0xx`.

---

## `internal/bt` — BlueZ and RFCOMM

| API | Purpose |
|-----|---------|
| `DiscoverDevices` | List candidates |
| `BindRFCOMMDevice` / `ReleaseRFCOMMDevice` | rfcomm bind/release |
| `ReviveRFCOMMDevice` | release → bind → chmod → verify open |
| `OpenRFCOMMDevice` | Used by `session.Connect` |
| `ConfigurePrivileges` | sudo / polkit / auto / none |
| `ResolveDeviceChannel` | Per-MAC cache or default 15 |

Privilege helper: `cmd/tws_manager_rfcomm_helper` — `bind`, `release`, `fix-perms` only.

---

## `internal/connect` — UI-friendly connection

Wraps `bt` + `session` for TUI/Gio.

```go
mgr := connect.New(sess, connect.Options{RFCOMMPath: "/dev/rfcomm0", Channel: 15})
err := mgr.ConnectBest(ctx, statusFn)
go mgr.AutoConnect(ctx, connect.AutoOptions{Interval: 5 * time.Second})
```

Uses `internal/audio` to wait for active Bluetooth sink when needed.

---

## `internal/ui/presenter` + `internal/ui/actions`

Shared UI command layer — **edit here first** for new menu commands.

```go
cmds := presenter.BuildCommands(model, dualList, allowUnsafe)
res := actions.Execute(sess, cmd, actions.ExecOpts{Source: "ui"})
err := actions.ExecuteScan(ctx, sess, scanFields)
```

TUI: `internal/ui/tui`. Gio: `internal/ui/gio` (build tag `gio`).

---

## `internal/trace`

NDJSON logger. Raw hex only with `--log-raw`.

```go
logger, _ := trace.NewLogger(path, logRaw)
logger.LogTX(raw, pkt, ctx)
```

---

## `internal/security`

All external paths/MACs/channels validated here.

```go
security.ValidateRFCOMMDevice("/dev/rfcomm0")  // required pattern
security.ValidateChannel(15)                   // 1..63
security.NormalizeMAC(mac)
```

---

## `internal/app`

Flags, bootstrap, shutdown. Shared by TUI and Gio.

```go
app.RegisterFlags(fs, app.ProfileCLI)  // or ProfileGUI
rt, _ := app.Bootstrap(ctx, cfg)
app.WireServices(ctx, rt)
app.Run(ctx, cfg, runFn)
```

---

## Other packages

| Package | Role |
|---------|------|
| `internal/notify` | Desktop notifications on session events |
| `internal/dualpolicy` | `--pc-primary ask\|off` |
| `internal/audio` | Sink readiness for connect |
| `internal/ui/tray` | Systray (`-tags systray`) |
| `internal/ui/dualprompt` | Dual device picker |
