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
    "failed/needs_better_input",
}

text = index.read_text(encoding="utf-8")
fixtures = sorted(fixture_dir.glob("validate-coverage-task-*.json"))
real_project_fixtures = sorted((fixture_dir / "real-project-agent-loop").glob("*.json"))

failures = []
seen_actions = set()

def has_key(value, key):
    if isinstance(value, dict):
        return key in value or any(has_key(child, key) for child in value.values())
    if isinstance(value, list):
        return any(has_key(child, key) for child in value)
    return False

if not fixtures:
    failures.append(f"{fixture_dir}: no fixture JSON files found")

for snippet in [
    "node scripts/verify-release-response-adopter-artifact.mjs /path/to/testloop-release-response-adopter-artifacts",
    "testloop-release-response-adopter-artifacts/",
]:
    if snippet not in text:
        failures.append(f"{index}: missing release response adopter artifact verifier snippet {snippet!r}")

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

for fixture in real_project_fixtures:
    try:
        data = json.loads(fixture.read_text(encoding="utf-8"))
    except json.JSONDecodeError as exc:
        failures.append(f"{fixture}: invalid JSON at line {exc.lineno}, column {exc.colno}: {exc.msg}")
        continue

    rel = fixture.relative_to(fixture_dir).as_posix()
    if rel not in text:
        failures.append(f"{index}: missing real project fixture link/name for {rel}")
    status = data.get("status")
    action = data.get("action")
    if action == "ready" and status != "passed":
        failures.append(f"{fixture}: expected passed/ready real project fixture")
    elif str(action).startswith("manual_review_") and status not in {"passed", "failed"}:
        failures.append(f"{fixture}: expected passed or failed manual_review_* real project fixture")
    elif action != "ready" and not str(action).startswith("manual_review_"):
        failures.append(f"{fixture}: expected ready or manual_review_* real project fixture")
    if has_key(data, "raw_output"):
        failures.append(f"{fixture}: raw_output must not be stored in real project fixture")
    if "redaction_note" not in data:
        failures.append(f"{fixture}: missing redaction_note")

if failures:
    print("fixture index test failed:", file=sys.stderr)
    for failure in failures:
        print(f"- {failure}", file=sys.stderr)
    sys.exit(1)

print(f"fixture index test passed ({len(fixtures)} task fixtures, {len(real_project_fixtures)} real project fixtures)")
PY
