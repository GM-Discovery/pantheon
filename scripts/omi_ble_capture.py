#!/usr/bin/env python3
"""Capture Omi live BLE audio into a Rumor-compatible WAV file.

This is deliberately a small host adapter. It uses BlueZ's legacy gatttool
CLI for GATT transport and libopus through ctypes, so it does not add a Python
package dependency. It handles the documented live stream protocol only;
offline-storage packets are a separate format.
"""

from __future__ import annotations

import argparse
import ctypes
import ctypes.util
import datetime as dt
import os
import re
import select
import shutil
import subprocess
import sys
import time
import wave
from pathlib import Path


OMI_SERVICE_UUID = "19b10000-e8f2-537e-4f6c-d104768a1214"
OMI_AUDIO_UUID = "19b10001-e8f2-537e-4f6c-d104768a1214"
OMI_CODEC_UUID = "19b10002-e8f2-537e-4f6c-d104768a1214"
OMI_TIME_SYNC_UUID = "19b10031-e8f2-537e-4f6c-d104768a1214"
PACKET_HEADER_BYTES = 3
SAMPLE_RATE = 16_000
CHANNELS = 1
OPUS_CODEC_ID = 0x14
OPUS_CODEC_IDS = {0x14, 0x15}

CHARACTERISTIC_RE = re.compile(
    r"handle\s*[:=]\s*(0x[0-9a-fA-F]+)\s*,\s*"
    r"char properties\s*[:=]\s*(0x[0-9a-fA-F]+)\s*,\s*"
    r"char value handle\s*[:=]\s*(0x[0-9a-fA-F]+)\s*,\s*"
    r"uuid\s*[:=]\s*([0-9a-fA-F-]+)"
)
NOTIFICATION_RE = re.compile(
    r"Notification handle\s*=\s*(0x[0-9a-fA-F]+)\s+value:\s*(.*)$"
)
HEX_RE = re.compile(r"\b[0-9a-fA-F]{2}\b")


class OmiBLEError(RuntimeError):
    pass


def require_command(name: str) -> str:
    path = shutil.which(name)
    if not path:
        raise OmiBLEError(f"required command is not installed: {name}")
    return path


def gatttool_base(adapter: str, device: str, address_type: str) -> list[str]:
    return [require_command("gatttool"), "-i", adapter, "-t", address_type, "-b", device]


def run_gatttool(base: list[str], operation: list[str], timeout: float = 15.0) -> str:
    completed = subprocess.run(
        base + operation,
        capture_output=True,
        text=True,
        timeout=timeout,
        check=False,
    )
    output = (completed.stdout or "") + (completed.stderr or "")
    if completed.returncode != 0:
        raise OmiBLEError(f"gatttool {' '.join(operation)} failed: {output.strip()}")
    return output


def parse_characteristics(output: str) -> dict[str, tuple[int, int]]:
    result: dict[str, tuple[int, int]] = {}
    for match in CHARACTERISTIC_RE.finditer(output):
        _, _, value_handle, uuid = match.groups()
        result[uuid.lower()] = (int(value_handle, 16), int(match.group(1), 16))
    return result


def discover_handles(base: list[str]) -> dict[str, tuple[int, int]]:
    output = run_gatttool(base, ["--characteristics"])
    characteristics = parse_characteristics(output)
    missing = [uuid for uuid in (OMI_AUDIO_UUID, OMI_CODEC_UUID) if uuid not in characteristics]
    if missing:
        available = ", ".join(sorted(characteristics)) or "none"
        raise OmiBLEError(f"Omi characteristics not found ({', '.join(missing)}); found: {available}")
    return characteristics


def parse_hex_bytes(value: str) -> bytes:
    return bytes(int(item, 16) for item in HEX_RE.findall(value))


def read_codec_id(base: list[str], handle: int) -> int:
    output = run_gatttool(base, ["--char-read", "-a", f"0x{handle:04x}"])
    marker = output.lower().find("characteristic value/descriptor:")
    value = output[marker:] if marker >= 0 else output
    data = parse_hex_bytes(value)
    if not data:
        raise OmiBLEError(f"codec characteristic 0x{handle:04x} returned no value: {output.strip()}")
    return data[0]


def current_epoch_le() -> str:
    return int(time.time()).to_bytes(4, "little", signed=False).hex()


class OpusDecoder:
    def __init__(self) -> None:
        library = ctypes.util.find_library("opus") or "libopus.so.0"
        try:
            self._lib = ctypes.CDLL(library)
        except OSError as exc:
            raise OmiBLEError("libopus is required to decode Omi Opus audio") from exc
        self._lib.opus_decoder_create.argtypes = [ctypes.c_int, ctypes.c_int, ctypes.POINTER(ctypes.c_int)]
        self._lib.opus_decoder_create.restype = ctypes.c_void_p
        self._lib.opus_decoder_destroy.argtypes = [ctypes.c_void_p]
        self._lib.opus_decode.argtypes = [
            ctypes.c_void_p,
            ctypes.c_void_p,
            ctypes.c_int,
            ctypes.POINTER(ctypes.c_int16),
            ctypes.c_int,
            ctypes.c_int,
        ]
        self._lib.opus_decode.restype = ctypes.c_int
        error = ctypes.c_int()
        self._decoder = self._lib.opus_decoder_create(SAMPLE_RATE, CHANNELS, ctypes.byref(error))
        if not self._decoder or error.value != 0:
            raise OmiBLEError(f"libopus decoder creation failed with code {error.value}")

    def decode(self, packet: bytes) -> bytes:
        encoded = (ctypes.c_ubyte * len(packet)).from_buffer_copy(packet)
        pcm = (ctypes.c_int16 * 5760)()
        samples = self._lib.opus_decode(
            self._decoder,
            ctypes.cast(encoded, ctypes.c_void_p),
            len(packet),
            pcm,
            5760,
            0,
        )
        if samples < 0:
            raise OmiBLEError(f"libopus rejected an Omi packet with code {samples}")
        return ctypes.string_at(pcm, samples * 2)

    def close(self) -> None:
        if self._decoder:
            self._lib.opus_decoder_destroy(self._decoder)
            self._decoder = None


class WavSink:
    def __init__(self, output: Path) -> None:
        self.output = output
        self.partial = output.with_name(output.name + ".partial")
        output.parent.mkdir(parents=True, exist_ok=True)
        try:
            os.chmod(output.parent, 0o700)
        except PermissionError:
            # Shared locations such as /tmp may intentionally reject chmod;
            # do not turn that into a false capture failure.
            pass
        self.stream = wave.open(str(self.partial), "wb")
        self.stream.setnchannels(CHANNELS)
        self.stream.setsampwidth(2)
        self.stream.setframerate(SAMPLE_RATE)
        self.frames = 0

    def write(self, pcm: bytes) -> None:
        self.stream.writeframesraw(pcm)
        self.frames += len(pcm) // 2

    def finish(self) -> Path:
        self.stream.close()
        if self.frames == 0:
            self.partial.unlink(missing_ok=True)
            raise OmiBLEError(
                "Omi connected but emitted no audio notifications; refusing to publish an empty WAV"
            )
        os.replace(self.partial, self.output)
        return self.output

    def abort(self) -> None:
        self.stream.close()
        self.partial.unlink(missing_ok=True)


def output_path(args: argparse.Namespace) -> Path:
    if args.output:
        return Path(args.output).expanduser().resolve()
    root = Path(args.output_dir).expanduser().resolve()
    stamp = dt.datetime.now(dt.timezone.utc).strftime("%Y%m%d-%H%M%SZ")
    return root / f"omi-ble-{stamp}.wav"


def parse_notification(line: str) -> tuple[int, bytes] | None:
    match = NOTIFICATION_RE.search(line)
    if not match:
        return None
    return int(match.group(1), 16), parse_hex_bytes(match.group(2))


def connect_and_capture(args: argparse.Namespace, handles: dict[str, tuple[int, int]], sink: WavSink) -> None:
    base = gatttool_base(args.adapter, args.device, args.address_type)
    audio_value, _ = handles[OMI_AUDIO_UUID]
    cccd = args.cccd_handle or audio_value + 1
    codec_id = args.codec_id
    if codec_id is None:
        codec_id = read_codec_id(base, handles[OMI_CODEC_UUID][0])
    if codec_id not in (0, *OPUS_CODEC_IDS):
        raise OmiBLEError(f"unsupported Omi codec id 0x{codec_id:02x}; use --codec-id to override")

    decoder = OpusDecoder() if codec_id in OPUS_CODEC_IDS else None
    process = subprocess.Popen(
        base + ["--interactive", "--listen"],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        text=True,
        bufsize=1,
    )
    started = time.monotonic()
    try:
        assert process.stdin is not None
        assert process.stdout is not None
        process.stdin.write("connect\n")
        process.stdin.flush()
        connected = False
        deadline = time.monotonic() + args.connect_timeout
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
                raise OmiBLEError(line.strip())
        if not connected:
            raise OmiBLEError("timed out waiting for the Omi BLE connection")

        # In interactive gatttool mode, --mtu only sets the requested value;
        # the explicit exchange is required before CV1 emits large packets.
        process.stdin.write(f"mtu {args.mtu}\n")
        process.stdin.flush()
        time.sleep(0.5)

        time_sync = handles.get(OMI_TIME_SYNC_UUID)
        if time_sync:
            # Omi's official client synchronizes the device RTC after every
            # connection.  CV1 can remain silent until this transaction has
            # completed, even though GATT connection and CCCD subscription
            # both succeed.
            process.stdin.write(f"char-write-cmd 0x{time_sync[0]:04x} {current_epoch_le()}\n")
            process.stdin.flush()

        process.stdin.write(f"char-write-req 0x{cccd:04x} 0100\n")
        process.stdin.flush()
        print(
            f"connected to {args.device}; codec=0x{codec_id:02x}; "
            f"audio_handle=0x{audio_value:04x}; cccd=0x{cccd:04x}",
            flush=True,
        )
        while True:
            if args.duration and time.monotonic() - started >= args.duration:
                break
            ready, _, _ = select.select([process.stdout], [], [], 0.25)
            if not ready:
                continue
            line = process.stdout.readline()
            if not line:
                if process.poll() is not None:
                    raise OmiBLEError("gatttool exited while receiving Omi audio")
                continue
            notification = parse_notification(line)
            if not notification:
                continue
            handle, packet = notification
            if handle != audio_value or len(packet) <= PACKET_HEADER_BYTES:
                continue
            payload = packet[PACKET_HEADER_BYTES:]
            if decoder:
                sink.write(decoder.decode(payload))
            else:
                sink.write(payload)
    finally:
        if decoder:
            decoder.close()
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


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--device", required=True, help="Omi BLE MAC address")
    parser.add_argument("--adapter", default="hci0", help="Linux Bluetooth adapter (default: hci0)")
    parser.add_argument("--address-type", choices=("public", "random"), default="public")
    parser.add_argument("--output", help="exact WAV output path")
    parser.add_argument("--output-dir", default=os.environ.get("PANTHEON_OMI_AUDIO_DIR", "/home/brick/Pantheon Audio/Omi BLE"))
    parser.add_argument("--cccd-handle", type=lambda value: int(value, 0), help="override the audio CCCD handle")
    parser.add_argument("--codec-id", type=lambda value: int(value, 0), help="override the codec characteristic value")
    parser.add_argument("--connect-timeout", type=float, default=15.0)
    parser.add_argument("--mtu", type=int, default=247, help="BLE ATT MTU to exchange (default: 247)")
    parser.add_argument("--duration", type=float, help="stop after this many seconds")
    parser.add_argument("--discover-only", action="store_true", help="discover handles and codec, then exit")
    return parser


def main() -> int:
    args = build_parser().parse_args()
    try:
        base = gatttool_base(args.adapter, args.device, args.address_type)
        handles = discover_handles(base)
        codec_id = args.codec_id if args.codec_id is not None else read_codec_id(base, handles[OMI_CODEC_UUID][0])
        print(
            f"Omi handles: audio=0x{handles[OMI_AUDIO_UUID][0]:04x}, "
            f"codec=0x{handles[OMI_CODEC_UUID][0]:04x}, codec_id=0x{codec_id:02x}",
            flush=True,
        )
        if args.discover_only:
            return 0
        sink = WavSink(output_path(args))
        try:
            connect_and_capture(args, handles, sink)
            finished = sink.finish()
            print(f"wrote {finished} ({sink.frames} samples)", flush=True)
        except BaseException:
            sink.abort()
            raise
        return 0
    except (OmiBLEError, subprocess.SubprocessError, OSError, KeyboardInterrupt) as exc:
        print(f"omi-ble-capture: {exc}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
