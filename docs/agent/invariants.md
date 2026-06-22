# Invariants

**Agents: treat these as hard constraints.** Violating them requires explicit user approval.

## 1. RFCOMM path

Only `/dev/rfcommN` is valid.

```go
path, err := security.ValidateRFCOMMDevice(device) // rejects /dev/ttyUSB0, relative paths, etc.
```

## 2. Safety guardrails

- Never remove or loosen `--unsafe` gating, scan limits, or permanent blocks (`Unsafe: true`).
- Every TX must pass `session.authorizeCommand`.

## 3. Scan

- Opcodes: `cmd & 0xF000 == 0xC000` (GET only)
- Delay ≥ `200ms`
- Max `32` commands per scan

## 4. Idempotent connect

Same MAC while link is live → return nil, no second `readLoop`:

```go
if s.f != nil && device.MAC != "" && s.device.MAC == device.MAC {
    return nil // already connected
}
```

## 5. Raw byte redaction

Trace/export omit raw hex unless `--log-raw`. Do not change default to include raw bytes.

## 6. Battery stereo mapping

For models with `BatteryCaseSource == "stereo"`, always run:

```go
data, warnings := spp.NormalizeBatteryForModel(data, model)
```

before UI display (Elekid, etc.).

## 7. FSN lifecycle

`spp.ResetFSN()` on each successful `Connect`. TX assigns FSN via `NextFSN()` in `Send`.

## 8. Privileged operations

Polkit helper accepts only `bind`, `release`, `fix-perms` with validated args. No generic shell execution.

## 9. Channel range

RFCOMM channel validated as **1..63** (`security.ValidateChannel` in `internal/security/validate.go`). Note: [SECURITY.md](../../SECURITY.md) documents 1–30 — that doc is stale relative to the validator.

## 10. Scope

Default agent edits: `cmd/` + `internal/`. Packaging/docs only when requested.

---

If a task conflicts with an invariant, **stop and ask the user**.

See [safety.md](safety.md) for enforcement details.
