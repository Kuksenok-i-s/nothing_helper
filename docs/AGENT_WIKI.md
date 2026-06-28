# tws_manager — Agent Wiki

Instructions and reference for **AI agents** editing this repo.

Short Russian checklist: [AGENTS.md](../AGENTS.md).

---

## How to work in this repo

1. **Search before reading.** Use SocratiCode (`codebase_search`) or [hot-files.md](agent/hot-files.md) to locate code. Do not open files speculatively.
2. **Read invariants first.** [invariants.md](agent/invariants.md) and [safety.md](agent/safety.md) — never weaken guards without explicit user request.
3. **Stay in scope.** Default edit surface: `cmd/` + `internal/`. Do not touch packaging/docs unless asked.
4. **Match existing patterns.** Copy structure from nearby tests and handlers; run `go test ./...` before finishing.
5. **Both UIs share logic.** Changes to `presenter` or `actions` affect TUI and Gio — verify both paths mentally.
6. **No hardware assumptions.** You cannot run RFCOMM here; rely on unit tests and fixtures in `internal/spp/spp_test.go`.

When unsure about device-specific protocol behaviour (undocumented opcodes, new models), **ask the user** rather than guessing.

---

## Contents

| Topic | File |
|-------|------|
| **Agent guide** (rules, task routing) | [agent-guide.md](agent/agent-guide.md) |
| **Code examples** (patterns from real code) | [examples.md](agent/examples.md) |
| Project overview, entry points | [overview.md](agent/overview.md) |
| Layer diagram, data flow, bootstrap | [architecture.md](agent/architecture.md) |
| Package reference | [packages.md](agent/packages.md) |
| Supported devices and features | [devices.md](agent/devices.md) |
| Safety model and `--unsafe` | [safety.md](agent/safety.md) |
| CLI flags | [cli-flags.md](agent/cli-flags.md) |
| Task workflows | [workflows.md](agent/workflows.md) |
| Critical invariants | [invariants.md](agent/invariants.md) |
| Testing, build tags, packaging, CI | [development.md](agent/development.md) |
| Topic → source file map | [hot-files.md](agent/hot-files.md) |
| SocratiCode search | [socraticode.md](agent/socraticode.md) |
| PR checklist | [pr-checklist.md](agent/pr-checklist.md) |
| macOS port research (feasibility, roadmap) | [macos-adaptation.md](agent/macos-adaptation.md) |

---

## Suggested reading order

1. [agent-guide.md](agent/agent-guide.md) — how to route tasks
2. [examples.md](agent/examples.md) — copy these patterns
3. [architecture.md](agent/architecture.md) + [packages.md](agent/packages.md)
4. [invariants.md](agent/invariants.md) + [safety.md](agent/safety.md)
5. [workflows.md](agent/workflows.md) for your specific task

---

## Quick task routing

| User asks for… | Start here |
|----------------|------------|
| New SPP command / parser | [examples.md § Protocol](agent/examples.md#protocol-spp) → `internal/spp/` |
| UI command or toggle | [examples.md § UI](agent/examples.md#ui-presenter--actions) → `presenter/commands.go` |
| Connect / RFCOMM bug | `internal/bt/bt.go`, `internal/session/session.go` Connect |
| Safety / `--unsafe` | `device_safety.go`, `session.authorizeCommand` |
| New model support | `internal/spp/models.go` |
| Tests only | mirror existing `*_test.go` next to changed package |
