#!/usr/bin/env bash
# build-universal.sh — build arm64 + x86_64 binaries and merge with lipo.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
BUILD_DIR="${BUILD_DIR:-$ROOT/dist/build/macos}"
CMD="${CMD:-./cmd/nothing_helper}"
TAGS="${TAGS:-gio systray}"
OUT="${OUT:-$BUILD_DIR/nothing_helper-universal}"

mkdir -p "$BUILD_DIR"

echo "Building darwin/arm64..."
(
	cd "$ROOT"
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -tags "$TAGS" \
		-o "$BUILD_DIR/nothing_helper-arm64" "$CMD"
)

echo "Building darwin/amd64..."
(
	cd "$ROOT"
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -tags "$TAGS" \
		-o "$BUILD_DIR/nothing_helper-amd64" "$CMD"
)

echo "Merging universal binary..."
lipo -create -output "$OUT" \
	"$BUILD_DIR/nothing_helper-arm64" \
	"$BUILD_DIR/nothing_helper-amd64"
lipo -info "$OUT"
echo "Universal binary: $OUT"
