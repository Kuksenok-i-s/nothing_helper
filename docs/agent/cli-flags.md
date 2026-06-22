# CLI flags

| Flag | Default (TUI / Gio) | Meaning |
|------|---------------------|---------|
| `--device` | `/dev/rfcomm0` | RFCOMM device path |
| `--addr` | — | Skip discovery; bind/open this MAC |
| `--channel` | `15` | RFCOMM channel when binding (validated **1..63** in `security.ValidateChannel`; [SECURITY.md](../../SECURITY.md) still says 1–30 — code wins) |
| `--model` | — | Force model codename / product / Fast Pair ID |
| `--log` | auto in `captures/` | NDJSON trace path |
| `--log-raw` | false | Include raw bytes in trace/export |
| `--capture-dir` | `captures` | JSON export directory |
| `--no-probe` | false | Skip post-connect probes |
| `--query-every` | 0 (60s if `--notify`) | Periodic GET_BATTERY |
| `--unsafe` | false | Allow raw scan + non-UI SET |
| `--auto` | false / **true** | Auto-discover and connect |
| `--notify` | false / **true** | Desktop notifications |
| `--privilege-helper` | sudo / **auto** | sudo \| polkit \| auto \| none |
| `--privilege-helper-path` | — | Custom polkit helper path |
| `--pc-primary` | ask | Dual PC-primary policy |
