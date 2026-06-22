#!/usr/bin/env bash
# mk-icns.sh — create .icns from a PNG template (macOS iconutil).
set -euo pipefail

SRC="${1:?usage: mk-icns.sh source.png output.icns}"
OUT="${2:?usage: mk-icns.sh source.png output.icns}"

ICONSET="$(mktemp -d "${TMPDIR:-/tmp}/iconset.XXXXXX")"
trap 'rm -rf "$ICONSET"' EXIT

declare -a sizes=(16 32 128 256 512)
for size in "${sizes[@]}"; do
	sips -z "$size" "$size" "$SRC" --out "$ICONSET/icon_${size}x${size}.png" >/dev/null
	double=$((size * 2))
	sips -z "$double" "$double" "$SRC" --out "$ICONSET/icon_${size}x${size}@2x.png" >/dev/null
done

iconutil -c icns -o "$OUT" "$ICONSET"
echo "Icon: $OUT"
