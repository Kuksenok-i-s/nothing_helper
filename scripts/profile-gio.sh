#!/usr/bin/env bash
# Capture a CPU profile from the Gio GUI while it runs.
#
# Usage:
#   ./scripts/profile-gio.sh
#   PROFILE_SECONDS=30 ./scripts/profile-gio.sh --addr XX:XX:XX:XX:XX:XX
#   ./scripts/profile-gio.sh --notify=false --auto=true
#
# Output: captures/profiles/cpu-<timestamp>.prof
# Inspect: go tool pprof -http=:8080 captures/profiles/cpu-*.prof

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

PROFILE_ADDR="${PROFILE_ADDR:-127.0.0.1:6060}"
PROFILE_SECONDS="${PROFILE_SECONDS:-20}"
PROFILE_WAIT="${PROFILE_WAIT:-5}"
OUT_DIR="${PROFILE_OUT_DIR:-captures/profiles}"
BINARY="${PROFILE_BINARY:-bin/tws_manager_gio}"
TAGS="${PROFILE_TAGS:-gio}"
PID_FILE="$OUT_DIR/gio.pid"
STAMP="$(date +%Y%m%d-%H%M%S)"
OUT_PROF="$OUT_DIR/cpu-${STAMP}.prof"

mkdir -p "$OUT_DIR"

echo "==> building $BINARY (tags: $TAGS)"
go build -tags "$TAGS" -o "$BINARY" ./cmd/tws_manager_gio

if [[ -f "$PID_FILE" ]] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null; then
  echo "error: stale profile run detected (pid $(cat "$PID_FILE")); remove $PID_FILE or kill the process" >&2
  exit 1
fi

cleanup() {
  if [[ -f "$PID_FILE" ]]; then
    pid="$(cat "$PID_FILE")"
    if kill -0 "$pid" 2>/dev/null; then
      kill "$pid" 2>/dev/null || true
      for _ in $(seq 1 20); do
        kill -0 "$pid" 2>/dev/null || break
        sleep 0.25
      done
      kill -9 "$pid" 2>/dev/null || true
      wait "$pid" 2>/dev/null || true
    fi
    rm -f "$PID_FILE"
  fi
}
trap cleanup EXIT INT TERM

echo "==> starting Gio with pprof on $PROFILE_ADDR"
# Defaults minimize background work; pass extra flags after -- to override.
"$BINARY" \
  --pprof-addr="$PROFILE_ADDR" \
  --notify=false \
  --no-probe \
  --auto=false \
  "$@" &
echo $! >"$PID_FILE"

echo "==> waiting ${PROFILE_WAIT}s for startup"
sleep "$PROFILE_WAIT"

if ! kill -0 "$(cat "$PID_FILE")" 2>/dev/null; then
  echo "error: Gio exited before profiling started" >&2
  exit 1
fi

echo "==> capturing ${PROFILE_SECONDS}s CPU profile -> $OUT_PROF"
if [[ "$(uname -s)" == "Darwin" ]]; then
  OUT_SAMPLE="$OUT_DIR/sample-${STAMP}.txt"
  echo "==> macOS sample(1) in parallel -> $OUT_SAMPLE"
  sample "$(cat "$PID_FILE")" "$PROFILE_SECONDS" -file "$OUT_SAMPLE" &
  SAMPLE_PID=$!
fi
curl -sS -o "$OUT_PROF" "http://${PROFILE_ADDR}/debug/pprof/profile?seconds=${PROFILE_SECONDS}"
if [[ -n "${SAMPLE_PID:-}" ]]; then
  wait "$SAMPLE_PID" 2>/dev/null || true
fi

echo "==> top CPU consumers (Go pprof)"
go tool pprof -top "$OUT_PROF" | head -40

if [[ -f "${OUT_SAMPLE:-}" ]]; then
  echo ""
  echo "==> hot symbols (macOS sample, includes native IOBluetooth/CFRunLoop)"
  rg -n "tws_manager|gioui|IOBluetooth|CFRunLoop|bt_transport|runtime\." "$OUT_SAMPLE" | head -30 || true
fi

echo ""
echo "Profile saved: $OUT_PROF"
echo "Interactive:   go tool pprof -http=:8080 $OUT_PROF"
echo "Goroutines:    curl -s 'http://${PROFILE_ADDR}/debug/pprof/goroutine?debug=2' | head -80"
