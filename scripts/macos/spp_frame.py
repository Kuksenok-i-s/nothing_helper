"""Minimal SPP frame helpers mirroring internal/spp/frame.go for Python spikes."""

from __future__ import annotations

import struct
from dataclasses import dataclass
from typing import Optional

SOF = 0x55
CONTROL_CRC = 0x20
CONTROL_MULTI_FRAME = 0x40
DEVICE_TYPE_TWS = 1 << 8
CONTROL_TX_DEFAULT = CONTROL_CRC | CONTROL_MULTI_FRAME | DEVICE_TYPE_TWS

CMD_GET_BATTERY = 0xC007
CMD_RSP_BATTERY = 0x4007


def crc16(data: bytes) -> int:
    crc = 0xFFFF
    for b in data:
        crc ^= b
        for _ in range(8):
            if crc & 1:
                crc = (crc >> 1) ^ 0xA001
            else:
                crc >>= 1
    return crc & 0xFFFF


def marshal_packet(cmd: int, payload: bytes = b"", *, fsn: int = 1, control: int = CONTROL_TX_DEFAULT) -> bytes:
    body_len = 8 + len(payload) + (2 if control & CONTROL_CRC else 0)
    buf = bytearray(body_len)
    buf[0] = SOF
    struct.pack_into("<H", buf, 1, control)
    struct.pack_into("<H", buf, 3, cmd)
    struct.pack_into("<H", buf, 5, len(payload))
    buf[7] = fsn & 0xFF
    if payload:
        buf[8 : 8 + len(payload)] = payload
    if control & CONTROL_CRC:
        c = crc16(bytes(buf[: 8 + len(payload)]))
        struct.pack_into("<H", buf, 8 + len(payload), c)
    return bytes(buf)


@dataclass
class Packet:
    control: int
    cmd: int
    length: int
    fsn: int
    payload: bytes
    crc: int
    crc_valid: bool
    raw: bytes


def decode_packet(buf: bytes) -> Packet:
    if len(buf) < 10:
        raise ValueError("short frame")
    if buf[0] != SOF:
        raise ValueError("bad SOF")
    control, cmd, length, fsn = struct.unpack_from("<HHHB", buf, 1)
    end = 8 + length
    if len(buf) < end:
        raise ValueError("waiting for payload")
    payload = bytes(buf[8:end])
    crc = 0
    crc_valid = True
    if control & CONTROL_CRC:
        if len(buf) < end + 2:
            raise ValueError("waiting for crc")
        crc = struct.unpack_from("<H", buf, end)[0]
        crc_valid = crc16(buf[:end]) == crc
    return Packet(
        control=control,
        cmd=cmd,
        length=length,
        fsn=fsn,
        payload=payload,
        crc=crc,
        crc_valid=crc_valid,
        raw=bytes(buf[: end + (2 if control & CONTROL_CRC else 0)]),
    )


def find_frame(buf: bytes) -> tuple[Optional[Packet], bytes]:
    """Return decoded packet and remaining buffer, or (None, buf) if incomplete."""
    idx = buf.find(bytes([SOF]))
    if idx < 0:
        return None, buf
    if idx > 0:
        buf = buf[idx:]
    if len(buf) < 8:
        return None, buf
    control = struct.unpack_from("<H", buf, 1)[0]
    length = struct.unpack_from("<H", buf, 5)[0]
    need = 8 + length + (2 if control & CONTROL_CRC else 0)
    if len(buf) < need:
        return None, buf
    try:
        pkt = decode_packet(buf[:need])
    except ValueError:
        return None, buf[1:]
    return pkt, buf[need:]


def battery_summary(payload: bytes) -> str:
    if len(payload) < 3:
        return f"battery payload ({len(payload)} bytes): {payload.hex()}"
    left, right, case = payload[0], payload[1], payload[2]
    return f"L={left}% R={right}% case={case}%"
