#!/usr/bin/env python3
import pathlib
import struct
import sys
import unittest


sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import omi_ble_storage_capture as storage  # noqa: E402


class OmiBLEStorageCaptureTests(unittest.TestCase):
    def test_status_is_little_endian(self):
        value = struct.pack("<IIII", 444, 1, 1024, 1)
        self.assertEqual(
            storage.parse_status(value),
            storage.RingStatus(used_bytes=444, unread_packets=1, free_bytes=1024, rtc_valid=1),
        )

    def test_info_notification_is_big_endian(self):
        value = bytes([storage.NOTIFY_INFO]) + struct.pack(">QQIQH", 12, 48, 100, 2, 444)
        self.assertEqual(storage.parse_info_notification(value), storage.RingInfo(12, 48, 100, 2, 444))

    def test_read_begin_notification_is_big_endian(self):
        value = bytes([storage.NOTIFY_READ_BEGIN]) + struct.pack(">QI", 12, 36)
        self.assertEqual(storage.parse_read_begin_notification(value), (12, 36))

    def test_data_reassembler_handles_unaligned_notifications(self):
        reassembler = storage.RingRecordReassembler()
        record = bytes.fromhex("01020304") + bytes([0] * storage.RING_AUDIO_BYTES)
        reassembler.append(record[:17])
        self.assertEqual(reassembler.drain(), [])
        reassembler.append(record[17:])
        self.assertEqual(reassembler.drain(), [record])
        self.assertEqual(reassembler.pending_bytes, 0)

    def test_packed_opus_frames_ignore_padding(self):
        payload = bytes([3]) + b"abc" + bytes([2]) + b"de" + bytes([0] * (storage.RING_AUDIO_BYTES - 7))
        self.assertEqual(storage.parse_audio_payload(payload), [b"abc", b"de"])

    def test_commands_use_documented_big_endian_wire_shape(self):
        self.assertEqual(storage.ring_command(storage.CMD_RING_INFO), "10")
        self.assertEqual(
            storage.ring_command(storage.CMD_RING_READ, (12, 8), (36, 4)),
            "11000000000000000c00000024",
        )
        self.assertEqual(
            storage.ring_command(storage.CMD_RING_ADVANCE, (48, 8)),
            "120000000000000030",
        )


if __name__ == "__main__":
    unittest.main()
