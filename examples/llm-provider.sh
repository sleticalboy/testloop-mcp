#!/bin/sh
set -eu

tmp_file="$(mktemp)"
trap 'rm -f "$tmp_file"' EXIT
cat > "$tmp_file"

python3 - "$tmp_file" <<'PY'
import json
import os
import re
import subprocess
import sys
from pathlib import Path

with open(sys.argv[1], "r", encoding="utf-8") as f:
    payload = json.load(f)

code = payload.get("static_code", "")
source_file = payload.get("source_file", "source")


def candidate_files_from_payload_notes(payload):
    context = payload.get("context") or {}
    targets = context.get("targets") or []
    candidates = []
    seen = set()
    for target in targets:
        for note in target.get("payload_notes") or []:
            match = re.search(r"read candidate source files:\s*(.+)$", note)
            if not match:
                continue
            for raw_candidate in match.group(1).split(","):
                candidate = raw_candidate.strip()
                if candidate and candidate not in seen:
                    seen.add(candidate)
                    candidates.append(candidate)
    return candidates


def readable_candidate_context(source_file, candidates):
    source_path = Path(source_file)
    source_dir = source_path.parent if source_path.parent != Path("") else Path(".")
    sections = []
    for candidate in candidates:
        candidate_path = Path(candidate)
        if candidate_path.is_absolute() or ".." in candidate_path.parts:
            continue
        path = (source_dir / candidate_path).resolve()
        try:
            text = path.read_text(encoding="utf-8")
        except OSError:
            continue
        sections.append(f"### {candidate}\n```ts\n{text.rstrip()}\n```")
    return "\n\n".join(sections)


candidate_context = readable_candidate_context(source_file, candidate_files_from_payload_notes(payload))

prompt_parts = [
    "You are generating unit tests from a static draft.",
    "Use the static code as the base and only improve it when the extra context is relevant.",
    "",
    "## Request JSON",
    "```json",
    json.dumps(payload, ensure_ascii=False, indent=2),
    "```",
]
if candidate_context:
    prompt_parts.extend(["", "## Imported Type Context", candidate_context])
prompt = "\n".join(prompt_parts)

prompt_file = os.environ.get("TESTLOOP_LLM_PROVIDER_PROMPT_FILE")
if prompt_file:
    Path(prompt_file).write_text(prompt, encoding="utf-8")

model_cmd = os.environ.get("TESTLOOP_LLM_PROVIDER_MODEL_CMD")
if model_cmd:
    completed = subprocess.run(
        model_cmd,
        input=prompt,
        text=True,
        shell=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=False,
    )
    if completed.returncode != 0:
        sys.stderr.write(completed.stderr)
        sys.exit(completed.returncode)
    code = completed.stdout

if not code.strip():
    code = f"// No static test code was generated for {source_file}\n"

sys.stdout.write(json.dumps({"code": code}))
PY
