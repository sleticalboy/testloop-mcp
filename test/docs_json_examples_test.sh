#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"

cd "$repo_root"

python3 - <<'PY'
from pathlib import Path
import json
import re
import sys

doc = Path("docs/validate-coverage-task-samples.md")
text = doc.read_text(encoding="utf-8")
blocks = re.findall(r"```json\n(.*?)\n```", text, flags=re.S)

if not blocks:
    print(f"{doc}: no json examples found", file=sys.stderr)
    sys.exit(1)

failures = []
for index, block in enumerate(blocks, start=1):
    try:
        value = json.loads(block)
    except json.JSONDecodeError as exc:
        failures.append(f"block {index}: invalid JSON at line {exc.lineno}, column {exc.colno}: {exc.msg}")
        continue

    if not isinstance(value, dict):
        failures.append(f"block {index}: top-level JSON must be an object")
        continue
    for key in ("status", "action"):
        if key not in value or not isinstance(value[key], str) or not value[key]:
            failures.append(f"block {index}: missing string field {key!r}")
    if "coverage_task" not in value:
        failures.append(f"block {index}: missing coverage_task")
    if "metadata" not in value:
        failures.append(f"block {index}: missing metadata")

if failures:
    print("invalid documentation JSON examples:", file=sys.stderr)
    for failure in failures:
        print(f"- {failure}", file=sys.stderr)
    sys.exit(1)

print(f"documentation JSON examples test passed ({len(blocks)} examples)")
PY
