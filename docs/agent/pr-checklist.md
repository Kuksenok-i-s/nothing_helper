# PR checklist

Before finishing a task or opening a PR:

- [ ] Changes scoped to `cmd/` + `internal/` (unless user asked for packaging/docs)
- [ ] [invariants.md](invariants.md) respected — safety not weakened
- [ ] Tests added/updated next to changed behaviour
- [ ] `go test ./...` passes (or targeted packages documented)
- [ ] New parser: fixture test with real/captured bytes + meaningful `Summary` string
- [ ] UI change: updated `presenter` / `actions`; considered TUI + Gio impact
- [ ] No raw bytes in default logs
- [ ] `gofmt -w cmd internal` if Go files touched

```bash
go test ./...
gofmt -w cmd internal
```

Task-specific steps: [workflows.md](workflows.md). Code patterns: [examples.md](examples.md).
