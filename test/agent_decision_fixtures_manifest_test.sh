#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"

cd "$repo_root"

python3 - <<'PY'
from pathlib import Path
import json
import sys

manifest_path = Path("docs/fixtures/agent-decision-fixtures.json")
schema_path = Path("docs/fixtures/agent-decision-fixtures.schema.json")
fixtures_doc = Path("docs/fixtures.md")
client_doc = Path("docs/client-integration.md")
contract_doc = Path("docs/mcp-client-contract-tests.md")

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

def has_key(value, key):
    if isinstance(value, dict):
        return key in value or any(has_key(child, key) for child in value.values())
    if isinstance(value, list):
        return any(has_key(child, key) for child in value)
    return False

failures = []
try:
    manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
except json.JSONDecodeError as exc:
    print(f"{manifest_path}: invalid JSON at line {exc.lineno}, column {exc.colno}: {exc.msg}", file=sys.stderr)
    raise SystemExit(1)

if manifest.get("$schema") != "./agent-decision-fixtures.schema.json":
    failures.append("$schema must point to ./agent-decision-fixtures.schema.json")
if manifest.get("schema_version") != 1:
    failures.append("schema_version must be 1")
if not schema_path.is_file():
    failures.append(f"missing schema file: {schema_path}")

doc_texts = {
    fixtures_doc: fixtures_doc.read_text(encoding="utf-8"),
    client_doc: client_doc.read_text(encoding="utf-8"),
    contract_doc: contract_doc.read_text(encoding="utf-8"),
}
for doc, text in doc_texts.items():
    for snippet in ("agent-decision-fixtures.json", "agent-decision-fixtures.schema.json"):
        if snippet not in text:
            failures.append(f"{doc}: missing {snippet}")
if "sh test/agent_decision_fixtures_manifest_test.sh" not in doc_texts[contract_doc]:
    failures.append(f"{contract_doc}: missing manifest test command")

fixtures = manifest.get("fixtures")
if not isinstance(fixtures, list) or not fixtures:
    failures.append("fixtures must be a non-empty list")
    fixtures = []

expected_paths = {
    *[path.as_posix() for path in Path("docs/fixtures").glob("validate-coverage-task-*.json")],
    *[path.as_posix() for path in (Path("docs/fixtures") / "real-project-agent-loop").glob("*.json")],
}
seen_paths = []
seen_decisions = []
for item in fixtures:
    if not isinstance(item, dict):
        failures.append(f"fixture entry must be object: {item!r}")
        continue
    path_text = item.get("path")
    if not isinstance(path_text, str) or not path_text:
        failures.append(f"fixture entry missing path: {item!r}")
        continue
    seen_paths.append(path_text)
    path = Path(path_text)
    if not path.is_file():
        failures.append(f"missing fixture file: {path}")
        continue
    try:
        payload = json.loads(path.read_text(encoding="utf-8"))
    except json.JSONDecodeError as exc:
        failures.append(f"{path}: invalid JSON at line {exc.lineno}, column {exc.colno}: {exc.msg}")
        continue

    status = payload.get("status")
    action = payload.get("action")
    if item.get("status") != status or item.get("action") != action:
        failures.append(f"{path}: status/action = {status}/{action}, manifest has {item.get('status')}/{item.get('action')}")
    expected_decision = decision_for(status, action)
    if item.get("expected_decision") != expected_decision:
        failures.append(f"{path}: expected_decision = {item.get('expected_decision')}, want {expected_decision}")
    seen_decisions.append(expected_decision)

    kind = item.get("kind")
    source = item.get("source")
    if path_text.startswith("docs/fixtures/real-project-agent-loop/"):
        if kind != "real_project_agent_loop" or source != "real_project":
            failures.append(f"{path}: real project fixture must use kind=real_project_agent_loop and source=real_project")
        if has_key(payload, "raw_output"):
            failures.append(f"{path}: real project fixture must not contain raw_output")
    else:
        if kind != "validate_coverage_task" or source != "synthetic":
            failures.append(f"{path}: validate fixture must use kind=validate_coverage_task and source=synthetic")

missing = expected_paths - set(seen_paths)
extra = set(seen_paths) - expected_paths
if missing:
    failures.append("manifest missing fixtures: " + ", ".join(sorted(missing)))
if extra:
    failures.append("manifest lists unknown fixtures: " + ", ".join(sorted(extra)))
if len(seen_paths) != len(set(seen_paths)):
    failures.append("manifest fixture paths must be unique")

if failures:
    print("agent decision fixtures manifest test failed:", file=sys.stderr)
    for failure in failures:
        print(f"- {failure}", file=sys.stderr)
    raise SystemExit(1)

print(f"agent decision fixtures manifest test passed ({len(fixtures)} fixtures, decisions={','.join(seen_decisions)})")
PY
