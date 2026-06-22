# Architecture

## Layer diagram

```mermaid
flowchart TB
  subgraph entry [Entrypoints]
    TUI[cmd/tws_manager]
    GIO[cmd/tws_manager_gio]
    HELPER[cmd/tws_manager_rfcomm_helper]
  end

  subgraph app [Bootstrap]
    APP[internal/app]
  end

  subgraph ui [UI]
    TUIUI[internal/ui/tui]
    GIOUI[internal/ui/gio]
    TRAY[internal/ui/tray]
    PRES[internal/ui/presenter]
    ACT[internal/ui/actions]
  end

  subgraph core [Core]
    CONN[internal/connect]
    SESS[internal/session]
    SPP[internal/spp]
    BT[internal/bt]
    TRACE[internal/trace]
    SEC[internal/security]
  end

  subgraph side [Side services]
    NOTIFY[internal/notify]
    AUDIO[internal/audio]
    DUAL[internal/dualpolicy]
  end

  TUI --> APP
  GIO --> APP
  APP --> SESS
  APP --> CONN
  APP --> NOTIFY
  TUI --> TUIUI
  GIO --> GIOUI
  TUI --> TRAY
  GIO --> TRAY
  TUIUI --> CONN
  TUIUI --> PRES
  TUIUI --> ACT
  GIOUI --> CONN
  GIOUI --> PRES
  GIOUI --> ACT
  ACT --> SESS
  CONN --> SESS
  CONN --> BT
  SESS --> SPP
  SESS --> BT
  SESS --> TRACE
  BT --> SEC
  SESS --> SEC
  HELPER -.->|polkit pkexec| BT
```

## Runtime data flow

```mermaid
sequenceDiagram
  participant UI as UI / connect.Manager
  participant S as session.Session
  participant BT as internal/bt
  participant RFCOMM as /dev/rfcommN
  participant SPP as internal/spp
  participant T as trace.Logger

  UI->>S: Connect(device, path, channel)
  S->>BT: OpenRFCOMMDevice (bind/revive/open)
  BT->>RFCOMM: rfcomm bind / open
  S->>SPP: ResetFSN()
  S-->>S: go readLoop()
  S-->>S: go initialProbe() [if probe enabled]

  Note over S: initialProbe: C001 → C042 → C005 → C007 → config GETs

  UI->>S: Send / SendCommand
  S->>S: authorizeCommand()
  S->>SPP: MarshalBinary()
  S->>RFCOMM: write
  S->>T: Log TX, pending[FSN]=request

  loop readLoop
    RFCOMM->>SPP: ReadPacket
    SPP->>SPP: DecodePacket → ParsePacket
    S->>S: matchRequest(FSN) or 0x40xx↔0xC0xx
    S->>T: Log RX
    S->>S: publish(Event) → Events channel + subscribers
  end
```

## Bootstrap lifecycle (`internal/app`)

1. **`RegisterFlags` / `ConfigFromFlags`** — TUI: `--auto=false`, `--notify=false`, `--privilege-helper=sudo`. Gio: `--auto=true`, `--notify=true`, `--privilege-helper=auto`.
2. **`Bootstrap`** — `trace.Logger` + `session.Session` + optional battery polling.
3. **`WireServices`** — privileges, `connect.Manager`, optional `notify.Run`.
4. **`Run`** — user callback; shutdown on ctx cancel.

**Agent tip:** new global behaviour usually starts in `flags.go` → `config.go` → `bootstrap.go` or `entry.go`.

Key files: `bootstrap.go`, `entry.go`, `shutdown.go`, `flags.go`, `config.go`.
