#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"

cd "$repo_root"

python3 - <<'PY'
from pathlib import Path
import json
import sys

index = Path("docs/fixtures.md")
fixture_dir = Path("docs/fixtures")
expected_actions = {
    "passed/ready",
    "passed/manual_review_internal",
    "failed/apply_fix_suggestions",
}

text = index.read_text(encoding="utf-8")
fixtures = sorted(fixture_dir.glob("*.json"))

failures = []
seen_actions = set()

if not fixtures:
    failures.append(f"{fixture_dir}: no fixture JSON files found")

for fixture in fixtures:
    try:
        data = json.loads(fixture.read_text(encoding="utf-8"))
    except json.JSONDecodeError as exc:
        failures.append(f"{fixture}: invalid JSON at line {exc.lineno}, column {exc.colno}: {exc.msg}")
        continue

    status = data.get("status")
    action = data.get("action")
    if not isinstance(status, str) or not status:
        failures.append(f"{fixture}: missing string status")
        continue
    if not isinstance(action, str) or not action:
        failures.append(f"{fixture}: missing string action")
        continue

    action_key = f"{status}/{action}"
    seen_actions.add(action_key)
    if fixture.name not in text:
        failures.append(f"{index}: missing fixture link/name for {fixture.name}")
    if f"`{action_key}`" not in text:
        failures.append(f"{index}: missing status/action entry `{action_key}` for {fixture.name}")

missing = expected_actions - seen_actions
extra = seen_actions - expected_actions
if missing:
    failures.append("missing expected fixture actions: " + ", ".join(sorted(missing)))
if extra:
    failures.append("unexpected fixture actions; update expected_actions and docs/fixtures.md: " + ", ".join(sorted(extra)))

if failures:
    print("fixture index test failed:", file=sys.stderr)
    for failure in failures:
        print(f"- {failure}", file=sys.stderr)
    sys.exit(1)

print(f"fixture index test passed ({len(fixtures)} fixtures)")
PY
