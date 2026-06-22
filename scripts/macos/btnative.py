"""Minimal IOBluetooth helpers for macOS hardware research.

Adapted from PyBluez macOS/_lightblue.py patterns (SDP sync query, getRFCOMMChannelID)
and PT-P300BT run-loop drain on RFCOMM close.

Not used by the Go binary — standalone spike tooling only.
"""

from __future__ import annotations

import sys
import time
from dataclasses import dataclass
from typing import Callable, Iterable, Optional

import objc
from Foundation import NSDate, NSRunLoop, NSObject

try:
    from IOBluetooth import (
        IOBluetoothDevice,
        IOBluetoothRFCOMMChannel,
        IOBluetoothSDPUUID,
    )
except ImportError as exc:  # pragma: no cover - import guard for non-macOS
    raise SystemExit(
        "IOBluetooth bindings missing. On macOS run:\n"
        "  python3 -m venv .venv && source .venv/bin/activate\n"
        "  pip install -r scripts/macos/requirements.txt"
    ) from exc

kIOReturnSuccess = 0

NOTHING_SPP_UUID = "AEAC4A03-DFF5-498F-843A-34487CF133EB"
# BluetoothSDPUUID16ServiceClassSerialPort
SERIAL_PORT_UUID16 = 0x1101

DEFAULT_OPEN_TIMEOUT = 1.0
DEFAULT_CONNECT_TIMEOUT = 1.0
DEFAULT_SDP_TIMEOUT = 5.0
DEFAULT_CLOSE_DRAIN = 0.5
DEFAULT_OPEN_RETRY_DRAIN = 1.0
OPEN_MAX_ATTEMPTS = 3


class BluetoothError(Exception):
    def __init__(self, code: int, message: str):
        super().__init__(message)
        self.code = code


def drain_runloop(seconds: float, step: float = 0.02, loop: Optional[NSRunLoop] = None) -> None:
    """Process pending run-loop sources (delegate callbacks, SDP completion)."""
    rl = loop or NSRunLoop.currentRunLoop()
    deadline = time.monotonic() + seconds
    while time.monotonic() < deadline:
        rl.runMode_beforeDate_("kCFRunLoopDefaultMode", NSDate.dateWithTimeIntervalSinceNow_(step))


def wait_until(
    predicate: Callable[[], bool],
    timeout: float,
    step: float = 0.02,
    loop: Optional[NSRunLoop] = None,
) -> bool:
    rl = loop or NSRunLoop.mainRunLoop()
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        if predicate():
            return True
        drain_runloop(step, step, rl)
    return predicate()


def normalize_mac(mac: str) -> str:
    return mac.replace("-", ":").upper()


def device_for_mac(mac: str) -> Optional[IOBluetoothDevice]:
    norm = normalize_mac(mac)
    dev = IOBluetoothDevice.deviceWithAddressString_(norm)
    if dev is not None:
        return dev
    dash = norm.replace(":", "-")
    dev = IOBluetoothDevice.deviceWithAddressString_(dash)
    if dev is not None:
        return dev
    paired = IOBluetoothDevice.pairedDevices() or []
    for candidate in paired:
        if normalize_mac(candidate.addressString()) == norm:
            return candidate
    return None


class _SDPQueryRunner(NSObject):
    """Blocks until performSDPQuery_ completes (PyBluez _SDPQueryRunner pattern)."""

    def init(self):
        self = objc.super(_SDPQueryRunner, self).init()
        if self is None:
            return None
        self._result: Optional[int] = None
        return self

    @objc.python_method
    def query(self, device: IOBluetoothDevice, timeout: float = DEFAULT_SDP_TIMEOUT) -> None:
        self._result = None
        err = device.performSDPQuery_(self)
        if err != kIOReturnSuccess:
            raise BluetoothError(err, f"performSDPQuery failed: {err}")
        if not wait_until(lambda: self._result is not None, timeout):
            raise BluetoothError(-1, f"SDP query timed out after {timeout}s")
        if self._result != kIOReturnSuccess:
            raise BluetoothError(self._result, f"SDP query failed: {self._result}")

    def sdpQueryComplete_status_(self, device, status):
        self._result = status

    sdpQueryComplete_status_ = objc.selector(
        sdpQueryComplete_status_,
        signature=b"v@:@i",
    )


def get_rfcomm_channel(record) -> Optional[int]:
    """Return RFCOMM channel from an IOBluetoothSDPServiceRecord, or None."""
    # PyObjC 10.x: signature i@:* — pass bytearray out-param (not None/tuple).
    try:
        buf = bytearray(1)
        result = record.getRFCOMMChannelID_(buf)
        if result != kIOReturnSuccess:
            return None
        return int(buf[0])
    except TypeError:
        pass
    # Older PyObjC tuple return forms.
    try:
        result, channel = record.getRFCOMMChannelID_(None)
        if result != kIOReturnSuccess:
            return None
        return int(channel)
    except TypeError:
        result, channel = record.getRFCOMMChannelID_()
        if result != kIOReturnSuccess:
            return None
        return int(channel)


def service_uuid_strings(record) -> list[str]:
    uuids: list[str] = []
    try:
        svc_uuid = record.getServiceUUID()
        if svc_uuid is not None:
            uuids.append(str(svc_uuid))
    except Exception:
        pass
    try:
        for u in record.getServiceUUIDs() or []:
            uuids.append(str(u))
    except Exception:
        pass
    return uuids


@dataclass
class ServiceInfo:
    name: str
    uuids: list[str]
    rfcomm_channel: Optional[int]
    description: str


def list_services(device: IOBluetoothDevice, sdp_timeout: float = DEFAULT_SDP_TIMEOUT) -> list[ServiceInfo]:
    services = device.services()
    if services is None or len(services) == 0:
        _SDPQueryRunner.alloc().init().query(device, timeout=sdp_timeout)
        services = device.services() or []
    out: list[ServiceInfo] = []
    for rec in services:
        name = rec.getServiceName() or ""
        uuids = service_uuid_strings(rec)
        ch = get_rfcomm_channel(rec)
        desc = str(rec.description()) if rec.description() else ""
        out.append(ServiceInfo(name=name, uuids=uuids, rfcomm_channel=ch, description=desc))
    return out


def find_service_record(device: IOBluetoothDevice, uuid_str: str):
    hexbytes = bytes.fromhex(uuid_str.replace("-", ""))
    if len(hexbytes) != 16:
        return None
    uuid = IOBluetoothSDPUUID.alloc().initWithBytes_length_(hexbytes, len(hexbytes))
    return device.getServiceRecordForUUID_(uuid)


def resolve_rfcomm_channel(
    device: IOBluetoothDevice,
    *,
    primary_uuid: str = NOTHING_SPP_UUID,
    hint: int = 15,
    sdp_timeout: float = DEFAULT_SDP_TIMEOUT,
) -> tuple[int, str]:
    """Resolve RFCOMM channel via SDP. Returns (channel, source_label)."""
    services = device.services()
    if services is None or len(services) == 0:
        _SDPQueryRunner.alloc().init().query(device, timeout=sdp_timeout)

    rec = find_service_record(device, primary_uuid)
    if rec is not None:
        ch = get_rfcomm_channel(rec)
        if ch is not None:
            return ch, f"sdp:{primary_uuid}"

    serial_uuid = IOBluetoothSDPUUID.alloc().initWithUUID16_(SERIAL_PORT_UUID16)
    rec = device.getServiceRecordForUUID_(serial_uuid)
    if rec is not None:
        ch = get_rfcomm_channel(rec)
        if ch is not None:
            return ch, "sdp:serial-port-0x1101"

    for svc in list_services(device, sdp_timeout=0):
        if svc.rfcomm_channel is None:
            continue
        blob = " ".join([svc.name, svc.description, *svc.uuids]).upper()
        if primary_uuid.replace("-", "").upper() in blob.replace("-", ""):
            return svc.rfcomm_channel, "sdp:description-scan"

    if hint >= 1:
        return hint, "hint"
    raise BluetoothError(-6, "no RFCOMM channel in SDP")


class _RFCOMMDelegate(NSObject):
    def init(self):
        self = objc.super(_RFCOMMDelegate, self).init()
        if self is None:
            return None
        self._open_status: Optional[int] = None
        self._open_done = False
        self._close_done = False
        self._read_buf = bytearray()
        return self

    @objc.python_method
    def reset_open(self):
        self._open_status = None
        self._open_done = False

    @objc.python_method
    def wait_open(self, timeout: float) -> int:
        loop = NSRunLoop.mainRunLoop()
        if wait_until(lambda: self._open_done, timeout, loop=loop):
            return self._open_status if self._open_status is not None else -1
        return -1

    @objc.python_method
    def wait_close(self, timeout: float) -> bool:
        loop = NSRunLoop.mainRunLoop()
        return wait_until(lambda: self._close_done, timeout, loop=loop)

    @objc.python_method
    def read_available(self) -> bytes:
        if not self._read_buf:
            return b""
        data = bytes(self._read_buf)
        self._read_buf.clear()
        return data

    def rfcommChannelOpenComplete_status_(self, channel, status):
        self._open_status = status
        self._open_done = True

    rfcommChannelOpenComplete_status_ = objc.selector(
        rfcommChannelOpenComplete_status_,
        signature=b"v@:@i",
    )

    def rfcommChannelClosed_(self, channel):
        self._close_done = True

    rfcommChannelClosed_ = objc.selector(
        rfcommChannelClosed_,
        signature=b"v@:@",
    )

    def rfcommChannelData_data_length_(self, channel, data_pointer, length):
        chunk = bytes(data_pointer[:length])
        self._read_buf.extend(chunk)

    rfcommChannelData_data_length_ = objc.selector(
        rfcommChannelData_data_length_,
        signature=b"v@:@^vQ",
    )


class RFCOMMTransport:
    """RFCOMM transport using openRFCOMMChannelSync (PyBluez pattern)."""

    def __init__(
        self,
        device: IOBluetoothDevice,
        channel: int,
        *,
        connect_timeout: float = DEFAULT_CONNECT_TIMEOUT,
        open_timeout: float = DEFAULT_OPEN_TIMEOUT,
        close_drain: float = DEFAULT_CLOSE_DRAIN,
    ):
        self.device = device
        self.channel_id = channel
        self.connect_timeout = connect_timeout
        self.open_timeout = open_timeout  # reserved for future async path
        self.close_drain = close_drain
        self._channel: Optional[IOBluetoothRFCOMMChannel] = None
        self._delegate = _RFCOMMDelegate.alloc().init()
        self._closed = False

    @property
    def mac(self) -> str:
        return normalize_mac(self.device.addressString())

    def connect_acl_if_needed(self) -> None:
        if self.device.isConnected():
            return
        if not self.device.openConnection():
            raise BluetoothError(-3, "openConnection failed")
        if not wait_until(lambda: self.device.isConnected(), self.connect_timeout):
            raise BluetoothError(-3, f"ACL connect timed out after {self.connect_timeout}s")

    def open(self) -> None:
        self.connect_acl_if_needed()
        loop = NSRunLoop.mainRunLoop()
        last_status = 0
        for pass_num in range(2):
            if pass_num > 0:
                if self.device.isConnected():
                    self.device.closeConnection()
                    drain_runloop(DEFAULT_OPEN_RETRY_DRAIN, loop=loop)
                self.connect_acl_if_needed()
            for attempt in range(OPEN_MAX_ATTEMPTS):
                self._delegate.reset_open()
                try:
                    last_status, self._channel = self.device.openRFCOMMChannelSync_withChannelID_delegate_(
                        None,
                        self.channel_id,
                        self._delegate,
                    )
                except TypeError:
                    last_status, self._channel = self.device.openRFCOMMChannelSync_withChannelID_delegate_(
                        self.channel_id,
                        self._delegate,
                    )
                if self._channel is not None and self._channel.isOpen():
                    self._channel.setDelegate_(self._delegate)
                    return
                if self._channel is not None:
                    self._channel.closeChannel()
                    self._channel = None
                if attempt + 1 < OPEN_MAX_ATTEMPTS:
                    drain_runloop(DEFAULT_OPEN_RETRY_DRAIN, loop=loop)
        raise BluetoothError(-5, f"openRFCOMMChannelSync failed after retry: {last_status}")

    def write(self, data: bytes) -> int:
        if self._channel is None:
            raise BluetoothError(-2, "RFCOMM channel not open")
        status = self._channel.writeSync_length_(data, len(data))
        if status != kIOReturnSuccess:
            raise BluetoothError(-3, f"writeSync failed: {status}")
        return len(data)

    def read(self, timeout: float = 0.5) -> bytes:
        loop = NSRunLoop.mainRunLoop()
        deadline = time.monotonic() + timeout
        while time.monotonic() < deadline:
            chunk = self._delegate.read_available()
            if chunk:
                return chunk
            drain_runloop(0.02, loop=loop)
        return b""

    def close(self) -> None:
        if self._closed:
            return
        self._closed = True
        if self._channel is None:
            return
        self._delegate._close_done = False
        self._channel.closeChannel()
        loop = NSRunLoop.mainRunLoop()
        if not self._delegate.wait_close(self.close_drain):
            drain_runloop(self.close_drain, loop=loop)
        self._channel = None


def ensure_darwin() -> None:
    if sys.platform != "darwin":
        raise SystemExit("scripts/macos/*.py require macOS with IOBluetooth")


def format_service_table(services: Iterable[ServiceInfo]) -> str:
    lines = ["#  ch  name                         uuids"]
    for i, svc in enumerate(services, start=1):
        ch = str(svc.rfcomm_channel) if svc.rfcomm_channel is not None else "-"
        name = (svc.name or "(unnamed)")[:28]
        uuids = ", ".join(svc.uuids) if svc.uuids else "(none)"
        lines.append(f"{i:2} {ch:>3}  {name:<28}  {uuids}")
    return "\n".join(lines)
