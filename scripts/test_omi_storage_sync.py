#!/usr/bin/env python3
import argparse
import json
import pathlib
import sys
import tempfile
import unittest
from unittest import mock
import wave


sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import omi_storage_sync as sync  # noqa: E402


class OmiStorageSyncTests(unittest.TestCase):
    def test_unique_outputs_do_not_collide(self):
        root = pathlib.Path("/tmp")
        first = sync.unique_output(root)
        second = sync.unique_output(root)
        self.assertNotEqual(first, second)

    def test_empty_ring_writes_successful_receipt(self):
        with tempfile.TemporaryDirectory() as directory:
            args = argparse.Namespace(
                device="Omi", adapter="hci0", address_type="random", output_dir=pathlib.Path(directory),
                mtu=247, packet_count=36, max_batches=128, connect_timeout=20,
                info_timeout=12, read_timeout=45,
            )
            with mock.patch.object(sync, "ensure_device_connected"), mock.patch.object(sync, "capture_batch", return_value=(None, "test")):
                self.assertEqual(sync.sync(args), 0)
            receipts = list(pathlib.Path(directory).glob("omi-sync-*.json"))
            self.assertEqual(len(receipts), 1)
            self.assertEqual(json.loads(receipts[0].read_text())["state"], "empty")

    def test_capture_arguments_are_bounded_and_oldest_first(self):
        args = argparse.Namespace(
            device="Omi", adapter="hci0", address_type="random", output_dir=pathlib.Path("/tmp"),
            mtu=247, packet_count=36, max_batches=128, connect_timeout=20,
            info_timeout=12, read_timeout=45,
        )
        captured = sync.build_capture_args(args, pathlib.Path("/tmp/one.wav"))
        self.assertEqual(captured.packet_count, 36)
        self.assertFalse(captured.latest)
        self.assertTrue(captured.allow_empty)

    def test_assemble_cuttings_combines_raw_batches(self):
        with tempfile.TemporaryDirectory() as directory:
            root = pathlib.Path(directory)
            raw = root / ".raw"
            raw.mkdir()
            items = []
            for index in range(3):
                path = raw / f"omi-ring-20260827-07000{index}-000000000Z.wav"
                with wave.open(str(path), "wb") as output:
                    output.setnchannels(1)
                    output.setsampwidth(2)
                    output.setframerate(10)
                    output.writeframes((index.to_bytes(2, "little") * 10))
                items.append({"wav_path": str(path), "metadata_path": str(path.with_suffix(".json"))})

            cuttings = sync.assemble_cuttings(items, root, seconds=2)

            self.assertEqual(len(cuttings), 2)
            with wave.open(cuttings[0]["wav_path"], "rb") as output:
                self.assertEqual(output.getnframes(), 20)
            with wave.open(cuttings[1]["wav_path"], "rb") as output:
                self.assertEqual(output.getnframes(), 10)
            metadata = json.loads(pathlib.Path(cuttings[0]["metadata_path"]).read_text())
            self.assertEqual(metadata["kind"], "omi-audio-cutting")
            self.assertEqual(metadata["raw_batch_count"], 2)

    def test_bounded_capture_is_successful_and_stages_raw_files(self):
        with tempfile.TemporaryDirectory() as directory:
            root = pathlib.Path(directory)
            args = argparse.Namespace(
                device="Omi", adapter="hci0", address_type="random", output_dir=root,
                mtu=247, packet_count=36, max_batches=1, cutting_seconds=1,
                connect_timeout=20, info_timeout=12, read_timeout=45,
            )
            def capture(_args, output):
                with wave.open(str(output), "wb") as wav:
                    wav.setnchannels(1)
                    wav.setsampwidth(2)
                    wav.setframerate(10)
                    wav.writeframes(b"\x00\x00" * 10)
                return output, "test"

            with mock.patch.object(sync, "ensure_device_connected"), mock.patch.object(sync, "capture_batch", side_effect=capture):
                self.assertEqual(sync.sync(args), 0)
            receipts = list(root.glob("omi-sync-*.json"))
            receipt = json.loads(receipts[0].read_text())
            self.assertEqual(receipt["state"], "completed")
            self.assertTrue(receipt["bounded"])
            self.assertEqual(receipt["cutting_count"], 1)
            self.assertTrue((root / ".raw").is_dir())
            self.assertEqual(list(root.glob("omi-cutting-*.wav")), [root / receipt["outputs"][0]["wav_path"].split("/")[-1]])

    def test_connection_failure_writes_durable_receipt(self):
        with tempfile.TemporaryDirectory() as directory:
            args = argparse.Namespace(
                device="Omi", adapter="hci0", address_type="random", output_dir=pathlib.Path(directory),
                mtu=247, packet_count=36, max_batches=128, connect_timeout=20,
                info_timeout=12, read_timeout=45,
            )
            with mock.patch.object(sync, "ensure_device_connected", side_effect=RuntimeError("not connectable")):
                self.assertEqual(sync.sync(args), 1)
            receipts = list(pathlib.Path(directory).glob("omi-sync-*.json"))
            receipt = json.loads(receipts[0].read_text())
            self.assertEqual(receipt["state"], "failed")
            self.assertIn("connection preflight", receipt["error"])

    def test_owned_connection_is_released_after_extraction(self):
        with tempfile.TemporaryDirectory() as directory:
            root = pathlib.Path(directory)
            args = argparse.Namespace(
                device="Omi", adapter="hci0", address_type="random", output_dir=root,
                mtu=247, packet_count=36, max_batches=1, cutting_seconds=1,
                connect_timeout=20, info_timeout=12, read_timeout=45,
            )

            def capture(_args, output):
                with wave.open(str(output), "wb") as wav:
                    wav.setnchannels(1)
                    wav.setsampwidth(2)
                    wav.setframerate(10)
                    wav.writeframes(b"\x00\x00" * 10)
                output.with_suffix(".json").write_text(json.dumps({"offline_storage_quiet": True}))
                return output, "test"

            with mock.patch.object(sync, "ensure_device_connected", return_value=True), \
                    mock.patch.object(sync, "disconnect_device") as disconnect, \
                    mock.patch.object(sync, "capture_batch", side_effect=capture):
                self.assertEqual(sync.sync(args), 0)
                disconnect.assert_called_once_with(args)

            receipt = json.loads(next(root.glob("omi-sync-*.json")).read_text())
            self.assertTrue(receipt["connection_owned"])
            self.assertTrue(receipt["disconnected_after"])
            self.assertTrue(receipt["offline_storage_quiet"])


if __name__ == "__main__":
    unittest.main()
