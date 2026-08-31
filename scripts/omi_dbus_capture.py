#!/usr/bin/env python3
"""Capture Omi live audio through BlueZ's native D-Bus GATT API.

This is the Kernel 26 transport comparison path. It deliberately uses the
system-provided dbus-python and GLib bindings rather than adding Bleak or
another package. The sequence mirrors Omi's documented client: connect,
synchronize the device clock, subscribe to the audio characteristic, and
consume Value property notifications.
"""

from __future__ import annotations

import argparse
import sys
import time
from pathlib import Path

try:
    import dbus
    from dbus.mainloop.glib import DBusGMainLoop
    from gi.repository import GLib
except ImportError as exc:  # pragma: no cover - exercised on unsupported hosts
    raise SystemExit(f"native BlueZ D-Bus bindings are required: {exc}") from exc

import omi_ble_capture


OMI_AUDIO_UUID = omi_ble_capture.OMI_AUDIO_UUID
OMI_CODEC_UUID = omi_ble_capture.OMI_CODEC_UUID
OMI_TIME_SYNC_UUID = omi_ble_capture.OMI_TIME_SYNC_UUID
PACKET_HEADER_BYTES = omi_ble_capture.PACKET_HEADER_BYTES


class NativeBLEError(RuntimeError):
    pass


def object_manager(bus: dbus.SystemBus):
    return dbus.Interface(bus.get_object("org.bluez", "/"), "org.freedesktop.DBus.ObjectManager")


def managed_objects(bus: dbus.SystemBus):
    return object_manager(bus).GetManagedObjects()


def property_iface(bus: dbus.SystemBus, path: str):
    return dbus.Interface(bus.get_object("org.bluez", path), "org.freedesktop.DBus.Properties")


def find_device(objects, address: str) -> str:
    target = address.upper()
    for path, interfaces in objects.items():
        values = interfaces.get("org.bluez.Device1")
        if values and str(values.get("Address", "")).upper() == target:
            return str(path)
    raise NativeBLEError(f"BlueZ does not know Omi device {address}")


def find_characteristics(objects, device_path: str) -> dict[str, str]:
    wanted = {OMI_AUDIO_UUID, OMI_CODEC_UUID, OMI_TIME_SYNC_UUID}
    found: dict[str, str] = {}
    prefix = device_path + "/"
    for path, interfaces in objects.items():
        if not str(path).startswith(prefix):
            continue
        values = interfaces.get("org.bluez.GattCharacteristic1")
        if not values:
            continue
        uuid = str(values.get("UUID", "")).lower()
        if uuid in wanted:
            found[uuid] = str(path)
    missing = wanted - set(found)
    if missing:
        raise NativeBLEError(f"BlueZ GATT characteristics missing: {', '.join(sorted(missing))}")
    return found


def wait_for_services_resolved(
    bus: dbus.SystemBus, device_path: str, timeout: float = 20.0
) -> None:
    """Wait until BlueZ has materialized the device's GATT hierarchy.

    Device1.Connect() can return while the ATT link is up but service
    discovery is still in progress.  Reading ObjectManager immediately at
    that point produces a false "characteristics missing" failure.
    """
    props = property_iface(bus, device_path)
    if not bool(props.Get("org.bluez.Device1", "Connected")):
        raise NativeBLEError("Omi connected attempt returned without an ATT link")
    if bool(props.Get("org.bluez.Device1", "ServicesResolved")):
        return

    resolved = False
    loop = GLib.MainLoop()

    def on_properties_changed(interface, changed, invalidated, **kwargs):
        nonlocal resolved
        if interface != "org.bluez.Device1" or "ServicesResolved" not in changed:
            return
        resolved = bool(changed["ServicesResolved"])
        if resolved:
            loop.quit()

    bus.add_signal_receiver(
        on_properties_changed,
        dbus_interface="org.freedesktop.DBus.Properties",
        signal_name="PropertiesChanged",
        path=device_path,
        arg0="org.bluez.Device1",
    )
    try:
        # Recheck after installing the receiver so a fast discovery cannot
        # race the signal subscription.
        if bool(props.Get("org.bluez.Device1", "ServicesResolved")):
            return
        GLib.timeout_add(max(1000, int(timeout * 1000)), loop.quit)
        loop.run()
    finally:
        bus.remove_signal_receiver(
            on_properties_changed,
            dbus_interface="org.freedesktop.DBus.Properties",
            signal_name="PropertiesChanged",
            path=device_path,
            arg0="org.bluez.Device1",
        )
    if not resolved and not bool(props.Get("org.bluez.Device1", "ServicesResolved")):
        raise NativeBLEError(f"BlueZ GATT service discovery did not resolve within {timeout:g}s")


def byte_array(data: bytes):
    return dbus.Array([dbus.Byte(value) for value in data], signature="y")


def read_value(bus: dbus.SystemBus, path: str) -> bytes:
    characteristic = dbus.Interface(bus.get_object("org.bluez", path), "org.bluez.GattCharacteristic1")
    return bytes(int(value) for value in characteristic.ReadValue({}))


def write_value(bus: dbus.SystemBus, path: str, data: bytes) -> None:
    characteristic = dbus.Interface(bus.get_object("org.bluez", path), "org.bluez.GattCharacteristic1")
    characteristic.WriteValue(byte_array(data), {})


def capture(args: argparse.Namespace) -> Path:
    DBusGMainLoop(set_as_default=True)
    bus = dbus.SystemBus()
    objects = managed_objects(bus)
    device_path = find_device(objects, args.device)
    device = dbus.Interface(bus.get_object("org.bluez", device_path), "org.bluez.Device1")
    device_props = property_iface(bus, device_path)
    connected = bool(device_props.Get("org.bluez.Device1", "Connected"))
    if not connected:
        device.Connect()
    wait_for_services_resolved(bus, device_path)
    objects = managed_objects(bus)
    chars = find_characteristics(objects, device_path)
    device_state = property_iface(bus, device_path)
    print(f"native BlueZ connected={bool(device_state.Get('org.bluez.Device1', 'Connected'))}; device={args.device}", flush=True)
    codec = read_value(bus, chars[OMI_CODEC_UUID])
    codec_id = codec[0] if codec else 0
    print(f"native Omi codec=0x{codec_id:02x}; audio={chars[OMI_AUDIO_UUID]}", flush=True)
    write_value(bus, chars[OMI_TIME_SYNC_UUID], int(time.time()).to_bytes(4, "little"))

    output = Path(args.output).expanduser().resolve()
    sink = omi_ble_capture.WavSink(output)
    decoder = omi_ble_capture.OpusDecoder() if codec_id in omi_ble_capture.OPUS_CODEC_IDS else None
    packets = 0
    samples = 0
    errors: list[str] = []

    def on_properties_changed(interface, changed, invalidated, **kwargs):
        nonlocal packets, samples
        if interface != "org.bluez.GattCharacteristic1" or "Value" not in changed:
            return
        packet = bytes(int(value) for value in changed["Value"])
        if len(packet) <= PACKET_HEADER_BYTES:
            return
        try:
            pcm = decoder.decode(packet[PACKET_HEADER_BYTES:]) if decoder else packet[PACKET_HEADER_BYTES:]
            sink.write(pcm)
            packets += 1
            samples += len(pcm) // 2
        except Exception as exc:  # preserve the connection result for diagnosis
            errors.append(str(exc))

    bus.add_signal_receiver(
        on_properties_changed,
        dbus_interface="org.freedesktop.DBus.Properties",
        signal_name="PropertiesChanged",
        path=chars[OMI_AUDIO_UUID],
        arg0="org.bluez.GattCharacteristic1",
    )
    audio = dbus.Interface(bus.get_object("org.bluez", chars[OMI_AUDIO_UUID]), "org.bluez.GattCharacteristic1")
    audio.StartNotify()
    print("native audio notification subscription active; speak now", flush=True)
    loop = GLib.MainLoop()
    GLib.timeout_add(int(args.duration * 1000), loop.quit)
    loop.run()
    try:
        audio.StopNotify()
    finally:
        if decoder:
            decoder.close()
    if errors:
        sink.abort()
        raise NativeBLEError(f"native decoder errors: {errors[:3]}")
    if packets == 0:
        sink.abort()
        raise NativeBLEError("BlueZ subscription active but emitted no Omi audio notifications")
    finished = sink.finish()
    print(f"native capture wrote {finished}; packets={packets}; samples={samples}", flush=True)
    return finished


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--device", required=True)
    parser.add_argument("--duration", type=float, default=20)
    parser.add_argument("--output", required=True)
    return parser


def main() -> int:
    try:
        capture(build_parser().parse_args())
        return 0
    except (NativeBLEError, omi_ble_capture.OmiBLEError, dbus.DBusException, OSError, KeyboardInterrupt) as exc:
        print(f"omi-dbus-capture: {exc}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
