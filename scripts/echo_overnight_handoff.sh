#!/usr/bin/env bash
set -euo pipefail

TOKEN=$(< /home/brick/pantheon-runtime/state/control.token)
OUTDIR="/home/brick/Documents/Obsidian Vault/80 Pantheon Raw/Audio/2026/08/Kernel24/Echo Apollo"
PY=/home/brick/pantheon-runtime/.echo-venv/bin/python
NOTIFY_URL=http://127.0.0.1:8765/v1/notify

notify() {
  local message="$1"
  curl -fsS -X POST "$NOTIFY_URL" \
    -H "Content-Type: application/json" \
    --data-binary "$(jq -n --arg input "$message" '{input:$input}')" >/dev/null || true
}

on_error() {
  notify "Overnight Echo/Apollo run stopped before completion. Check Panopticon for the exact boundary."
}
trap on_error ERR

mkdir -p "$OUTDIR"

run_one() {
  local label="$1" source="$2" transcript="$3" speakers="$4" title="$5"
  local tsv="/home/brick/pantheon-runtime/state/echo/${label}.diarization.tsv"
  while [ ! -s "$tsv" ]; do sleep 30; done

  "$PY" /home/brick/pantheon-runtime/scripts/echo_prepare_apollo.py \
    --source "$source" \
    --diarization "$tsv" \
    --transcript-json "$transcript" \
    --output-dir "$OUTDIR" \
    --label "$label" \
    --chunk-seconds 600

  curl -fsS -X POST http://127.0.0.1:8765/v1/echo/runs/import \
    -H "Content-Type: application/json" \
    --data-binary "$(jq -n --arg runid "echo-${label}" --arg source "$source" --arg diarization "$tsv" --argjson speakers "$speakers" '{run_id:$runid,source_path:$source,diarization_path:$diarization,speaker_count:$speakers}')" >/dev/null

  local refs_json
  refs_json=$(find "$OUTDIR" -maxdepth 1 -type f -name "${label}.chunk-*.md" -print | sort | jq -R -s 'split("\n") | map(select(length > 0))')
  local request
  request="Prepare a human-readable briefing for ${title}. Work from the timestamped, diarized evidence chunks. Extract a concise overview, decisions, action items, commitments, unresolved questions, and important facts. Attribute statements only to the anonymous speaker labels unless Fame evidence is present. Do not infer identity from names mentioned in the transcript. Cite the exact evidence chunks for material claims. Ignore transcript filler and explain uncertainty where the audio or attribution is ambiguous."
  local order_payload order order_id
  order_payload=$(jq -n --arg request "$request" --arg scope "echo-apollo:${label}" --argjson refs "$refs_json" '{request_text:$request,recipient:"apollo",capability:"apollo.reason",scope:$scope,purpose:"Create a cited human-readable briefing from a completed local Echo run.",payload:{source_refs:$refs}}')
  order=$(curl -fsS -X POST http://127.0.0.1:8765/v1/orders \
    -H "Content-Type: application/json" \
    -H "X-Pantheon-Control-Token: ${TOKEN}" \
    --data-binary "$order_payload")
  order_id=$(jq -r '.order_id // empty' <<< "$order")
  [ -n "$order_id" ]

  curl -fsS -X POST http://127.0.0.1:8765/v1/mercury/round \
    -H "Content-Type: application/json" \
    -H "X-Pantheon-Control-Token: ${TOKEN}" \
    --data-binary '{}' >/dev/null
}

run_one \
  recording-4speaker-20260820 \
  "/home/brick/Pantheon Intake/Recording.m4a" \
  "/home/brick/Documents/Obsidian Vault/80 Pantheon Raw/Audio/2026/08/Kernel24/pantheon-k24-safe-recording.json" \
  4 \
  "Recording 1"

run_one \
  recording2-2speaker-20260820 \
  "/home/brick/Pantheon Intake/Recording (2).m4a" \
  "/home/brick/Documents/Obsidian Vault/80 Pantheon Raw/Audio/2026/08/Kernel24/pantheon-k24-safe-recording2.json" \
  2 \
  "Recording 2"

notify "Echo finished both constrained diarization passes and Apollo produced the cited overnight briefings. Open Panopticon or the Apollo reports in the Vault."
trap - ERR
