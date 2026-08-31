#!/usr/bin/env python3
"""Assemble Echo's timestamped evidence chunks into one verbatim transcript."""

from __future__ import annotations

import argparse
import pathlib
import re


CHUNK_RE = re.compile(r"\.chunk-(\d{3})\.md$")


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--chunk-dir", required=True, type=pathlib.Path)
    parser.add_argument("--label", required=True)
    parser.add_argument("--source", required=True)
    parser.add_argument("--output", required=True, type=pathlib.Path)
    args = parser.parse_args()

    chunks = []
    prefix = f"{args.label}.chunk-"
    for path in args.chunk_dir.glob(f"{prefix}*.md"):
        match = CHUNK_RE.search(path.name)
        if match:
            chunks.append((int(match.group(1)), path))
    chunks.sort()
    if not chunks:
        raise SystemExit(f"no chunks found for {args.label} in {args.chunk_dir}")

    sections = [
        f"# Verbatim diarized transcript — {args.label}",
        "",
        f"Source audio: `{args.source}`",
        f"Evidence chunks: `{args.chunk_dir}`",
        "",
        "Speaker labels are anonymous diarization labels. No names or roles are inferred from words spoken in the recording.",
        "",
        "## Transcript",
        "",
    ]
    for number, path in chunks:
        text = path.read_text(encoding="utf-8")
        marker = "## Timestamped transcript"
        if marker not in text:
            raise SystemExit(f"missing transcript section in {path}")
        body = text.split(marker, 1)[1].lstrip("\n")
        sections.append(f"### Evidence chunk {number:03d}")
        sections.append("")
        sections.append(body.rstrip())
        sections.append("")

    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text("\n".join(sections).rstrip() + "\n", encoding="utf-8")
    print(f"wrote {args.output} from {len(chunks)} chunks")


if __name__ == "__main__":
    main()
