#!/usr/bin/env bash
# bundle.sh — assemble tws_manager.app (native or prebuilt universal binary).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
APP_NAME="${APP_NAME:-tws_manager}"
VERSION="${VERSION:-0.1.0}"
OUT_DIR="${OUT_DIR:-$ROOT/dist}"
PREBUILT_BIN="${PREBUILT_BIN:-}"
BIN="$OUT_DIR/$APP_NAME.app/Contents/MacOS/$APP_NAME"
RES="$OUT_DIR/$APP_NAME.app/Contents/Resources"
ICON_SRC="$ROOT/internal/ui/tray/icon_template.png"

mkdir -p "$(dirname "$BIN")" "$RES"

if [[ -n "$PREBUILT_BIN" ]]; then
	echo "Using prebuilt binary: $PREBUILT_BIN"
	install -m 755 "$PREBUILT_BIN" "$BIN"
else
	echo "Building Gio binary for $(uname -m)..."
	(cd "$ROOT" && CGO_ENABLED=1 go build -tags "gio systray" -o "$BIN" ./cmd/tws_manager_gio)
fi

PLIST_SRC="$ROOT/packaging/macos/Info.plist.template"
PLIST_DST="$OUT_DIR/$APP_NAME.app/Contents/Info.plist"
sed -e "s/{{APP_NAME}}/$APP_NAME/g" -e "s/{{VERSION}}/$VERSION/g" "$PLIST_SRC" > "$PLIST_DST"

if [[ -f "$ICON_SRC" ]]; then
	cp "$ICON_SRC" "$RES/icon.png"
	if ! "$ROOT/packaging/macos/mk-icns.sh" "$ICON_SRC" "$RES/$APP_NAME.icns"; then
		echo "warning: could not create .icns (iconutil); tray still uses embedded PNG" >&2
	fi
fi

echo "Ad-hoc signing..."
codesign --force --deep --sign - "$OUT_DIR/$APP_NAME.app" || true

echo "Bundle ready: $OUT_DIR/$APP_NAME.app"
lipo -info "$BIN" 2>/dev/null || file "$BIN"
