#!/usr/bin/env python3
"""Run a bounded local pyannote diarization pass and preserve its provenance."""

import argparse
import datetime as dt
import json
import os
import pathlib
import subprocess
import tempfile
import wave

import numpy as np
import torch
from pyannote.audio import Pipeline


def read_mono_wav(path: pathlib.Path):
    with wave.open(str(path), "rb") as stream:
        if stream.getnchannels() != 1 or stream.getsampwidth() != 2:
            raise RuntimeError("normalized audio must be mono 16-bit PCM")
        sample_rate = stream.getframerate()
        samples = np.frombuffer(stream.readframes(stream.getnframes()), dtype=np.int16).copy()
    waveform = torch.from_numpy(samples).float().unsqueeze(0) / 32768.0
    return waveform, sample_rate


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--source", required=True, type=pathlib.Path)
    parser.add_argument("--speakers", required=True, type=int)
    parser.add_argument("--output-prefix", required=True, type=pathlib.Path)
    parser.add_argument("--model", default="pyannote/speaker-diarization-community-1")
    args = parser.parse_args()
    if not args.source.is_file():
        raise SystemExit(f"source does not exist: {args.source}")
    if args.speakers < 1:
        raise SystemExit("--speakers must be positive")

    args.output_prefix.parent.mkdir(parents=True, exist_ok=True)
    started = dt.datetime.now(dt.timezone.utc).isoformat()
    with tempfile.TemporaryDirectory(prefix="pantheon-echo-") as work:
        normalized = pathlib.Path(work) / "audio.wav"
        print(f"converting {args.source} to normalized WAV", flush=True)
        subprocess.run(
            ["ffmpeg", "-hide_banner", "-loglevel", "error", "-y", "-i", str(args.source), "-vn", "-ar", "16000", "-ac", "1", "-c:a", "pcm_s16le", str(normalized)],
            check=True,
        )
        waveform, sample_rate = read_mono_wav(normalized)
        print(f"loading {args.model} for {args.speakers} speakers", flush=True)
        pipeline = Pipeline.from_pretrained(args.model)
        print("diarizing", flush=True)
        output = pipeline({"waveform": waveform, "sample_rate": sample_rate}, num_speakers=args.speakers)
        annotation = getattr(output, "speaker_diarization", output)

    rows = []
    for turn, _, speaker in annotation.itertracks(yield_label=True):
        rows.append({"start": float(turn.start), "end": float(turn.end), "speaker": str(speaker)})
    rows.sort(key=lambda item: (item["start"], item["end"], item["speaker"]))
    if not rows:
        raise SystemExit("diarization produced no segments")
    speakers = sorted({row["speaker"] for row in rows})
    tsv_path = args.output_prefix.with_suffix(".diarization.tsv")
    rttm_path = args.output_prefix.with_suffix(".rttm")
    meta_path = args.output_prefix.with_suffix(".json")
    with tsv_path.open("w", encoding="utf-8") as stream:
        stream.write("start_seconds\tend_seconds\tspeaker\tduration_seconds\n")
        for row in rows:
            stream.write(f"{row['start']:.3f}\t{row['end']:.3f}\t{row['speaker']}\t{row['end'] - row['start']:.3f}\n")
    with rttm_path.open("w", encoding="utf-8") as stream:
        uri = args.source.stem.replace(" ", "_")
        for row in rows:
            stream.write(f"SPEAKER {uri} 1 {row['start']:.3f} {row['end'] - row['start']:.3f} <NA> <NA> {row['speaker']} <NA> <NA>\n")
    metadata = {
        "source_path": str(args.source),
        "source_size": args.source.stat().st_size,
        "model": args.model,
        "requested_speakers": args.speakers,
        "detected_speakers": len(speakers),
        "speaker_labels": speakers,
        "segment_count": len(rows),
        "started_at": started,
        "completed_at": dt.datetime.now(dt.timezone.utc).isoformat(),
        "tsv_path": str(tsv_path),
        "rttm_path": str(rttm_path),
    }
    meta_path.write_text(json.dumps(metadata, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(metadata), flush=True)


if __name__ == "__main__":
    main()
