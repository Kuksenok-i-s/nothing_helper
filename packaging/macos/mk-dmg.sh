#!/usr/bin/env bash
# mk-dmg.sh — create a compressed DMG with .app and Applications symlink.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
APP_NAME="${APP_NAME:-tws_manager}"
VERSION="${VERSION:-0.1.0}"
OUT_DIR="${OUT_DIR:-$ROOT/dist}"
APP="$OUT_DIR/$APP_NAME.app"
DMG="$OUT_DIR/${APP_NAME}-${VERSION}-universal.dmg"
STAGING="$OUT_DIR/dmg-staging"

if [[ ! -d "$APP" ]]; then
	echo "error: app bundle not found: $APP" >&2
	exit 1
fi

rm -rf "$STAGING" "$DMG"
mkdir -p "$STAGING"
ditto "$APP" "$STAGING/$APP_NAME.app"
ln -s /Applications "$STAGING/Applications"

echo "Creating DMG..."
hdiutil create \
	-volname "$APP_NAME" \
	-srcfolder "$STAGING" \
	-ov \
	-format UDZO \
	"$DMG" >/dev/null

rm -rf "$STAGING"
echo "DMG ready: $DMG"
