#!/usr/bin/env python3
"""Drain Omi CV1 stored audio into the existing local Rumor intake path.

Each invocation owns one bounded BLE session. A completed batch retains the
WAV, raw ring bytes, and capture metadata produced by the storage reader; this
wrapper adds one sync receipt describing the whole attempt. It never starts
Rumor/Apollo ingestion, and it disconnects the device it connected when the
drain ends so Omi does not remain in its offline-recording rest state.
"""

from __future__ import annotations

import argparse
import datetime as dt
import json
import os
import subprocess
import time
import wave
from pathlib import Path

import omi_ble_storage_capture as storage


def unique_output(root: Path) -> Path:
    stamp = dt.datetime.now(dt.timezone.utc).strftime("%Y%m%d-%H%M%S")
    return root / f"omi-ring-{stamp}-{time.time_ns() % 1_000_000_000:09d}Z.wav"


def unique_cutting_output(root: Path, first: Path, last: Path) -> Path:
    first_stamp = first.stem.removeprefix("omi-ring-")
    last_stamp = last.stem.removeprefix("omi-ring-")
    return root / f"omi-cutting-{first_stamp}--{last_stamp}-{time.time_ns() % 1_000_000_000:09d}Z.wav"


def atomic_json(path: Path, value: object) -> None:
    partial = path.with_name(path.name + ".partial")
    partial.write_text(json.dumps(value, indent=2) + "\n")
    os.replace(partial, path)


def build_capture_args(args: argparse.Namespace, output: Path) -> argparse.Namespace:
    return argparse.Namespace(
        device=args.device,
        adapter=args.adapter,
        address_type=args.address_type,
        output=str(output),
        output_dir=str(args.output_dir),
        cccd_handle=None,
        mtu=args.mtu,
        packet_count=args.packet_count,
        latest=False,
        connect_timeout=args.connect_timeout,
        info_timeout=args.info_timeout,
        read_timeout=args.read_timeout,
        advance_timeout=getattr(args, "advance_timeout", 8),
        advance=getattr(args, "advance", True),
        allow_empty=True,
    )


def ensure_device_connected(args: argparse.Namespace) -> bool:
    """Connect once and report whether this sync invocation owns that link."""
    timeout = max(5.0, min(float(args.connect_timeout), 60.0))
    try:
        info = subprocess.run(
            ["bluetoothctl", "info", args.device],
            capture_output=True,
            text=True,
            timeout=timeout,
            check=False,
        )
    except (FileNotFoundError, subprocess.TimeoutExpired) as exc:
        raise RuntimeError(f"BlueZ preflight unavailable: {exc}") from exc
    info_output = info.stdout + info.stderr
    if "Connected: yes" in info_output:
        return False
    try:
        result = subprocess.run(
            ["bluetoothctl", "connect", args.device],
            capture_output=True,
            text=True,
            timeout=timeout,
            check=False,
        )
    except subprocess.TimeoutExpired as exc:
        raise RuntimeError(f"BlueZ reconnect timed out after {timeout:g}s") from exc
    output = (result.stdout + result.stderr).strip()
    if result.returncode != 0 or "Connection successful" not in output:
        raise RuntimeError(f"BlueZ reconnect failed for {args.device}: {output or 'no diagnostic output'}")
    return True


def disconnect_device(args: argparse.Namespace) -> None:
    """Release the BLE link without powering off the user's pendant."""
    timeout = max(5.0, min(float(args.connect_timeout), 60.0))
    try:
        result = subprocess.run(
            ["bluetoothctl", "disconnect", args.device],
            capture_output=True,
            text=True,
            timeout=timeout,
            check=False,
        )
    except (FileNotFoundError, subprocess.TimeoutExpired) as exc:
        raise RuntimeError(f"BlueZ disconnect unavailable: {exc}") from exc
    output = (result.stdout + result.stderr).strip()
    if result.returncode != 0 and "successful" not in output.lower():
        raise RuntimeError(f"BlueZ disconnect failed for {args.device}: {output or 'no diagnostic output'}")


def capture_batch(args: argparse.Namespace, output: Path):
    """Prefer native BlueZ D-Bus; retain gatttool only as a compatibility fallback."""
    try:
        import omi_dbus_storage_capture as native_storage
    except (ImportError, SystemExit):
        native_storage = None
    capture_args = build_capture_args(args, output)
    if native_storage is not None:
        capture_args.info_timeout = args.info_timeout
        capture_args.read_timeout = args.read_timeout
        return native_storage.capture(capture_args), "bluez_dbus"
    return storage.capture(capture_args), "gatttool"


def assemble_cuttings(raw_outputs: list[dict[str, str]], root: Path, seconds: int) -> list[dict[str, object]]:
    """Combine short raw batches into longer Rumor-visible WAV cuttings."""
    if not raw_outputs:
        return []
    if seconds <= 0:
        raise ValueError("cutting duration must be positive")

    result: list[dict[str, object]] = []
    group: list[dict[str, str]] = []
    group_frames = 0
    sample_rate = 0
    target_frames = 0

    def flush() -> None:
        nonlocal group, group_frames, sample_rate, target_frames
        if not group:
            return
        first = Path(group[0]["wav_path"])
        last = Path(group[-1]["wav_path"])
        output = unique_cutting_output(root, first, last)
        partial = output.with_name(output.name + ".partial")
        with wave.open(str(first), "rb") as source:
            params = source.getparams()
            if params.nchannels != 1 or params.sampwidth != 2 or params.comptype != "NONE":
                raise ValueError(f"unsupported Omi WAV format in {first}")
            with wave.open(str(partial), "wb") as sink:
                sink.setparams(params)
                for item in group:
                    with wave.open(item["wav_path"], "rb") as batch:
                        batch_params = batch.getparams()
                        if (batch_params.nchannels, batch_params.sampwidth, batch_params.framerate, batch_params.comptype) != (params.nchannels, params.sampwidth, params.framerate, params.comptype):
                            raise ValueError(f"inconsistent Omi WAV format in {item['wav_path']}")
                        sink.writeframes(batch.readframes(batch.getnframes()))
        os.replace(partial, output)
        metadata = output.with_suffix(".json")
        atomic_json(
            metadata,
            {
                "kind": "omi-audio-cutting",
                "source": "omi_cv1_ring",
                "transport": "bluez_dbus_or_gatttool",
                "duration_seconds": round(group_frames / sample_rate, 3),
                "raw_batch_count": len(group),
                "raw_batches": group,
                "wav_path": str(output),
            },
        )
        result.append({"wav_path": str(output), "metadata_path": str(metadata), "raw_batch_count": len(group)})
        group = []
        group_frames = 0
        sample_rate = 0
        target_frames = 0

    for item in raw_outputs:
        path = Path(item["wav_path"])
        with wave.open(str(path), "rb") as batch:
            frames = batch.getnframes()
            rate = batch.getframerate()
            params = batch.getparams()
            if params.nchannels != 1 or params.sampwidth != 2 or params.comptype != "NONE":
                raise ValueError(f"unsupported Omi WAV format in {path}")
            if group and rate != sample_rate:
                flush()
            if not group:
                sample_rate = rate
                target_frames = rate * seconds
            group.append(item)
            group_frames += frames
        if group_frames >= target_frames:
            flush()
    flush()
    return result


def sync(args: argparse.Namespace) -> int:
    root = args.output_dir.expanduser().resolve()
    root.mkdir(parents=True, exist_ok=True)
    raw_root = root / ".raw"
    raw_root.mkdir(parents=True, exist_ok=True)
    started = dt.datetime.now(dt.timezone.utc)
    raw_outputs: list[dict[str, str]] = []
    error = ""
    bounded = False
    owned_connection = False
    disconnected_after = False
    cutting_seconds = int(getattr(args, "cutting_seconds", 60))
    try:
        import omi_dbus_storage_capture  # noqa: F401
        backend = "bluez_dbus"
    except (ImportError, SystemExit):
        backend = "gatttool"
    if not error and backend == "gatttool" and not getattr(args, "allow_reconnect_fallback", False):
        error = "native BlueZ D-Bus backend unavailable; refusing reconnect-per-batch fallback"
    if not error and backend == "bluez_dbus":
        try:
            owned_connection = ensure_device_connected(args) is True
        except Exception as exc:  # the receipt is the durable failure surface
            error = f"connection preflight: {exc}"
    try:
        if not error:
            for batch in range(1, args.max_batches + 1):
                output = unique_output(raw_root)
                try:
                    finished, backend = capture_batch(args, output)
                except Exception as exc:  # the receipt is the durable failure surface
                    error = f"batch {batch}: {exc}"
                    break
                if finished is None:
                    break
                metadata = finished.with_suffix(".json")
                raw_outputs.append({"wav_path": str(finished), "metadata_path": str(metadata)})
            else:
                bounded = True
    finally:
        if owned_connection:
            try:
                disconnect_device(args)
                disconnected_after = True
            except Exception as exc:
                if not error:
                    error = f"post-extraction disconnect: {exc}"

    outputs: list[dict[str, object]] = []
    if raw_outputs:
        try:
            outputs = assemble_cuttings(raw_outputs, root, cutting_seconds)
        except Exception as exc:
            error = f"assemble cuttings: {exc}"

    quiet_checks: list[bool] = []
    for item in raw_outputs:
        try:
            value = json.loads(Path(item["metadata_path"]).read_text()).get("offline_storage_quiet")
        except (OSError, ValueError):
            value = None
        if isinstance(value, bool):
            quiet_checks.append(value)

    completed = dt.datetime.now(dt.timezone.utc)
    receipt = {
        "kind": "omi-storage-sync",
        "device": args.device,
        "adapter": args.adapter,
        "address_type": args.address_type,
        "started_at": started.isoformat(),
        "completed_at": completed.isoformat(),
        "batch_count": len(raw_outputs),
        "cutting_count": len(outputs),
        "max_batches": args.max_batches,
        "cutting_seconds": cutting_seconds,
        "packet_count": args.packet_count,
        "transport": backend,
        "connection_owned": owned_connection,
        "disconnected_after": disconnected_after,
        "offline_storage_quiet": all(quiet_checks) if quiet_checks else None,
        "outputs": outputs,
        "raw_outputs": raw_outputs,
        "bounded": bounded,
        "state": "failed" if error else ("completed" if raw_outputs else "empty"),
    }
    if error:
        receipt["error"] = error
    receipt_path = root / f"omi-sync-{completed.strftime('%Y%m%d-%H%M%S')}-{time.time_ns() % 1_000_000_000:09d}Z.json"
    atomic_json(receipt_path, receipt)
    print(json.dumps(receipt, indent=2), flush=True)
    return 1 if error else 0


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--device", required=True)
    parser.add_argument("--adapter", default="hci0")
    parser.add_argument("--address-type", choices=("public", "random"), default="random")
    parser.add_argument("--output-dir", type=Path, default=Path("/home/brick/Pantheon Audio/Omi BLE"))
    parser.add_argument("--packet-count", type=int, default=36)
    parser.add_argument("--max-batches", type=int, default=512)
    parser.add_argument("--cutting-seconds", type=int, default=60)
    parser.add_argument("--mtu", type=int, default=247)
    parser.add_argument("--connect-timeout", type=float, default=20)
    parser.add_argument("--info-timeout", type=float, default=12)
    parser.add_argument("--read-timeout", type=float, default=45)
    parser.add_argument("--advance-timeout", type=float, default=8)
    parser.add_argument(
        "--allow-reconnect-fallback",
        action="store_true",
        help="allow the legacy gatttool backend, which reconnects between batches",
    )
    parser.add_argument(
        "--no-advance",
        dest="advance",
        action="store_false",
        help="leave Omi's read cursor untouched (compatibility/debug mode)",
    )
    parser.set_defaults(advance=True)
    return parser


if __name__ == "__main__":
    raise SystemExit(sync(build_parser().parse_args()))
