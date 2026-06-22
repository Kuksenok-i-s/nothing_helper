# Safety model

Rules enforced on **every** TX path. Agents must not bypass these layers.

## Default mode (no `--unsafe`)

| Allowed | Blocked |
|---------|---------|
| GET commands (`0xC0xx`) via UI catalog | SET with `Unsafe: true` (always) |
| UI-validated SET: ANC, EQ, spatial, lag, dual | Arbitrary SET payloads |
| | Raw scan UI entry |

Catalog example (`internal/spp/battery.go`):

```go
0xF00F: {Name: "set_current_noise_reduction", Kind: "set", Safe: true},
0xF03D: {Name: "restore_factory_setting", Kind: "set", Unsafe: true}, // blocked even with --unsafe
0xF041: {Name: "set_custom_eq", Kind: "set"},                          // needs --unsafe (not Safe)
```

## With `--unsafe`

- Non-UI SET allowed (except `Unsafe: true` catalog entries).
- Raw scan menu entry shown; TUI requires double Enter confirmation.
- `Session` created with `allowUnsafe: true` from `--unsafe` flag.

## Scan rules

```go
// internal/spp/device_safety.go
const (
    MinScanDelay    = 200 * time.Millisecond
    MaxScanCommands = 32
)

func SafeScanCommand(cmd uint16) bool {
    return cmd&0xF000 == 0xC000  // internal/spp/scan.go
}
```

User syntax in TUI comment field: `scan c001 c020 500ms`

Parsed by:

```go
start, end, delay, err := spp.ParseScanCommand([]string{"scan", "c001", "c020", "500ms"})
err = spp.ValidateScanRange(start, end, delay)
```

## Enforcement chain

```
UI: presenter.BuildCommands(allowUnsafe)
  → actions.Execute / ExecuteScan
    → session.Send / RunQueryScan
      → authorizeCommand(cmd)
        → spp.CommandInfoFor(cmd)  // Safe / Unsafe / Kind
      → spp.FeatureCommandPacket   // payload builders for feature SET
      → spp.ValidateScanRange      // scan only
```

## authorizeCommand (reference)

```go
func (s *Session) authorizeCommand(cmd uint16) error {
    info := spp.CommandInfoFor(cmd)
    if info.Unsafe {
        return fmt.Errorf("command %s is blocked for device safety", spp.CommandLabel(cmd))
    }
    if info.Kind == "set" && !info.Safe && !allowUnsafe {
        return fmt.Errorf("command %s requires --unsafe", spp.CommandLabel(cmd))
    }
    return nil
}
```

## SET payload validation (UI-safe)

Built in `internal/spp/device_safety.go`:

| Feature | Builder | Notes |
|---------|---------|-------|
| ANC | `BuildANCSetPayload` | Allowlist 0-5,7,252-255; off uses `ancOffModeValue(model)` |
| EQ | `BuildEQSetPayload` | Max preset from `MaxEQMode(model)` (3 or 7) |
| Spatial | inline in `feature_commands.go` | on/off only |
| Lag | inline | on/off only |
| Dual | inline | on/off; connect/disconnect via device list |

## Tests agents should mirror

```go
// Permanently blocked
err := s.SendCommand(0xF03D, Meta{Source: "test"}) // must fail: "blocked"

// SET without --unsafe
err := s.SendCommand(0xF041, Meta{Source: "test"}) // must fail: "--unsafe"

// UI-safe SET without --unsafe
err := s.authorizeCommand(spp.CmdSetNoiseReduction) // must succeed
```

See [SECURITY.md](../../SECURITY.md) and [examples.md](examples.md).
