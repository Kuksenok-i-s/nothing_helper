# Agent workflows

Step-by-step tasks with pointers to [examples.md](examples.md).

---

## Add a new protocol command / parser

**When:** new GET/RSP/SET opcode or event needs parsing or UI exposure.

1. Add constant in `internal/spp/commands.go` (if new opcode).
2. Add `commandCatalog` entry in `internal/spp/battery.go`:

```go
0x40XX: {Name: "rsp_my_feature", Kind: "binary"},  // or "text", "battery_pairs", …
0xC0XX: {Name: "get_my_feature", Kind: "query"},
0xF0XX: {Name: "set_my_feature", Kind: "set", Safe: true}, // or Unsafe: true
```

3. Register parser in `internal/spp/parse_dispatch.go`:

```go
CmdRspMyFeature: parseMyFeaturePacket("my_feature_response"),
```

4. Add test in `internal/spp/spp_test.go` with real or captured payload bytes.
5. If UI feature: extend `featureCommands` in `feature_commands.go`, add entries in `presenter.BuildCommands`, validate SET payload in `device_safety.go`.

**Verify:** `go test ./internal/spp -run TestMyFeature -v`

**Ask user if:** payload layout is unknown — request NDJSON line or hex dump from `--log-raw`.

---

## Change safety behaviour

**When:** block/allow a command, tighten SET validation, adjust scan limits.

| Change | File |
|--------|------|
| Catalog `Safe` / `Unsafe` flags | `internal/spp/battery.go` |
| SET payload validation | `internal/spp/device_safety.go` |
| TX gate | `internal/session/session.go` → `authorizeCommand` |
| Scan limits | `device_safety.go` (`MinScanDelay`, `MaxScanCommands`, `ValidateScanRange`) |
| UI visibility | `internal/ui/presenter/commands.go` |

**Required tests:**

```bash
go test ./internal/spp -run TestValidate
go test ./internal/session -run TestSendBlocks
go test ./internal/ui/presenter -run TestBuildCommands
```

See [examples.md § Session safety tests](examples.md#session-tx--rx--safety).

---

## Fix RFCOMM / connect issues

**When:** bind fails, stale `/dev/rfcommN`, reconnect loops, permission errors.

| Symptom | Check |
|---------|-------|
| Permission denied | `--privilege-helper`, polkit helper path, `tws_manager` group |
| Stale device node | `bt.ReviveRFCOMMDevice` in `internal/bt/bt.go` |
| Double connect wedge | `session.Connect` idempotent guard (same MAC) |
| Auto-reconnect spam | `internal/connect/autodiscover.go` → `AutoConnect`, `ConnectBest` |
| Wrong channel | `bt.ResolveDeviceChannel`, cached channel in `config_cache.go` |

**Tests:** `go test ./internal/bt ./internal/connect ./internal/session -v`

**Do not:** bypass `security.ValidateRFCOMMDevice` or open arbitrary device paths.

---

## UI change (TUI + Gio)

**When:** new menu item, toggle, or command label.

1. Add to `presenter.BuildCommands` (shared) — see [examples.md § UI](examples.md#ui-presenter--actions).
2. Execution already goes through `actions.Execute` — no duplicate Send logic in TUI/Gio.
3. TUI-only keys/layout: `internal/ui/tui/tui.go`.
4. Gio-only widgets: `internal/ui/gio/view/`, state in `internal/ui/gio/state/`.

**Toggle features** (lag/spatial/dual): add both `set on` and `set off` entries; `ToggleFeatures()` picks them up.

**Scan:** only via `actions.ExecuteScan`, never `Execute`.

---

## Add / update device model

**When:** new Nothing/CMF product or wrong feature gating.

Edit `internal/spp/models.go`:

```go
{Codename: "MyModel", Product: "Nothing …", FastPairID: "XXXXXX",
 Protocol: "EarTwosProtocol", Tier: "B+",
 BatteryCaseSource: "case", // or "stereo" for headphones
 Features: []string{"anc", "eq", "spatial", "dual"},
 Aliases: []string{"my model", "nothing …"}},
```

- `BatteryCaseSource: "stereo"` → `NormalizeBatteryForModel` maps stereo id to case.
- Run `go test ./internal/spp -run TestNormalizeBattery -v`.

**Ask user if:** Fast Pair ID or protocol tier is unknown.

---

## Debugging protocol issues

```bash
# Capture session (user runs on Linux with device)
go run ./cmd/tws_manager --device /dev/rfcomm0 \
  --log captures/session.ndjson --log-raw

# Agent-side unit tests
go test ./internal/spp -run TestParse -v
go test ./internal/session -run TestMatchRequest -v
```

1. Find failing opcode in NDJSON `command` / `raw_hex` fields.
2. Add fixture test with exact payload bytes.
3. Fix parser in `parse_dispatch.go`; re-run tests.

---

## Run before any PR

See [pr-checklist.md](pr-checklist.md) and [development.md](development.md).
