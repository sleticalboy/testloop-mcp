#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"

cd "$repo_root"

python3 - <<'PY'
from pathlib import Path
import json
import sys

fixture = Path("docs/fixtures/run-tests/apply-fix-suggestions.json")
fixtures_doc = Path("docs/fixtures.md")
client_doc = Path("docs/client-integration.md")

failures = []

try:
    payload = json.loads(fixture.read_text(encoding="utf-8"))
except FileNotFoundError:
    failures.append(f"{fixture}: missing")
    payload = {}
except json.JSONDecodeError as exc:
    failures.append(f"{fixture}: invalid JSON at line {exc.lineno}, column {exc.colno}: {exc.msg}")
    payload = {}

if payload:
    if payload.get("status") != "fail":
        failures.append(f"{fixture}: status={payload.get('status')!r}, want 'fail'")
    if payload.get("action") != "apply_fix_suggestions":
        failures.append(f"{fixture}: action={payload.get('action')!r}, want 'apply_fix_suggestions'")
    suggestions = payload.get("fix_suggestions")
    if not isinstance(suggestions, list) or not suggestions:
        failures.append(f"{fixture}: missing fix_suggestions")
    else:
        first = suggestions[0]
        if first.get("category") != "expectation_mismatch":
            failures.append(f"{fixture}: category={first.get('category')!r}, want 'expectation_mismatch'")
        repair = first.get("repair_task")
        if not isinstance(repair, dict):
            failures.append(f"{fixture}: missing repair_task")
        else:
            if repair.get("target_file") != "calc_test.go":
                failures.append(f"{fixture}: target_file={repair.get('target_file')!r}, want 'calc_test.go'")
            if "go test ./..." not in repair.get("suggested_commands", []):
                failures.append(f"{fixture}: missing suggested command 'go test ./...'")

fixtures_text = fixtures_doc.read_text(encoding="utf-8")
client_text = client_doc.read_text(encoding="utf-8")
for doc, text in ((fixtures_doc, fixtures_text), (client_doc, client_text)):
    if "run-tests/apply-fix-suggestions.json" not in text:
        failures.append(f"{doc}: missing run-tests fixture link")
if "`fail/apply_fix_suggestions`" not in fixtures_text:
    failures.append(f"{fixtures_doc}: missing status/action `fail/apply_fix_suggestions`")
if "fix_suggestions[0].category" not in client_text:
    failures.append(f"{client_doc}: missing category assertion guidance")

if failures:
    print("run_tests fixture index test failed:", file=sys.stderr)
    for failure in failures:
        print(f"- {failure}", file=sys.stderr)
    sys.exit(1)

print("run_tests fixture index test passed")
PY
