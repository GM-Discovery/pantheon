#!/usr/bin/env python3
"""Suggest local Echo voice-profile matches without changing identity state."""

from __future__ import annotations

import argparse
import datetime as dt
import io
import json
import pathlib
import sqlite3
import subprocess
import wave

import numpy as np
import torch
from pyannote.audio import Pipeline


SAMPLE_RATE = 16000
WINDOW_SECONDS = 3.0


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--db", required=True, type=pathlib.Path)
    parser.add_argument("--run-id", required=True)
    parser.add_argument("--output", required=True, type=pathlib.Path)
    parser.add_argument("--model", default="pyannote/speaker-diarization-community-1")
    parser.add_argument("--max-segments", type=int, default=500, help="bound CPU work; zero means all unassigned segments")
    parser.add_argument("--threshold", type=float, default=0.65)
    parser.add_argument("--margin", type=float, default=0.10)
    return parser.parse_args()


def read_audio(path: pathlib.Path) -> np.ndarray:
    completed = subprocess.run(
        ["ffmpeg", "-v", "error", "-i", str(path), "-ar", str(SAMPLE_RATE), "-ac", "1", "-f", "wav", "pipe:1"],
        check=True,
        stdout=subprocess.PIPE,
    )
    with wave.open(io.BytesIO(completed.stdout)) as stream:
        return np.frombuffer(stream.readframes(stream.getnframes()), dtype=np.int16).astype(np.float32) / 32768.0


def fixed_clip(audio: np.ndarray, start: float, end: float) -> np.ndarray:
    center = max(0.0, (float(start) + float(end)) / 2.0)
    width = int(WINDOW_SECONDS * SAMPLE_RATE)
    begin = int(center * SAMPLE_RATE - width / 2)
    begin = max(0, min(begin, max(0, len(audio) - width)))
    clip = audio[begin : begin + width]
    if len(clip) == width:
        return clip
    if len(clip) == 0:
        return np.zeros(width, dtype=np.float32)
    return np.pad(clip, (0, width - len(clip)), mode="edge")


def normalize(vector: np.ndarray) -> np.ndarray:
    norm = np.linalg.norm(vector)
    return vector / norm if norm else vector


def embed(model, clips: list[np.ndarray]) -> np.ndarray:
    with torch.inference_mode():
        output = model(torch.from_numpy(np.stack(clips)).unsqueeze(1))
    return np.asarray(output, dtype=np.float32)


def main() -> None:
    args = parse_args()
    connection = sqlite3.connect(args.db)
    connection.row_factory = sqlite3.Row
    profiles = connection.execute(
        "SELECT voice_id, name FROM echo_voice_profiles WHERE state != 'disabled' ORDER BY name"
    ).fetchall()
    samples = connection.execute(
        "SELECT voice_id, source_path, start_seconds, end_seconds FROM echo_voice_samples ORDER BY voice_id, created_at"
    ).fetchall()
    run = connection.execute("SELECT source_path FROM echo_runs WHERE run_id = ?", (args.run_id,)).fetchone()
    if run is None:
        raise SystemExit(f"unknown Echo run: {args.run_id}")
    query = "SELECT segment_id, start_seconds, end_seconds, anonymous_speaker FROM echo_speaker_segments WHERE run_id = ? AND mapping_state != 'confirmed' ORDER BY start_seconds"
    if args.max_segments > 0:
        query += f" LIMIT {int(args.max_segments)}"
    segments = connection.execute(query, (args.run_id,)).fetchall()
    connection.close()
    if not profiles or not samples or not segments:
        args.output.write_text(json.dumps({"run_id": args.run_id, "suggestions": [], "reason": "profiles, samples, or unassigned segments are missing"}, indent=2) + "\n")
        return

    pipeline = Pipeline.from_pretrained(args.model)
    embedding_model = pipeline._embedding
    audio_cache: dict[str, np.ndarray] = {}

    def audio_for(path: str) -> np.ndarray:
        if path not in audio_cache:
            audio_cache[path] = read_audio(pathlib.Path(path))
        return audio_cache[path]

    profile_names = {row["voice_id"]: row["name"] for row in profiles}
    profile_vectors: dict[str, list[np.ndarray]] = {voice_id: [] for voice_id in profile_names}
    sample_jobs: list[tuple[str, np.ndarray]] = []
    for row in samples:
        if row["voice_id"] not in profile_vectors:
            continue
        try:
            sample_jobs.append((row["voice_id"], fixed_clip(audio_for(row["source_path"]), row["start_seconds"], row["end_seconds"])))
        except (OSError, subprocess.CalledProcessError):
            continue
    for offset in range(0, len(sample_jobs), 16):
        batch = sample_jobs[offset : offset + 16]
        vectors = embed(embedding_model, [item[1] for item in batch])
        for (voice_id, _), vector in zip(batch, vectors):
            profile_vectors[voice_id].append(normalize(vector))
    centroids = {voice_id: normalize(np.mean(vectors, axis=0)) for voice_id, vectors in profile_vectors.items() if vectors}
    if not centroids:
        raise SystemExit("no usable local voice samples could be embedded")

    segment_jobs = [
        (row, fixed_clip(audio_for(run["source_path"]), row["start_seconds"], row["end_seconds"]))
        for row in segments
        if float(row["end_seconds"]) - float(row["start_seconds"]) >= 0.75
    ]
    suggestions = []
    for offset in range(0, len(segment_jobs), 16):
        batch = segment_jobs[offset : offset + 16]
        vectors = embed(embedding_model, [item[1] for item in batch])
        for (row, _), vector in zip(batch, vectors):
            scores = sorted(((float(np.dot(normalize(vector), centroid)), voice_id) for voice_id, centroid in centroids.items()), reverse=True)
            if not scores or scores[0][0] < args.threshold or (len(scores) > 1 and scores[0][0] - scores[1][0] < args.margin):
                continue
            suggestions.append({
                "segment_id": row["segment_id"],
                "anonymous_speaker": row["anonymous_speaker"],
                "start_seconds": row["start_seconds"],
                "end_seconds": row["end_seconds"],
                "candidate_voice_id": scores[0][1],
                "candidate_name": profile_names[scores[0][1]],
                "score": round(scores[0][0], 4),
                "runner_up_score": round(scores[1][0], 4) if len(scores) > 1 else None,
                "state": "suggested",
            })

    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps({
        "generated_at": dt.datetime.now(dt.timezone.utc).isoformat(),
        "run_id": args.run_id,
        "model": args.model,
        "threshold": args.threshold,
        "margin": args.margin,
        "suggestions": suggestions,
        "boundaries": ["Suggestions are local biometric inferences, not identity facts.", "No Echo segment was changed by this command."],
    }, indent=2) + "\n")
    print(json.dumps({"run_id": args.run_id, "suggestions": len(suggestions), "output": str(args.output)}))


if __name__ == "__main__":
    main()
