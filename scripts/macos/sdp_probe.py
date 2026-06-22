#!/usr/bin/env python3
"""SDP research probe for Nothing/CMF SPP on macOS.

Lists paired-device SDP records, resolves RFCOMM channel via getRFCOMMChannelID,
and optionally shells out to the Go spp_spike binary.

Examples:
  python3 scripts/macos/sdp_probe.py --addr 2C:BE:EE:4A:EC:9E
  python3 scripts/macos/sdp_probe.py --addr 2C-BE-EE-4A-EC-9E --print-channel
  python3 scripts/macos/sdp_probe.py --addr AA:BB:CC:DD:EE:FF --spike
"""

from __future__ import annotations

import argparse
import subprocess
import sys
from pathlib import Path

SCRIPT_DIR = Path(__file__).resolve().parent
if str(SCRIPT_DIR) not in sys.path:
    sys.path.insert(0, str(SCRIPT_DIR))

from btnative import (  # noqa: E402
    BluetoothError,
    NOTHING_SPP_UUID,
    device_for_mac,
    ensure_darwin,
    format_service_table,
    list_services,
    normalize_mac,
    resolve_rfcomm_channel,
)


def repo_root() -> Path:
    return SCRIPT_DIR.parent.parent


def parse_args() -> argparse.Namespace:
    p = argparse.ArgumentParser(description="macOS SDP / RFCOMM channel research probe")
    p.add_argument("--addr", help="Device Bluetooth MAC (colons or dashes)")
    p.add_argument("--hint", type=int, default=15, help="Fallback channel if SDP fails (default: 15)")
    p.add_argument("--sdp-timeout", type=float, default=5.0, help="SDP query timeout seconds")
    p.add_argument("--print-channel", action="store_true", help="Print only resolved channel number")
    p.add_argument("--spike", action="store_true", help="Run Go bin/spp_spike after SDP resolve")
    p.add_argument("--open-only", action="store_true", help="Pass --open-only to spp_spike")
    p.add_argument("--list-paired", action="store_true", help="List all paired devices and exit")
    return p.parse_args()


def list_paired() -> int:
    from IOBluetooth import IOBluetoothDevice

    paired = IOBluetoothDevice.pairedDevices() or []
    if not paired:
        print("No paired devices.")
        return 0
    print(f"{'MAC':<20} {'connected':<10} {'name'}")
    for dev in paired:
        mac = normalize_mac(dev.addressString())
        name = dev.name() or dev.nameOrAddress() or ""
        print(f"{mac:<20} {str(dev.isConnected()):<10} {name}")
    return 0


def run_spike(mac: str, channel: int, open_only: bool) -> int:
    spike = repo_root() / "bin" / "spp_spike"
    if not spike.exists():
        print(f"Go spike binary missing: {spike}", file=sys.stderr)
        print("Build with: go build -o bin/spp_spike ./cmd/spp_spike", file=sys.stderr)
        return 1
    cmd = [str(spike), "--addr", mac, "--channel", str(channel), "--exact"]
    if open_only:
        cmd.append("--open-only")
    print(f"$ {' '.join(cmd)}", file=sys.stderr)
    return subprocess.call(cmd)


def main() -> int:
    ensure_darwin()
    args = parse_args()

    if args.list_paired:
        return list_paired()

    if not args.addr:
        print("error: --addr is required unless --list-paired", file=sys.stderr)
        return 2

    mac = normalize_mac(args.addr)
    dev = device_for_mac(mac)
    if dev is None:
        print(f"device not found: {mac}", file=sys.stderr)
        return 2

    if not args.print_channel:
        name = dev.name() or dev.nameOrAddress() or mac
        print(f"device: {name} ({mac})")
        print(f"paired={dev.isPaired()} connected={dev.isConnected()}")
        print(f"Nothing SPP UUID: {NOTHING_SPP_UUID}")
        print()

    try:
        services = list_services(dev, sdp_timeout=args.sdp_timeout)
    except BluetoothError as exc:
        print(f"SDP query failed: {exc}", file=sys.stderr)
        return 3

    if not args.print_channel:
        print(f"SDP services ({len(services)}):")
        print(format_service_table(services))
        print()

    try:
        channel, source = resolve_rfcomm_channel(dev, hint=args.hint, sdp_timeout=0)
    except BluetoothError as exc:
        print(f"channel resolve failed: {exc}", file=sys.stderr)
        return 4

    if args.print_channel:
        print(channel)
    else:
        print(f"resolved channel: {channel} ({source})")

    if args.spike:
        return run_spike(mac, channel, args.open_only)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
