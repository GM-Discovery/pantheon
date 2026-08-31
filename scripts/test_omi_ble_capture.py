#!/usr/bin/env python3
import pathlib
import sys
import tempfile
import unittest
import wave


sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import omi_ble_capture  # noqa: E402


class OmiBLECaptureTests(unittest.TestCase):
    def test_characteristic_discovery(self):
        output = """
handle: 0x000e, char properties: 0x10, char value handle: 0x000f, uuid: 19b10001-e8f2-537e-4f6c-d104768a1214
handle: 0x0011, char properties: 0x02, char value handle: 0x0012, uuid: 19b10002-e8f2-537e-4f6c-d104768a1214
"""
        found = omi_ble_capture.parse_characteristics(output)
        self.assertEqual(found[omi_ble_capture.OMI_AUDIO_UUID], (0x000F, 0x000E))
        self.assertEqual(found[omi_ble_capture.OMI_CODEC_UUID], (0x0012, 0x0011))

    def test_notification_parser_strips_hex_formatting(self):
        parsed = omi_ble_capture.parse_notification(
            "Notification handle = 0x000f value: 01 aa bb 00 14 20"
        )
        self.assertEqual(parsed, (0x000F, bytes.fromhex("01 aa bb 00 14 20")))

    def test_wav_sink_is_finalized_atomically(self):
        with tempfile.TemporaryDirectory() as temporary:
            output = pathlib.Path(temporary) / "omi.wav"
            sink = omi_ble_capture.WavSink(output)
            sink.write(b"\x00\x00" * 160)
            sink.finish()
            self.assertTrue(output.exists())
            self.assertFalse(output.with_name("omi.wav.partial").exists())
            with wave.open(str(output), "rb") as stream:
                self.assertEqual(stream.getframerate(), 16000)
                self.assertEqual(stream.getnchannels(), 1)
                self.assertEqual(stream.getnframes(), 160)

    def test_empty_wav_is_not_published(self):
        with tempfile.TemporaryDirectory() as temporary:
            output = pathlib.Path(temporary) / "empty.wav"
            sink = omi_ble_capture.WavSink(output)
            with self.assertRaises(omi_ble_capture.OmiBLEError):
                sink.finish()
            self.assertFalse(output.exists())
            self.assertFalse(output.with_name("empty.wav.partial").exists())

    def test_opus_decoder_loads_system_libopus(self):
        decoder = omi_ble_capture.OpusDecoder()
        decoder.close()

    def test_cv1_codec_id_is_opus(self):
        self.assertIn(0x15, omi_ble_capture.OPUS_CODEC_IDS)


if __name__ == "__main__":
    unittest.main()
