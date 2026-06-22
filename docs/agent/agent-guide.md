# Agent guide

Operating rules for AI agents in **tws_manager**.

## Do

- Use SocratiCode or [hot-files.md](hot-files.md) before opening files.
- Keep diffs minimal; one concern per change.
- Add/update tests in the same package as changed behaviour.
- Preserve safety invariants ([invariants.md](invariants.md)).
- Use existing naming: `CmdGet…` / `CmdSet…` / `CmdRsp…`, `parse…Packet`, `Build…Payload`.
- Run `go test ./...` (or targeted package) after Go changes.

## Do not

- Weaken `--unsafe`, scan limits, or permanently-blocked commands.
- Allow RFCOMM paths other than `/dev/rfcommN`.
- Log or export raw bytes by default (only with `--log-raw`).
- Add network APIs or cloud dependencies — this is a local Bluetooth client.
- Duplicate connect/read loops (same-MAC reconnect must stay idempotent).
- Guess undocumented wire formats — ask the user for captures or device traces.

## Layer you should edit

| Task | Primary packages | Avoid |
|------|------------------|-------|
| Wire format / parsing | `internal/spp` | UI |
| TX/RX / connect lifecycle | `internal/session`, `internal/bt` | direct UI |
| Discover / autoconnect | `internal/connect` | spp parsers |
| Shared UI commands | `internal/ui/presenter`, `internal/ui/actions` | duplicating in tui/gio |
| TUI-only layout | `internal/ui/tui` | session internals |
| Gio-only layout | `internal/ui/gio/view`, `internal/ui/gio/state` | session internals |
| Flags / bootstrap | `internal/app` | — |
| Input validation | `internal/security` | ad-hoc regex in UI |

## Decision tree

```
User request
├─ "add command / parse response"
│   └─ spp: catalog → parser → tests → (optional) presenter featureCommands
├─ "UI button / menu item"
│   └─ presenter.BuildCommands → actions.Execute (usually no session changes)
├─ "connect fails / rfcomm"
│   └─ bt.OpenRFCOMMDevice / ReviveRFCOMMDevice → session.Connect idempotency
├─ "block / allow command"
│   └─ commandCatalog Safe/Unsafe → authorizeCommand → tests
└─ "new ear model"
    └─ models.go knownModels → feature list → battery mapping if needed
```

## Ask the user when

- Opcode or payload layout is unknown (need NDJSON capture or `--log-raw` dump).
- Behaviour differs by firmware version and no fixture exists.
- Change would alter default safety posture.
- Scope spans packaging, CI, or non-Go directories.
- macOS port work — read [macos-adaptation.md](macos-adaptation.md) first; Bluetooth hardware spike is the gate.

## Verify before marking done

```bash
go test ./internal/spp ./internal/session ./internal/bt -count=1   # common touch points
go test ./...                                                     # full suite
gofmt -w cmd internal                                             # formatting
```

See [pr-checklist.md](pr-checklist.md).
