#!/usr/bin/env python3
"""Run a narrow, local Apollo/Mistral calibration fixture without changing Pantheon state."""

from __future__ import annotations

import argparse
import datetime as dt
import json
import pathlib
import re
import sys
import urllib.request


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--fixture", type=pathlib.Path, default=pathlib.Path(__file__).with_name("fixtures") / "apollo_interaction_fixture.json")
    parser.add_argument("--endpoint", default="http://127.0.0.1:11434/api/chat")
    parser.add_argument("--model", default="mistral-small3.1:latest")
    parser.add_argument("--output", type=pathlib.Path)
    parser.add_argument("--timeout", type=int, default=900)
    parser.add_argument("--num-predict", type=int, default=512)
    parser.add_argument("--temperature", type=float, default=0.0)
    return parser.parse_args()


def request_text(fixture: dict) -> str:
    evidence = "\n\n".join(f"{item['citation']}\n{item['text']}" for item in fixture["evidence"])
    protocol = {
        "action_items": [{"speaker": "", "action": "", "time_constraint": "", "evidence": ""}],
        "commitments": [{"speaker": "", "commitment": "", "time_constraint": "", "evidence": ""}],
        "decisions": [{"speaker": "", "decision": "", "evidence": ""}],
        "unresolved_questions": [{"question": "", "evidence": ""}],
        "uncertainties": []
    }
    return (
        "You are Apollo performing a narrow interaction-extraction calibration. "
        "Return JSON only. Use empty arrays when a category has no explicit evidence. "
        "Do not infer names, ownership, dates, intent, or decisions. Keep speaker labels exactly as written. "
        "Every non-empty item must cite exactly one of the supplied evidence citations. "
        "Do not include citations that are not supplied.\n\n"
        f"REQUEST:\n{fixture['request']}\n\n"
        f"OUTPUT SHAPE:\n{json.dumps(protocol, indent=2)}\n\n"
        f"EVIDENCE:\n{evidence}\n"
    )


def decode_response(body: bytes) -> tuple[str, object | None]:
    envelope = json.loads(body.decode("utf-8"))
    content = str(envelope.get("message", {}).get("content", "")).strip()
    candidate = content
    if candidate.startswith("```"):
        candidate = candidate.split("\n", 1)[1] if "\n" in candidate else candidate
        candidate = candidate.rsplit("```", 1)[0].strip()
    try:
        return content, json.loads(candidate)
    except json.JSONDecodeError:
        match = re.search(r"\{.*\}", candidate, re.DOTALL)
        if not match:
            return content, None
        try:
            return content, json.loads(match.group(0))
        except json.JSONDecodeError:
            return content, None


def flattened(value: object) -> str:
    return json.dumps(value, ensure_ascii=False).lower()


def evaluate(fixture: dict, content: str, parsed: object | None) -> dict:
    expectations = fixture["expectations"]
    text = flattened(parsed if parsed is not None else content)
    raw = content.lower()
    missing_terms = [term for term in expectations["required_terms"] if term.lower() not in text]
    missing_citations = [citation for citation in expectations["required_citations"] if citation.lower() not in text]
    forbidden = [term for term in expectations["forbidden_terms"] if term.lower() in raw]
    return {
        "json_protocol": parsed is not None,
        "required_terms_present": not missing_terms,
        "missing_terms": missing_terms,
        "required_citations_present": not missing_citations,
        "missing_citations": missing_citations,
        "forbidden_terms_absent": not forbidden,
        "forbidden_terms": forbidden,
        "pass": parsed is not None and not missing_terms and not missing_citations and not forbidden,
    }


def main() -> int:
    args = parse_args()
    fixture = json.loads(args.fixture.read_text(encoding="utf-8"))
    prompt = request_text(fixture)
    payload = json.dumps({"model": args.model, "messages": [{"role": "user", "content": prompt}], "stream": False, "format": "json", "options": {"temperature": args.temperature, "num_predict": args.num_predict}}).encode("utf-8")
    request = urllib.request.Request(args.endpoint, data=payload, headers={"Content-Type": "application/json"}, method="POST")
    try:
        with urllib.request.urlopen(request, timeout=args.timeout) as response:
            content, parsed = decode_response(response.read())
    except Exception as error:  # noqa: BLE001 - calibration should report boundary clearly
        result = {"model": args.model, "endpoint": args.endpoint, "pass": False, "blocked": str(error)}
        rendered = json.dumps(result, indent=2) + "\n"
        if args.output:
            args.output.parent.mkdir(parents=True, exist_ok=True)
            args.output.write_text(rendered, encoding="utf-8")
        print(rendered, end="")
        return 2
    result = {
        "generated_at": dt.datetime.now(dt.timezone.utc).isoformat(),
        "fixture": str(args.fixture),
        "model": args.model,
        "endpoint": args.endpoint,
        "evaluation": evaluate(fixture, content, parsed),
        "parsed": parsed,
        "raw_response": content,
    }
    rendered = json.dumps(result, indent=2, ensure_ascii=False) + "\n"
    if args.output:
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(rendered, encoding="utf-8")
    print(rendered, end="")
    return 0 if result["evaluation"]["pass"] else 1


if __name__ == "__main__":
    sys.exit(main())
