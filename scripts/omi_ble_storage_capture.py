#!/usr/bin/env python3
"""Read Omi CV1 ring-buffer audio and write a Rumor-compatible WAV.

The CV1's offline-storage packets are not the live BLE packet format.  This
adapter implements the documented firmware 3.0.20+ ring protocol using the
same dependency-light gatttool/libopus approach as omi_ble_capture.py.

The reader sends CMD_RING_ADVANCE only after the raw ring bytes and decoded
WAV have been written. It never sends CMD_RING_CLEAR. The explicit advance
keeps a completed read resumable and makes the device cursor match the local
archive rather than relying on a firmware-side checkpoint.
"""

from __future__ import annotations

import argparse
import dataclasses
import json
import os
import select
import subprocess
import sys
import time
from pathlib import Path

import omi_ble_capture


RING_CONTROL_UUID = "30295781-4301-eabd-2904-2849adfeae43"
RING_STATUS_UUID = "30295782-4301-eabd-2904-2849adfeae43"
RING_RECORD_BYTES = 444
RING_AUDIO_BYTES = 440
CMD_RING_INFO = 0x10
CMD_RING_READ = 0x11
CMD_RING_ADVANCE = 0x12
CMD_RING_CLEAR = 0x13
CMD_STOP_SYNC = 0x03
NOTIFY_ACK = 0x01
NOTIFY_INFO = 0x02
NOTIFY_DATA = 0x03
NOTIFY_DONE = 0x04
NOTIFY_READ_BEGIN = 0x05


@dataclasses.dataclass(frozen=True)
class RingStatus:
    used_bytes: int
    unread_packets: int
    free_bytes: int
    rtc_valid: int


@dataclasses.dataclass(frozen=True)
class RingInfo:
    read_seq: int
    write_seq: int
    capacity_packets: int
    dropped_packets: int
    packet_size: int


@dataclasses.dataclass(frozen=True)
class RingDone:
    status: int
    next_seq: int


def parse_value_bytes(output: str) -> bytes:
    """Parse either gatttool characteristic-value label."""
    for line in output.splitlines():
        lowered = line.lower()
        for marker in ("value/descriptor:", "value:"):
            index = lowered.find(marker)
            if index >= 0:
                return omi_ble_capture.parse_hex_bytes(line[index + len(marker) :])
    raise omi_ble_capture.OmiBLEError(f"gatttool returned no characteristic value: {output.strip()}")


def parse_status(value: bytes) -> RingStatus:
    if len(value) < 16:
        raise omi_ble_capture.OmiBLEError(f"ring status is only {len(value)} bytes, expected 16")
    return RingStatus(*(int.from_bytes(value[index : index + 4], "little") for index in range(0, 16, 4)))


def parse_info_notification(value: bytes) -> RingInfo | None:
    if len(value) < 31 or value[0] != NOTIFY_INFO:
        return None
    return RingInfo(
        read_seq=int.from_bytes(value[1:9], "big"),
        write_seq=int.from_bytes(value[9:17], "big"),
        capacity_packets=int.from_bytes(value[17:21], "big"),
        dropped_packets=int.from_bytes(value[21:29], "big"),
        packet_size=int.from_bytes(value[29:31], "big"),
    )


def parse_done_notification(value: bytes) -> RingDone | None:
    if len(value) < 10 or value[0] != NOTIFY_DONE:
        return None
    return RingDone(status=value[1], next_seq=int.from_bytes(value[2:10], "big"))


def parse_read_begin_notification(value: bytes) -> tuple[int, int] | None:
    if len(value) < 13 or value[0] != NOTIFY_READ_BEGIN:
        return None
    return int.from_bytes(value[1:9], "big"), int.from_bytes(value[9:13], "big")


def parse_audio_payload(audio: bytes) -> list[bytes]:
    """Extract packed Opus frames from one 440-byte ring record payload."""
    frames: list[bytes] = []
    offset = 0
    while offset < len(audio) - 1:
        size = audio[offset]
        if size == 0:
            offset += 1
            continue
        # Match the firmware's boundary rule: a size byte at the final
        # boundary belongs to the next block and has no frame here.
        if offset + 1 + size >= len(audio):
            break
        frames.append(audio[offset + 1 : offset + 1 + size])
        offset += size + 1
    return frames


class RingRecordReassembler:
    def __init__(self) -> None:
        self._buffer = bytearray()

    def append(self, value: bytes) -> None:
        self._buffer.extend(value)

    def drain(self) -> list[bytes]:
        records: list[bytes] = []
        while len(self._buffer) >= RING_RECORD_BYTES:
            records.append(bytes(self._buffer[:RING_RECORD_BYTES]))
            del self._buffer[:RING_RECORD_BYTES]
        return records

    @property
    def pending_bytes(self) -> int:
        return len(self._buffer)


def discover_ring_handles(base: list[str]) -> dict[str, tuple[int, int]]:
    output = omi_ble_capture.run_gatttool(base, ["--characteristics"])
    found = omi_ble_capture.parse_characteristics(output)
    missing = [uuid for uuid in (RING_CONTROL_UUID, RING_STATUS_UUID) if uuid not in found]
    if missing:
        raise omi_ble_capture.OmiBLEError(
            f"Omi ring characteristics not found ({', '.join(missing)}); "
            f"found: {', '.join(sorted(found)) or 'none'}"
        )
    return found


def read_ring_status(base: list[str], handle: int) -> RingStatus:
    output = omi_ble_capture.run_gatttool(base, ["--char-read", "-a", f"0x{handle:04x}"])
    return parse_status(parse_value_bytes(output))


def ring_command(opcode: int, *values: tuple[int, int]) -> str:
    payload = bytearray([opcode])
    for value, width in values:
        payload.extend(value.to_bytes(width, "big"))
    return payload.hex()


def output_path(args: argparse.Namespace) -> Path:
    if args.output:
        return Path(args.output).expanduser().resolve()
    root = Path(args.output_dir).expanduser().resolve()
    stamp = time.strftime("%Y%m%d-%H%M%SZ", time.gmtime())
    return root / f"omi-ring-{stamp}.wav"


def read_ring(args: argparse.Namespace, handles: dict[str, tuple[int, int]]) -> tuple[list[bytes], RingInfo, RingDone, int]:
    base = omi_ble_capture.gatttool_base(args.adapter, args.device, args.address_type)
    control_value, _ = handles[RING_CONTROL_UUID]
    control_cccd = args.cccd_handle or control_value + 1
    process = subprocess.Popen(
        base + ["--mtu", str(args.mtu), "--interactive", "--listen"],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        text=True,
        bufsize=1,
    )
    records: list[bytes] = []
    reassembler = RingRecordReassembler()
    info: RingInfo | None = None
    done: RingDone | None = None
    read_begin: tuple[int, int] | None = None

    def send(command: str) -> None:
        assert process.stdin is not None
        process.stdin.write(command + "\n")
        process.stdin.flush()

    try:
        send("connect")
        assert process.stdout is not None
        deadline = time.monotonic() + args.connect_timeout
        connected = False
        while time.monotonic() < deadline:
            ready, _, _ = select.select([process.stdout], [], [], 0.25)
            if not ready:
                continue
            line = process.stdout.readline()
            if not line:
                break
            if "Connection successful" in line:
                connected = True
                break
            if "Error:" in line or "Connection failed" in line:
                raise omi_ble_capture.OmiBLEError(line.strip())
        if not connected:
            raise omi_ble_capture.OmiBLEError("timed out waiting for the Omi BLE connection")

        # In interactive gatttool mode, --mtu only sets the requested value;
        # the explicit exchange is required before large notifications.
        send(f"mtu {args.mtu}")
        time.sleep(0.5)
        send(f"char-write-req 0x{control_cccd:04x} 0100")
        time.sleep(0.5)
        send(f"char-write-req 0x{control_value:04x} {ring_command(CMD_RING_INFO)}")
        deadline = time.monotonic() + args.info_timeout
        while time.monotonic() < deadline and info is None:
            ready, _, _ = select.select([process.stdout], [], [], 0.25)
            if not ready:
                continue
            line = process.stdout.readline()
            if not line:
                break
            notification = omi_ble_capture.parse_notification(line)
            if not notification or notification[0] != control_value:
                continue
            value = notification[1]
            info = parse_info_notification(value)
            if info is None and value[:1] == bytes([NOTIFY_ACK]):
                raise omi_ble_capture.OmiBLEError(f"Omi ring INFO rejected with status 0x{value[1]:02x}")
        if info is None:
            raise omi_ble_capture.OmiBLEError("timed out waiting for Omi ring INFO notification")
        if info.packet_size != RING_RECORD_BYTES:
            raise omi_ble_capture.OmiBLEError(
                f"unsupported Omi ring packet size {info.packet_size}; expected {RING_RECORD_BYTES}"
            )
        start_seq = info.read_seq
        if args.latest:
            start_seq = max(info.read_seq, info.write_seq - args.packet_count)
        available = info.write_seq - start_seq
        count = min(args.packet_count, available)
        if count <= 0:
            raise omi_ble_capture.OmiBLEError("Omi ring contains no unread packets")

        send(f"char-write-req 0x{control_value:04x} {ring_command(CMD_RING_READ, (start_seq, 8), (count, 4))}")
        deadline = time.monotonic() + args.read_timeout
        while time.monotonic() < deadline and done is None:
            ready, _, _ = select.select([process.stdout], [], [], 0.25)
            if not ready:
                continue
            line = process.stdout.readline()
            if not line:
                if process.poll() is not None:
                    break
                continue
            notification = omi_ble_capture.parse_notification(line)
            if not notification or notification[0] != control_value:
                continue
            value = notification[1]
            if value[:1] == bytes([NOTIFY_DATA]):
                reassembler.append(value[1:])
                records.extend(reassembler.drain())
            parsed_done = parse_done_notification(value)
            if parsed_done is not None:
                done = parsed_done
            parsed_begin = parse_read_begin_notification(value)
            if parsed_begin is not None:
                read_begin = parsed_begin
        if done is None:
            raise omi_ble_capture.OmiBLEError(
                f"timed out waiting for Omi ring DONE; records={len(records)} pending={reassembler.pending_bytes}"
            )
        if done.status != 0:
            raise omi_ble_capture.OmiBLEError(f"Omi ring read failed with status 0x{done.status:02x}")
        if len(records) < count:
            raise omi_ble_capture.OmiBLEError(f"Omi returned {len(records)} records, expected {count}")
        if done.next_seq != start_seq + count:
            raise omi_ble_capture.OmiBLEError(
                f"Omi ring boundary mismatch: requested {start_seq}+{count}, DONE reported {done.next_seq}"
            )
        if read_begin is not None and read_begin != (start_seq, count):
            raise omi_ble_capture.OmiBLEError(
                f"Omi ring READ_BEGIN mismatch: requested {(start_seq, count)}, received {read_begin}"
            )

        if getattr(args, "advance", True):
            ack_status = None
            send(f"char-write-req 0x{control_value:04x} {ring_command(CMD_RING_ADVANCE, (done.next_seq, 8))}")
            deadline = time.monotonic() + getattr(args, "advance_timeout", 8)
            while time.monotonic() < deadline and ack_status is None:
                ready, _, _ = select.select([process.stdout], [], [], 0.25)
                if not ready:
                    continue
                line = process.stdout.readline()
                if not line:
                    if process.poll() is not None:
                        break
                    continue
                notification = omi_ble_capture.parse_notification(line)
                if not notification or notification[0] != control_value:
                    continue
                value = notification[1]
                if len(value) >= 2 and value[0] == NOTIFY_ACK:
                    ack_status = value[1]
            if ack_status is None:
                raise omi_ble_capture.OmiBLEError("timed out waiting for Omi ring ADVANCE acknowledgement")
            if ack_status != 0:
                raise omi_ble_capture.OmiBLEError(f"Omi ring ADVANCE failed with status 0x{ack_status:02x}")
        return records[:count], info, done, start_seq
    finally:
        if process.stdin and process.poll() is None:
            try:
                process.stdin.write("disconnect\nquit\n")
                process.stdin.flush()
            except BrokenPipeError:
                pass
        try:
            process.wait(timeout=2)
        except subprocess.TimeoutExpired:
            process.terminate()
            process.wait(timeout=2)


def capture(args: argparse.Namespace) -> Path | None:
    base = omi_ble_capture.gatttool_base(args.adapter, args.device, args.address_type)
    handles = discover_ring_handles(base)
    status = read_ring_status(base, handles[RING_STATUS_UUID][0])
    print(
        f"Omi ring: unread={status.unread_packets} used={status.used_bytes} "
        f"free={status.free_bytes} rtc_valid={status.rtc_valid}",
        flush=True,
    )
    if status.unread_packets <= 0 and args.allow_empty:
        print("Omi ring contains no unread packets", flush=True)
        return None
    records, info, done, start_seq = read_ring(args, handles)
    output = output_path(args)
    output.parent.mkdir(parents=True, exist_ok=True)
    try:
        os.chmod(output.parent, 0o700)
    except PermissionError:
        pass
    raw_partial = output.with_name(output.name + ".ring.partial")
    raw_output = output.with_suffix(".ring.bin")
    raw_partial.write_bytes(b"".join(records))

    sink = omi_ble_capture.WavSink(output)
    decoder = omi_ble_capture.OpusDecoder()
    frame_count = 0
    try:
        for record in records:
            if len(record) != RING_RECORD_BYTES:
                raise omi_ble_capture.OmiBLEError("incomplete Omi ring record")
            for frame in parse_audio_payload(record[4:]):
                sink.write(decoder.decode(frame))
                frame_count += 1
        finished = sink.finish()
    except BaseException:
        sink.abort()
        raise
    finally:
        decoder.close()
    os.replace(raw_partial, raw_output)
    metadata = output.with_suffix(".json")
    metadata.write_text(
        json.dumps(
            {
                "device": args.device,
                "source": "omi_cv1_ring",
                "read_seq": info.read_seq,
                "write_seq": info.write_seq,
                "capture_start_seq": start_seq,
                "packet_count": len(records),
                "next_seq": done.next_seq,
                "read_begin": {
                    "start_seq": read_begin[0],
                    "packet_count": read_begin[1],
                } if read_begin is not None else None,
                "frame_count": frame_count,
                "raw_ring_path": str(raw_output),
                "wav_path": str(finished),
                "ring_advance_requested": bool(getattr(args, "advance", True)),
                "ring_advanced_to_seq": done.next_seq if getattr(args, "advance", True) else None,
                "offline_storage_quiet": None,
            },
            indent=2,
        )
        + "\n"
    )
    print(f"wrote {finished} ({frame_count} Opus frames; {len(records)} ring records)", flush=True)
    return finished


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--device", required=True)
    parser.add_argument("--adapter", default="hci0")
    parser.add_argument("--address-type", choices=("public", "random"), default="public")
    parser.add_argument("--output")
    parser.add_argument("--output-dir", default="/home/brick/Pantheon Audio/Omi BLE")
    parser.add_argument("--cccd-handle", type=lambda value: int(value, 0))
    parser.add_argument("--mtu", type=int, default=247)
    parser.add_argument("--packet-count", type=int, default=36)
    parser.add_argument("--latest", action="store_true", help="read the newest requested packets instead of the oldest unread")
    parser.add_argument("--allow-empty", action="store_true", help="treat an empty ring as a successful no-op")
    parser.add_argument("--connect-timeout", type=float, default=20)
    parser.add_argument("--info-timeout", type=float, default=12)
    parser.add_argument("--read-timeout", type=float, default=45)
    parser.add_argument("--advance-timeout", type=float, default=8)
    parser.add_argument(
        "--no-advance",
        dest="advance",
        action="store_false",
        help="leave Omi's read cursor untouched (compatibility/debug mode)",
    )
    parser.set_defaults(advance=True)
    return parser


def main() -> int:
    try:
        capture(build_parser().parse_args())
        return 0
    except (omi_ble_capture.OmiBLEError, subprocess.SubprocessError, OSError, KeyboardInterrupt) as exc:
        print(f"omi-ble-storage-capture: {exc}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
