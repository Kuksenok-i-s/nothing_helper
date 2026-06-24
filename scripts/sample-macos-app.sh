#!/usr/bin/env bash
# Sample an already-running tws_manager process with macOS 'sample' (no rebuild needed).
#
# Usage:
#   ./scripts/sample-macos-app.sh
#   ./scripts/sample-macos-app.sh tws_manager 30

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

NAME="${1:-tws_manager}"
SECONDS="${2:-20}"
OUT_DIR="${PROFILE_OUT_DIR:-captures/profiles}"
STAMP="$(date +%Y%m%d-%H%M%S)"
OUT_TXT="$OUT_DIR/sample-${NAME}-${STAMP}.txt"

mkdir -p "$OUT_DIR"

pid="$(pgrep -x "$NAME" | head -1 || true)"
if [[ -z "$pid" ]]; then
  pid="$(pgrep -f "${NAME}.app/Contents/MacOS/${NAME}" | head -1 || true)"
fi
if [[ -z "$pid" ]]; then
  echo "error: no running process named $NAME" >&2
  echo "Start tws_manager first, then rerun this script." >&2
  exit 1
fi

echo "==> sampling pid $pid for ${SECONDS}s -> $OUT_TXT"
sample "$pid" "$SECONDS" -file "$OUT_TXT"

echo "Sample saved: $OUT_TXT"
echo "Search for hot symbols:"
echo "  rg 'Call graph|tws_manager|gioui|IOBluetooth|CFRunLoop' $OUT_TXT"
