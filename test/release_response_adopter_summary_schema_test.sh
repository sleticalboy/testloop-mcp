#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"

cd "$repo_root"

python3 - <<'PY'
from pathlib import Path
import json
import sys

schema = Path("docs/fixtures/release-response-adopter-summary.schema.json")
fixture = Path("docs/fixtures/release-response-adopter-summary/passed.json")
schema_payload = json.loads(schema.read_text(encoding="utf-8"))
fixture_payload = json.loads(fixture.read_text(encoding="utf-8"))

failures = []
required = schema_payload.get("required", [])
properties = schema_payload.get("properties", {})

for field in required:
    if field not in fixture_payload:
        failures.append(f"{fixture}: missing required field {field}")
    if field not in properties:
        failures.append(f"{schema}: required field {field} has no property schema")

extra = sorted(set(fixture_payload) - set(required))
if extra:
    failures.append(f"{fixture}: fixture has unexpected fields {extra}")

expected = {
    "schema_version": 1,
    "status": "passed",
    "release_ref": "v0.5.20",
    "fixture_count": 8,
    "agent_next_step": "ready",
    "should_accept": True,
    "npm_exit_code": 0,
    "failures": [],
}
for key, want in expected.items():
    if fixture_payload.get(key) != want:
        failures.append(f"{fixture}: {key} = {fixture_payload.get(key)!r}, want {want!r}")

for key in [
    "repo_dir",
    "readme_path",
    "workflow_path",
    "package_dir",
    "install_summary_json",
    "agent_response_json",
    "consumer_json",
]:
    if not isinstance(fixture_payload.get(key), str) or not fixture_payload[key]:
        failures.append(f"{fixture}: {key} must be a non-empty string")

if schema_payload.get("additionalProperties") is not False:
    failures.append(f"{schema}: additionalProperties must be false")

if failures:
    print("release response adopter summary schema test failed:", file=sys.stderr)
    for failure in failures:
        print(f"- {failure}", file=sys.stderr)
    sys.exit(1)

print("release response adopter summary schema test passed")
PY
