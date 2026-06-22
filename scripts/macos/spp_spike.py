#!/usr/bin/env python3
"""Pure-PyObjC SPP hardware spike: SDP resolve → RFCOMM open → GET_BATTERY.

Faster iteration than rebuilding the CGO bridge. Uses the same wire format as
internal/spp (see spp_frame.py).

Examples:
  python3 scripts/macos/spp_spike.py --addr 2C:BE:EE:4A:EC:9E
  python3 scripts/macos/spp_spike.py --addr 2C:BE:EE:4A:EC:9E --channel 15 --exact
  python3 scripts/macos/spp_spike.py --addr 2C:BE:EE:4A:EC:9E --open-only
"""

from __future__ import annotations

import argparse
import sys
import time
from pathlib import Path

SCRIPT_DIR = Path(__file__).resolve().parent
if str(SCRIPT_DIR) not in sys.path:
    sys.path.insert(0, str(SCRIPT_DIR))

from btnative import (  # noqa: E402
    BluetoothError,
    RFCOMMTransport,
    device_for_mac,
    ensure_darwin,
    normalize_mac,
    resolve_rfcomm_channel,
)
from spp_frame import (  # noqa: E402
    CMD_GET_BATTERY,
    CMD_RSP_BATTERY,
    battery_summary,
    find_frame,
    marshal_packet,
)


def parse_args() -> argparse.Namespace:
    p = argparse.ArgumentParser(description="PyObjC SPP GET_BATTERY spike")
    p.add_argument("--addr", required=True, help="Device Bluetooth MAC")
    p.add_argument("--channel", type=int, default=0, help="RFCOMM channel (0 = SDP resolve)")
    p.add_argument("--exact", action="store_true", help="Do not SDP-resolve when --channel is set")
    p.add_argument("--hint", type=int, default=15, help="SDP fallback channel (default: 15)")
    p.add_argument("--timeout", type=float, default=1.0, help="Read timeout seconds")
    p.add_argument("--open-only", action="store_true", help="Open RFCOMM and exit")
    p.add_argument("--open-timeout", type=float, default=1.0)
    p.add_argument("--connect-timeout", type=float, default=1.0)
    return p.parse_args()


def main() -> int:
    ensure_darwin()
    args = parse_args()
    mac = normalize_mac(args.addr)

    dev = device_for_mac(mac)
    if dev is None:
        print(f"device not found: {mac}", file=sys.stderr)
        return 2

    channel = args.channel
    source = "cli"
    if channel <= 0 or not args.exact:
        try:
            channel, source = resolve_rfcomm_channel(dev, hint=args.hint)
        except BluetoothError as exc:
            print(f"SDP resolve failed: {exc}", file=sys.stderr)
            if channel <= 0:
                return 3
    if channel < 1 or channel > 63:
        print(f"invalid channel: {channel}", file=sys.stderr)
        return 2

    print(f"opening RFCOMM {mac} channel {channel} ({source})", file=sys.stderr)
    transport = RFCOMMTransport(
        dev,
        channel,
        connect_timeout=args.connect_timeout,
        open_timeout=args.open_timeout,
    )
    try:
        transport.open()
    except BluetoothError as exc:
        print(f"open failed: {exc}", file=sys.stderr)
        return 4

    try:
        print(f"opened channel {channel}", file=sys.stderr)
        if args.open_only:
            print("open PASS", file=sys.stderr)
            return 0

        tx = marshal_packet(CMD_GET_BATTERY, fsn=1)
        print(f"TX GET_BATTERY ({len(tx)} bytes): {tx.hex()}", file=sys.stderr)
        transport.write(tx)

        deadline = time.monotonic() + args.timeout
        buf = bytearray()
        while time.monotonic() < deadline:
            buf.extend(transport.read(timeout=0.2))
            pkt, rest = find_frame(bytes(buf))
            if pkt is None:
                buf = bytearray(rest) if rest else buf
                continue
            print(f"RX cmd=0x{pkt.cmd:04x} crc_ok={pkt.crc_valid}")
            if pkt.cmd == CMD_RSP_BATTERY:
                print(battery_summary(pkt.payload))
                print("spike PASS", file=sys.stderr)
                return 0
            buf = bytearray(rest)

        print("timeout waiting for battery response", file=sys.stderr)
        if buf:
            print(f"partial RX ({len(buf)} bytes): {bytes(buf).hex()}", file=sys.stderr)
        return 5
    finally:
        transport.close()


if __name__ == "__main__":
    raise SystemExit(main())
