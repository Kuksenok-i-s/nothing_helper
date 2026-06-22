# SocratiCode

This repo is indexed (`projectId: nothing_helper` in `.socraticode.json`).

## Agent rule: search before read

Do not open files to "explore". Query the index first; read only files returned by search or [hot-files.md](hot-files.md).

## Example queries (use with `codebase_search`)

| Intent | Example query |
|--------|----------------|
| Connect flow | `"Session Connect OpenRFCOMMDevice readLoop"` |
| Safety | `"authorizeCommand unsafe blocked scan ValidateScanRange"` |
| ANC SET | `"BuildANCSetPayload ancOffModeValue"` |
| UI catalog | `"BuildCommands ModelSupportsFeature allowUnsafe"` |
| Battery parse | `"ParsePacket battery NormalizeBatteryForModel stereo"` |
| Autoconnect | `"AutoConnect ConnectBest BestConnectedCandidate"` |
| Event fan-out | `"publish EventKind battery connected subscribers"` |

## Other tools

| Goal | Tool |
|------|------|
| File imports/dependents | `codebase_graph_query { filePath: "internal/session/session.go" }` |
| Blast radius before rename | `codebase_impact { target: "Connect" }` |
| Call flow from entry | `codebase_flow { entrypoint: "cmd/tws_manager/main.go" }` |
| Index stale? | `codebase_status { projectPath: "…" }` |

## When grep is better

Use ripgrep when you know the **exact** symbol or opcode:

```bash
rg "CmdGetBattery" internal/spp/
rg "0xF03D" internal/
rg "authorizeCommand" internal/session/
```

Use semantic search when the question is conceptual ("how does reconnect work?").

## Infrastructure

- Embeddings: native Ollama `nomic-embed-text` (see `.cursor/mcp.json`)
- Vectors: Docker Qdrant (`socraticode-qdrant`)

Re-index after large refactors: ask user to run `codebase_index` or do it via MCP.
