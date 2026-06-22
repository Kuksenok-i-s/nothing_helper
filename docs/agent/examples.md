# Code examples

Patterns copied from the codebase. Use these as templates — do not invent parallel APIs.

---

## Bootstrap (entrypoint)

TUI wires flags → runtime → services → UI:

```go
// cmd/tws_manager/main.go (simplified)
flags := app.RegisterFlags(fs, app.ProfileCLI)
cfg, err := app.ConfigFromFlags(flags)

err = app.Run(ctx, cfg, func(ctx context.Context, rt *app.Runtime) error {
    services, err := app.WireServices(ctx, rt)
    mgr := services.Manager
    if rt.Config.AutoDiscover { // TUI: off by default; Gio: on by default
        app.StartAutoConnect(ctx, mgr, rt.Config, onStatus)
    }
    return tui.Run(ctx, rt.Session, tui.Options{
        Manager: mgr, AllowUnsafe: rt.Config.AllowUnsafe, /* ... */
    })
})
```

Gio uses `app.ProfileGUI` (defaults: `--auto`, `--notify`, `--privilege-helper=auto`).

---

## Protocol (SPP)

### Marshal and decode a frame

```go
// internal/spp/frame_test.go
raw := Packet{Cmd: CmdGetBattery, FSN: 1, FixedFSN: true}.MarshalBinary()
pkt, err := DecodePacket(raw)
// pkt.Cmd == 0xC007, pkt.CRCValid == true
```

Default TX control word: `ControlTXDefault` = `0x0160` (CRC + multi-frame + TWS device type).

### Register command in catalog

```go
// internal/spp/battery.go — commandCatalog
0xF00F: {Name: "set_current_noise_reduction", Kind: "set", Safe: true},
0xF03D: {Name: "restore_factory_setting", Kind: "set", Unsafe: true}, // always blocked
```

Constants live in `internal/spp/commands.go`:

```go
CmdGetBattery        = 0xC007
CmdSetNoiseReduction = 0xF00F
CmdRspBattery        = 0x4007
```

### Register a parser

```go
// internal/spp/parse_dispatch.go — packetParsers
CmdRspFirmware: parseTextPacket("firmware/version"),

// Helper for text responses:
func parseTextPacket(kind string) PacketParser {
    return func(pkt Packet, model ModelInfo) ParsedPacket {
        return ParsedPacket{
            Kind:    kind,
            Text:    string(pkt.Payload),
            Summary: fmt.Sprintf("%s raw=%q hex=% x", kind, pkt.Payload, pkt.Payload),
        }
    }
}
```

Dispatch order in `ParsePacket`: `pkt.ParserKey()` → `pkt.Cmd` → `pkt.ResponseCmd()`.

### Parser unit test (fixture-driven)

```go
// internal/spp/spp_test.go
func TestParseBatteryRealDeviceFixture(t *testing.T) {
    pkt := Packet{Cmd: CmdRspBattery, Payload: []byte{0x02, 0x02, 0x3c, 0x03, 0x37}}
    parsed := ParsePacket(pkt, DefaultModel())
    want := "battery_response: left: 60% | right: 55% | case: n/a"
    if parsed.Summary != want {
        t.Fatalf("summary = %q, want %q", parsed.Summary, want)
    }
}
```

Full round-trip with wire bytes:

```go
payload := []byte{0x03, 0x02, 0x46, 0x03, 0x55, 0x04, 0x64}
raw := BuildFrame(ControlCRC|ControlMultiFrame, CmdBattery, 1, payload)
pkt, err := DecodePacket(raw)
parsed := ParsePacket(pkt, DefaultModel())
```

### Feature command (UI SET with validation)

```go
// internal/spp/feature_commands.go
"anc": {
    GetCommand:      CmdGetNoiseReduction,
    GetPayload:      []byte{3},
    SetCommand:      CmdSetNoiseReduction,
    SafeSet:         true,
    BuildSetPayload: BuildANCSetPayload,
},

// internal/spp/device_safety.go
func BuildANCSetPayload(model ModelInfo, args []string) ([]byte, error) {
    // returns []byte{1, mode, 0} after ValidateANCModeValue / ancOffModeValue(model)
}
```

Build packet from field tokens:

```go
pkt, warnings, err := spp.FeatureCommandPacket([]string{"anc", "set", "strong"}, allowUnsafe, model)
```

### Battery normalization (stereo → case)

```go
// internal/spp/spp_test.go — Elekid (Fast Pair 2D6FDA)
model, _ := ResolveModelInfo("2D6FDA")
data, warnings := NormalizeBatteryForModel(
    map[string]Battery{"stereo": {Percent: 88, Charging: true}}, model)
// data["case"] present, data["stereo"] removed
```

---

## Session (TX / RX / safety)

### Send GET (requires connected session in production)

```go
// internal/session/session.go
err := s.SendCommand(spp.CmdGetBattery, session.Meta{
    Source:  "ui",
    Trigger: "Info: battery",
})
```

Low-level send with payload:

```go
err := s.Send(spp.Packet{Cmd: spp.CmdGetNoiseReduction, Payload: []byte{3}}, meta)
```

### authorizeCommand (every TX)

```go
// internal/session/session.go
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

### Safety tests (copy these patterns)

```go
// internal/session/session_test.go
func TestSendBlocksUnsafe(t *testing.T) {
    s := New(nil, false, false)
    err := s.SendCommand(0xF03D, Meta{Source: "test"}) // restore_factory_setting
    if err == nil || !strings.Contains(err.Error(), "blocked") {
        t.Fatal("expected device safety block")
    }
}

func TestAuthorizeAllowsUISafeSetWithoutUnsafe(t *testing.T) {
    s := New(nil, false, false)
    for _, cmd := range []uint16{
        spp.CmdSetNoiseReduction, spp.CmdSetEQMode, spp.CmdSetLagMode,
        spp.CmdSetSpatialAudio, spp.CmdSetDualEnable, spp.CmdSetConnectDevice,
    } {
        if err := s.authorizeCommand(cmd); err != nil {
            t.Fatalf("authorize %s: %v", spp.CommandLabel(cmd), err)
        }
    }
}
```

### FSN correlation

```go
// internal/session/session_test.go
s.pending[7] = pendingTX{command: "c042", trigger: "Info: firmware"}
got := s.matchRequest(spp.Packet{Cmd: spp.CmdRspFirmware, FSN: 7})
// got == "c042"; pending[7] deleted

// Fallback when no pending entry:
got = s.matchRequest(spp.Packet{Cmd: spp.CmdRspBattery, FSN: 99})
// got == "c007"  (0x4007 → 0xC007)
```

### Connect idempotency

```go
// internal/session/session.go — same MAC while link live → no-op
if s.f != nil && device.MAC != "" && s.device.MAC == device.MAC {
    s.publish(Event{Kind: EventProgress, /* already connected */})
    return nil
}
```

### Scan (unsafe only)

```go
// internal/spp/scan.go + device_safety.go
start, end, delay, err := spp.ParseScanCommand([]string{"scan", "c001", "c020", "500ms"})
err = spp.ValidateScanRange(start, end, delay) // GET 0xC0xx only, delay >= 200ms, max 32 cmds
err = sess.RunQueryScan(ctx, start, end, delay)
```

`SafeScanCommand`: `cmd & 0xF000 == 0xC000`.

---

## UI (presenter + actions)

### Build command catalog for model

```go
// internal/ui/presenter/commands.go
cmds := BuildCommands(model, dualDevices, allowUnsafe)
// Gates features via spp.ModelSupportsFeature(model, feature)
// Raw scan appended only when allowUnsafe == true
```

Example catalog entry:

```go
{Title: "Info: battery", Desc: "GET battery levels", Cmd: spp.CmdGetBattery},
{Title: "SET: anc strong", Desc: "ANC high", Fields: []string{"anc", "set", "strong"}},
```

### Execute from UI layer

```go
// internal/ui/actions/command.go
res := actions.Execute(sess, cmd, actions.ExecOpts{Source: "ui", Comment: userComment})
if res.Err != nil { /* handle */ }

// Scan must NOT go through Execute:
res := actions.Execute(sess, scanCmd, actions.ExecOpts{})
// → error: "scan commands must use ExecuteScan"

err := actions.ExecuteScan(ctx, sess, []string{"scan", "c001", "c010", "500ms"})
```

Feature fields path inside `Execute`:

```go
pkt, warnings, err := sess.FeaturePacket(cmd.Fields)
err = sess.Send(pkt, session.Meta{Source: source, Trigger: cmd.Title, UserComment: comment})
```

---

## Security validation

```go
// internal/security/validate.go
path, err := security.ValidateRFCOMMDevice("/dev/rfcomm0")  // must be /dev/rfcommN
err = security.ValidateChannel(15)                          // range 1..63
mac, err := security.NormalizeMAC("AA:BB:CC:DD:EE:FF")
dir, err := security.ValidateWritablePath("captures")       // no ".." segments
```

---

## Connect manager

```go
// internal/connect/manager.go
mgr := connect.New(sess, connect.Options{RFCOMMPath: "/dev/rfcomm0", Channel: 15})
err := mgr.Bind(ctx, device)
err := mgr.Connect(ctx, device)           // → session.Connect
err := mgr.ConnectBest(ctx, statusFn)     // discover + bind + connect
go mgr.AutoConnect(ctx, connect.AutoOptions{OnStatus: statusFn}) // reconnect loop
```

---

## Input validation test

```go
// internal/security/validate_test.go pattern
_, err := ValidateRFCOMMDevice("/dev/ttyUSB0")
if err == nil {
    t.Fatal("expected reject non-rfcomm path")
}
```

---

## Trace / debug

```bash
go run ./cmd/tws_manager --device /dev/rfcomm0 \
  --log captures/session.ndjson --log-raw
```

NDJSON events omit `raw_hex` unless `--log-raw` (see `internal/trace/trace.go`).

Export from UI uses `actions.ExportPackets` → JSON under `--capture-dir`.
