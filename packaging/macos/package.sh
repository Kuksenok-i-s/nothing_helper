#!/usr/bin/env bash
# package.sh — universal macOS .app bundle + signed DMG installer.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
APP_NAME="${APP_NAME:-tws_manager}"
VERSION="${VERSION:-0.1.0}"
OUT_DIR="${OUT_DIR:-$ROOT/dist}"

export APP_NAME VERSION OUT_DIR

echo "==> Universal binary (arm64 + x86_64)"
"$ROOT/packaging/macos/build-universal.sh"

echo "==> App bundle"
PREBUILT_BIN="$ROOT/dist/build/macos/tws_manager-universal" \
	"$ROOT/packaging/macos/bundle.sh"

echo "==> DMG"
"$ROOT/packaging/macos/mk-dmg.sh"

echo ""
echo "Install: open $OUT_DIR/${APP_NAME}-${VERSION}-universal.dmg"
echo "Then drag $APP_NAME.app to Applications."
