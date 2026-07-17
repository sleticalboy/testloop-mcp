#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"

cd "$repo_root"

python3 - <<'PY'
from pathlib import Path
import json
import sys

fixture_dir = Path("docs/fixtures")
fixtures_doc = Path("docs/fixtures.md")
client_doc = Path("docs/client-integration.md")

expected_decisions = {
    "passed/ready": "accept",
    "passed/manual_review_internal": "manual-review",
    "failed/apply_fix_suggestions": "apply-repair",
    "failed/needs_better_input": "needs-better-input",
}
expected_order = [
    "passed/ready",
    "passed/manual_review_internal",
    "failed/apply_fix_suggestions",
    "failed/needs_better_input",
]

required_client_snippets = {
    "accept": "`passed/ready` 映射为 `accept`",
    "manual-review": "`passed/manual_review_internal` 映射为 `manual-review`",
    "apply-repair": "`failed/apply_fix_suggestions` 映射为 `apply-repair`",
    "needs-better-input": "`failed/needs_better_input` 映射为 `needs-better-input`",
}

def decision_for(status, action):
    if status == "passed" and action == "ready":
        return "accept"
    if action.startswith("manual_review_"):
        return "manual-review"
    if action == "apply_fix_suggestions":
        return "apply-repair"
    if action == "needs_better_input":
        return "needs-better-input"
    if status == "generation_error":
        return "inspect-generation"
    if status == "run_error":
        return "inspect-runner"
    if status == "failed":
        return "repair-generated-test"
    return "inspect"

failures = []
fixtures = sorted(fixture_dir.glob("validate-coverage-task-*.json"))
if not fixtures:
    failures.append(f"{fixture_dir}: no validate coverage task fixtures found")

seen = {}
for fixture in fixtures:
    try:
        payload = json.loads(fixture.read_text(encoding="utf-8"))
    except json.JSONDecodeError as exc:
        failures.append(f"{fixture}: invalid JSON at line {exc.lineno}, column {exc.colno}: {exc.msg}")
        continue

    status = payload.get("status")
    action = payload.get("action")
    if not isinstance(status, str) or not isinstance(action, str):
        failures.append(f"{fixture}: status/action must be strings")
        continue

    key = f"{status}/{action}"
    decision = decision_for(status, action)
    seen[key] = decision
    expected = expected_decisions.get(key)
    if expected is None:
        failures.append(f"{fixture}: unexpected status/action {key}; update expected_decisions and docs")
    elif decision != expected:
        failures.append(f"{fixture}: decision for {key} = {decision}, want {expected}")

missing = set(expected_decisions) - set(seen)
if missing:
    failures.append("missing decision fixture actions: " + ", ".join(sorted(missing)))

fixtures_text = fixtures_doc.read_text(encoding="utf-8")
client_text = client_doc.read_text(encoding="utf-8")
for key, decision in expected_decisions.items():
    if f"`{key}`" not in fixtures_text:
        failures.append(f"{fixtures_doc}: missing `{key}` entry")
    snippet = required_client_snippets[decision]
    if snippet not in client_text:
        failures.append(f"{client_doc}: missing decision snippet {snippet!r}")

if failures:
    print("fixture decision mapping test failed:", file=sys.stderr)
    for failure in failures:
        print(f"- {failure}", file=sys.stderr)
    sys.exit(1)

ordered = [expected_decisions[key] for key in expected_order]
print(f"fixture decision mapping test passed ({len(fixtures)} fixtures, decisions={','.join(ordered)})")
PY
