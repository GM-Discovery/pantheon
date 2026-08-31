#!/usr/bin/env python3
"""Read Omi CV1 ring-buffer audio through native BlueZ D-Bus GATT."""

from __future__ import annotations

import argparse
import json
import os
import sys
import time
from pathlib import Path

try:
    import dbus
    from dbus.mainloop.glib import DBusGMainLoop
    from gi.repository import GLib
except ImportError as exc:  # pragma: no cover - unsupported hosts
    raise SystemExit(f"native BlueZ D-Bus bindings are required: {exc}") from exc

import omi_ble_storage_capture as storage
import omi_dbus_capture as native
import omi_ble_capture


RING_CONTROL_UUID = storage.RING_CONTROL_UUID
RING_STATUS_UUID = storage.RING_STATUS_UUID


def find_ring_characteristics(objects, device_path: str) -> dict[str, str]:
    found: dict[str, str] = {}
    prefix = device_path + "/"
    for path, interfaces in objects.items():
        if not str(path).startswith(prefix):
            continue
        values = interfaces.get("org.bluez.GattCharacteristic1")
        if not values:
            continue
        uuid = str(values.get("UUID", "")).lower()
        if uuid in {RING_CONTROL_UUID, RING_STATUS_UUID}:
            found[uuid] = str(path)
    missing = {RING_CONTROL_UUID, RING_STATUS_UUID} - set(found)
    if missing:
        raise native.NativeBLEError(f"Omi ring characteristics missing: {', '.join(sorted(missing))}")
    return found


def capture(args: argparse.Namespace) -> Path | None:
    DBusGMainLoop(set_as_default=True)
    bus = dbus.SystemBus()
    objects = native.managed_objects(bus)
    device_path = native.find_device(objects, args.device)
    device = dbus.Interface(bus.get_object("org.bluez", device_path), "org.bluez.Device1")
    device_props = native.property_iface(bus, device_path)
    connected = bool(device_props.Get("org.bluez.Device1", "Connected"))
    connected_by_reader = False
    if not connected:
        device.Connect()
        connected_by_reader = True
    native.wait_for_services_resolved(bus, device_path, args.info_timeout)
    objects = native.managed_objects(bus)
    chars = find_ring_characteristics(objects, device_path)
    control = dbus.Interface(bus.get_object("org.bluez", chars[RING_CONTROL_UUID]), "org.bluez.GattCharacteristic1")
    notifications: list[bytes] = []
    reassembler = storage.RingRecordReassembler()
    info: storage.RingInfo | None = None
    done: storage.RingDone | None = None
    read_begin: tuple[int, int] | None = None
    ack_status: int | None = None

    def on_properties_changed(interface, changed, invalidated, **kwargs):
        nonlocal info, done, read_begin, ack_status
        if interface != "org.bluez.GattCharacteristic1" or "Value" not in changed:
            return
        value = bytes(int(item) for item in changed["Value"])
        notifications.append(value)
        if len(value) >= 2 and value[0] == storage.NOTIFY_ACK:
            ack_status = value[1]
        if info is None:
            info = storage.parse_info_notification(value)
        parsed_done = storage.parse_done_notification(value)
        if parsed_done is not None:
            done = parsed_done
        parsed_begin = storage.parse_read_begin_notification(value)
        if parsed_begin is not None:
            read_begin = parsed_begin
        if info is not None and done is not None:
            # The read loop will stop on its next iteration.
            return

    bus.add_signal_receiver(
        on_properties_changed,
        dbus_interface="org.freedesktop.DBus.Properties",
        signal_name="PropertiesChanged",
        path=chars[RING_CONTROL_UUID],
        arg0="org.bluez.GattCharacteristic1",
    )

    output = Path(args.output).expanduser().resolve()
    output.parent.mkdir(parents=True, exist_ok=True)
    try:
        # Some Omi firmware revisions do not answer a direct ReadValue on the
        # ring-status characteristic even though the control characteristic is
        # available. INFO includes the same sequence information needed to
        # determine unread records, so use the control notification path for
        # both discovery and the subsequent bounded read.
        control.StartNotify()
        native.write_value(bus, chars[RING_CONTROL_UUID], bytes.fromhex(storage.ring_command(storage.CMD_RING_INFO)))
        info_loop = GLib.MainLoop()
        GLib.timeout_add(int(args.info_timeout * 1000), info_loop.quit)
        while info is None:
            info_loop.run()
            if info is not None:
                break
            raise native.NativeBLEError("timed out waiting for native Omi ring INFO notification")
        print(
            f"native Omi ring: unread={max(0, info.write_seq - info.read_seq)} "
            f"read_seq={info.read_seq} write_seq={info.write_seq}",
            flush=True,
        )
        if info.packet_size != storage.RING_RECORD_BYTES:
            raise native.NativeBLEError(f"unsupported Omi ring packet size {info.packet_size}; expected {storage.RING_RECORD_BYTES}")
        start_seq = info.read_seq
        if args.latest:
            start_seq = max(info.read_seq, info.write_seq - args.packet_count)
        available = info.write_seq - start_seq
        count = min(args.packet_count, available)
        if count <= 0:
            if args.allow_empty:
                print("Omi ring contains no unread packets", flush=True)
                return None
            raise native.NativeBLEError("Omi ring contains no unread packets")

        ring_info = info
        info = None
        done = None
        read_begin = None
        notifications.clear()
        reassembler = storage.RingRecordReassembler()
        records: list[bytes] = []
        command = storage.ring_command(storage.CMD_RING_READ, (start_seq, 8), (count, 4))
        native.write_value(bus, chars[RING_CONTROL_UUID], bytes.fromhex(command))
        read_loop = GLib.MainLoop()
        deadline = time.monotonic() + args.read_timeout

        def stop_when_done() -> bool:
            if done is not None or time.monotonic() >= deadline:
                read_loop.quit()
                return False
            return True

        GLib.timeout_add(50, stop_when_done)
        read_loop.run()
        for value in notifications:
            if value[:1] == bytes([storage.NOTIFY_DATA]):
                reassembler.append(value[1:])
                records.extend(reassembler.drain())
        if done is None:
            raise native.NativeBLEError(
                f"timed out waiting for native Omi ring DONE; records={len(records)} pending={reassembler.pending_bytes}"
            )
        if done.status != 0:
            raise native.NativeBLEError(f"Omi ring read failed with status 0x{done.status:02x}")
        if len(records) < count:
            raise native.NativeBLEError(f"Omi returned {len(records)} records, expected {count}")
        if done.next_seq != start_seq + count:
            raise native.NativeBLEError(
                f"Omi ring boundary mismatch: requested {start_seq}+{count}, DONE reported {done.next_seq}"
            )
        if read_begin is not None and read_begin != (start_seq, count):
            raise native.NativeBLEError(
                f"Omi ring READ_BEGIN mismatch: requested {(start_seq, count)}, received {read_begin}"
            )

        raw_records = records[:count]
        raw_partial = output.with_name(output.name + ".ring.partial")
        raw_output = output.with_suffix(".ring.bin")
        raw_partial.write_bytes(b"".join(raw_records))
        sink = omi_ble_capture.WavSink(output)
        decoder = omi_ble_capture.OpusDecoder()
        frame_count = 0
        try:
            for record in raw_records:
                if len(record) != storage.RING_RECORD_BYTES:
                    raise native.NativeBLEError("incomplete Omi ring record")
                for frame in storage.parse_audio_payload(record[4:]):
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
        metadata_value = {
            "device": args.device,
            "source": "omi_cv1_ring",
            "transport": "bluez_dbus",
            "read_seq": ring_info.read_seq,
            "write_seq": ring_info.write_seq,
            "capture_start_seq": start_seq,
            "packet_count": len(raw_records),
            "next_seq": done.next_seq,
            "read_begin": {
                "start_seq": read_begin[0],
                "packet_count": read_begin[1],
            } if read_begin is not None else None,
            "frame_count": frame_count,
            "raw_ring_path": str(raw_output),
            "wav_path": str(finished),
            "ring_advance_requested": bool(getattr(args, "advance", True)),
            "ring_advanced_to_seq": None,
            "write_seq_after_advance": None,
            "offline_storage_quiet": None,
        }

        # Commit the durable local archive before advancing Omi's read cursor.
        # If the process dies after this point, the next run can safely inspect
        # the receipt and continue from the device's acknowledged cursor.
        metadata_partial = metadata.with_name(metadata.name + ".partial")
        metadata_partial.write_text(json.dumps(metadata_value, indent=2) + "\n")
        os.replace(metadata_partial, metadata)

        if getattr(args, "advance", True):
            ack_status = None
            native.write_value(
                bus,
                chars[RING_CONTROL_UUID],
                bytes.fromhex(storage.ring_command(storage.CMD_RING_ADVANCE, (done.next_seq, 8))),
            )
            advance_loop = GLib.MainLoop()
            advance_deadline = time.monotonic() + args.advance_timeout

            def stop_when_advanced() -> bool:
                if ack_status is not None or time.monotonic() >= advance_deadline:
                    advance_loop.quit()
                    return False
                return True

            GLib.timeout_add(50, stop_when_advanced)
            advance_loop.run()
            if ack_status is None:
                raise native.NativeBLEError("timed out waiting for Omi ring ADVANCE acknowledgement")
            if ack_status != 0:
                raise native.NativeBLEError(f"Omi ring ADVANCE failed with status 0x{ack_status:02x}")

            # Read INFO again while the same BLE session is held.  A stable
            # write_seq is the useful safety signal here: the offline recorder
            # did not keep appending silence during this extraction.
            info = None
            native.write_value(bus, chars[RING_CONTROL_UUID], bytes.fromhex(storage.ring_command(storage.CMD_RING_INFO)))
            verify_loop = GLib.MainLoop()
            verify_deadline = time.monotonic() + args.info_timeout

            def stop_when_verified() -> bool:
                if info is not None or time.monotonic() >= verify_deadline:
                    verify_loop.quit()
                    return False
                return True

            GLib.timeout_add(50, stop_when_verified)
            verify_loop.run()
            if info is None:
                raise native.NativeBLEError("timed out verifying Omi ring INFO after ADVANCE")
            metadata_value["ring_advanced_to_seq"] = done.next_seq
            metadata_value["write_seq_after_advance"] = info.write_seq
            metadata_value["offline_storage_quiet"] = info.write_seq == ring_info.write_seq
            metadata_partial.write_text(json.dumps(metadata_value, indent=2) + "\n")
            os.replace(metadata_partial, metadata)
            if not metadata_value["offline_storage_quiet"]:
                raise native.NativeBLEError(
                    "Omi write_seq advanced during connected extraction; offline storage was not quiet"
                )
        print(f"native Omi storage wrote {finished} ({frame_count} Opus frames; {len(raw_records)} ring records)", flush=True)
        return finished
    finally:
        try:
            control.StopNotify()
        except Exception:
            pass
        if connected_by_reader:
            try:
                device.Disconnect()
            except Exception:
                pass


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--device", required=True)
    parser.add_argument("--output", required=True)
    parser.add_argument("--packet-count", type=int, default=36)
    parser.add_argument("--latest", action="store_true")
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
    parser.add_argument("--allow-empty", action="store_true")
    return parser


def main() -> int:
    try:
        capture(build_parser().parse_args())
        return 0
    except (native.NativeBLEError, omi_ble_capture.OmiBLEError, dbus.DBusException, OSError, KeyboardInterrupt) as exc:
        print(f"omi-dbus-storage-capture: {exc}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
