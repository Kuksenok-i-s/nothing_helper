#!/usr/bin/env bash
# Scan RFCOMM channels on macOS for a paired Bluetooth device.
# Requires: Go toolchain, Xcode CLI tools, paired Nothing/CMF earbuds connected.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SPIKE="$ROOT/bin/spp_spike"
SDP_PROBE="$ROOT/scripts/macos/sdp_probe.py"
PY="${ROOT}/scripts/macos/.venv/bin/python3"

MAC=""
FROM=1
TO=63
PROBE="open"   # open | battery
BUILD=0
SDP=0

usage() {
	cat <<EOF
Usage: $(basename "$0") --addr MAC [options]

Options:
  --addr MAC       Device Bluetooth address (colons or dashes)
  --from N         First RFCOMM channel (default: 1)
  --to N           Last RFCOMM channel (default: 63)
  --probe open     Only test RFCOMM open (fast, default)
  --probe battery  Full GET_BATTERY round-trip per channel
  --sdp            Run PyObjC sdp_probe first; narrow scan to resolved channel ±1
  --build          Build bin/spp_spike before scanning
  -h, --help       Show this help

Examples:
  $(basename "$0") --addr 2C:BE:EE:4A:EC:9E --build
  $(basename "$0") --addr 2C-BE-EE-4A-EC-9E --from 14 --to 16 --probe battery
EOF
}

while [[ $# -gt 0 ]]; do
	case "$1" in
		--addr)
			MAC="${2:-}"
			shift 2
			;;
		--from)
			FROM="${2:-}"
			shift 2
			;;
		--to)
			TO="${2:-}"
			shift 2
			;;
		--probe)
			PROBE="${2:-}"
			shift 2
			;;
		--build)
			BUILD=1
			shift
			;;
		--sdp)
			SDP=1
			shift
			;;
		-h|--help)
			usage
			exit 0
			;;
		*)
			echo "unknown argument: $1" >&2
			usage >&2
			exit 2
			;;
	esac
done

if [[ -z "$MAC" ]]; then
	echo "error: --addr is required" >&2
	usage >&2
	exit 2
fi

if [[ "$BUILD" -eq 1 ]] || [[ ! -x "$SPIKE" ]]; then
	echo "building $SPIKE..." >&2
	( cd "$ROOT" && go build -o bin/spp_spike ./cmd/spp_spike )
fi

if [[ "$PROBE" != "open" && "$PROBE" != "battery" ]]; then
	echo "error: --probe must be 'open' or 'battery'" >&2
	exit 2
fi

if [[ "$SDP" -eq 1 ]]; then
	if [[ ! -x "$PY" ]]; then
		echo "PyObjC venv missing. Run:" >&2
		echo "  python3 -m venv scripts/macos/.venv" >&2
		echo "  scripts/macos/.venv/bin/pip install -r scripts/macos/requirements.txt" >&2
		exit 2
	fi
	echo "SDP probe (PyObjC)..." >&2
	"$PY" "$SDP_PROBE" --addr "$MAC" || true
	ch="$("$PY" "$SDP_PROBE" --addr "$MAC" --print-channel 2>/dev/null || true)"
	if [[ -n "$ch" && "$ch" =~ ^[0-9]+$ ]]; then
		FROM=$((ch - 1))
		TO=$((ch + 1))
		[[ "$FROM" -lt 1 ]] && FROM=1
		[[ "$TO" -gt 63 ]] && TO=63
		echo "SDP resolved channel $ch; scanning $FROM..$TO" >&2
	fi
fi

echo "scanning RFCOMM channels $FROM..$TO for $MAC (probe=$PROBE, timeout=1s)" >&2
echo "channel result" | column -t

found=()
for ch in $(seq "$FROM" "$TO"); do
	args=(--addr "$MAC" --channel "$ch" --exact --timeout 1s)
	if [[ "$PROBE" == "open" ]]; then
		args+=(--open-only)
	fi
	set +e
	out="$("$SPIKE" "${args[@]}" 2>&1)"
	code=$?
	set -e
	if [[ $code -eq 0 ]]; then
		printf "%7d  OK\n" "$ch"
		found+=("$ch")
	else
		reason="$(echo "$out" | tail -1)"
		printf "%7d  FAIL  %s\n" "$ch" "$reason"
	fi
done

echo >&2
if [[ ${#found[@]} -eq 0 ]]; then
	echo "no working RFCOMM channels in range $FROM..$TO" >&2
	exit 1
fi
echo "working channels: ${found[*]}" >&2
