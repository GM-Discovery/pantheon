#!/usr/bin/env python3
"""Join Whisper timestamped segments to diarization and write cited Apollo chunks."""

import argparse
import csv
import datetime as dt
import hashlib
import json
import pathlib


def read_diarization(path: pathlib.Path):
    with path.open(newline="", encoding="utf-8") as stream:
        rows = []
        for row in csv.DictReader(stream, delimiter="\t"):
            start = float(row["start_seconds"])
            end = float(row["end_seconds"])
            if end > start and row.get("speaker", "").strip():
                rows.append((start, end, row["speaker"].strip()))
    return rows


def overlap(start, end, other_start, other_end):
    return max(0.0, min(end, other_end) - max(start, other_start))


def timestamp(seconds):
    whole = max(0, int(seconds))
    return f"{whole // 3600:02d}:{(whole % 3600) // 60:02d}:{whole % 60:02d}"


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--source", required=True, type=pathlib.Path)
    parser.add_argument("--diarization", required=True, type=pathlib.Path)
    parser.add_argument("--transcript-json", required=True, type=pathlib.Path)
    parser.add_argument("--output-dir", required=True, type=pathlib.Path)
    parser.add_argument("--label", required=True)
    parser.add_argument("--chunk-seconds", type=int, default=600)
    args = parser.parse_args()
    if args.chunk_seconds < 60:
        raise SystemExit("--chunk-seconds must be at least 60")

    diarization = read_diarization(args.diarization)
    transcript = json.loads(args.transcript_json.read_text(encoding="utf-8"))
    utterances = []
    for item in transcript.get("transcription", []):
        start = float(item.get("offsets", {}).get("from", 0)) / 1000.0
        end = float(item.get("offsets", {}).get("to", 0)) / 1000.0
        text = " ".join(str(item.get("text", "")).split())
        if end <= start or not text:
            continue
        candidates = [(overlap(start, end, ds, de), ds, de, speaker) for ds, de, speaker in diarization]
        if candidates:
            best = max(candidates, key=lambda candidate: (candidate[0], -abs(((candidate[1] + candidate[2]) / 2) - ((start + end) / 2))))
            speaker = best[3]
        else:
            speaker = "UNKNOWN"
        utterances.append((start, end, speaker, text))

    if not utterances:
        raise SystemExit("transcript JSON contains no timestamped utterances")
    args.output_dir.mkdir(parents=True, exist_ok=True)
    source_hash = hashlib.sha256(args.source.read_bytes()).hexdigest()
    chunks = {}
    for start, end, speaker, text in utterances:
        chunk_number = int(start // args.chunk_seconds)
        chunks.setdefault(chunk_number, []).append((start, end, speaker, text))

    chunk_paths = []
    for chunk_number in sorted(chunks):
        path = args.output_dir / f"{args.label}.chunk-{chunk_number:03d}.md"
        rows = chunks[chunk_number]
        body = [
            f"# Echo evidence chunk — {args.label} — {chunk_number:03d}",
            "",
            f"Source audio: `{args.source}`",
            f"Diarization: `{args.diarization}`",
            "Speaker labels are anonymous diarization labels, not person identities.",
            "",
            "## Timestamped transcript",
            "",
        ]
        body.extend(f"[{timestamp(start)}–{timestamp(end)}] **{speaker}:** {text}" for start, end, speaker, text in rows)
        path.write_text("\n".join(body) + "\n", encoding="utf-8")
        chunk_paths.append(path)

    index_path = args.output_dir / f"{args.label}.index.md"
    index = [
        f"# Echo evidence index — {args.label}",
        "",
        f"Generated: {dt.datetime.now(dt.timezone.utc).isoformat()}",
        f"Source audio: `{args.source}`",
        f"Source SHA-256: `{source_hash}`",
        f"Diarization: `{args.diarization}`",
        f"Transcript source: `{args.transcript_json}`",
        f"Chunk size: {args.chunk_seconds} seconds",
        f"Utterances: {len(utterances)}",
        "",
        "The chunks below are evidence for local Apollo. Speaker labels are anonymous and must not be converted into names without Fame evidence or human correction.",
        "",
        "## Chunks",
        "",
    ]
    index.extend(f"- `{path.name}`" for path in chunk_paths)
    index_path.write_text("\n".join(index) + "\n", encoding="utf-8")
    print(json.dumps({"index": str(index_path), "chunks": [str(path) for path in chunk_paths], "utterances": len(utterances)}), flush=True)


if __name__ == "__main__":
    main()
