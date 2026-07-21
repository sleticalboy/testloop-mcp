#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT INT TERM

cd "$repo_root"

summary_json="${tmp_dir}/consumer-smoke-summary.json"
scripts/showcase-agent-decision-client-consumer-smoke.sh --json > "$summary_json"

python3 - \
  "$summary_json" \
  docs/fixtures/agent-decision-client-consumer-smoke-summary.schema.json \
  docs/fixtures/agent-decision-client-consumer-smoke-summary/passed.json \
  docs/fixtures/agent-decision-client-consumer-smoke-summary/validator-failed.json \
  docs/fixtures/agent-decision-client-consumer-smoke-summary/fixture-drift.json <<'PY'
from pathlib import Path
import json
import sys

summary_path = Path(sys.argv[1])
schema_path = Path(sys.argv[2])
sample_path = Path(sys.argv[3])
failed_sample_paths = [Path(arg) for arg in sys.argv[4:]]
summary = json.loads(summary_path.read_text(encoding="utf-8"))
schema = json.loads(schema_path.read_text(encoding="utf-8"))
sample = json.loads(sample_path.read_text(encoding="utf-8"))
failed_samples = [(path, json.loads(path.read_text(encoding="utf-8"))) for path in failed_sample_paths]

required = schema["required"]
properties = schema["properties"]
failures = []
expected_decisions = [
    "accept",
    "accept",
    "accept",
    "manual-review",
    "manual-review",
    "manual-review",
    "apply-repair",
    "needs-better-input",
]

if schema.get("additionalProperties") is not False:
    failures.append("schema must reject additional properties")

for key in required:
    if key not in properties:
        failures.append(f"schema required field is not declared in properties: {key}")

def validate_payload(payload, label):
    missing = [key for key in required if key not in payload]
    if missing:
        failures.append(f"{label} missing required fields: " + ", ".join(missing))

    extra = sorted(set(payload) - set(properties))
    if extra:
        failures.append(f"{label} has fields not declared by schema: " + ", ".join(extra))

    expected_exact = {
        "schema_version": 1,
        "status": "passed",
        "helper_ref": "v0.5.18",
        "fixture_count": 8,
        "install_summary_validator_exit_code": 0,
        "fixture_validator_exit_code": 0,
        "npm_validator_exit_code": 0,
    }
    for key, want in expected_exact.items():
        if payload.get(key) != want:
            failures.append(f"{label} {key} = {payload.get(key)!r}, want {want!r}")

    if payload.get("decisions") != expected_decisions:
        failures.append(f"{label} unexpected decisions sequence")
    if payload.get("failures") != []:
        failures.append(f"{label} failures must be empty on passed smoke")
    for key in (
        "client_dir",
        "workflow_path",
        "install_summary_json",
        "install_summary_validator_json",
        "client_summary_json",
        "fixture_dir",
        "fixture_validation_json",
        "result_json",
        "agent_response_json",
    ):
        if not isinstance(payload.get(key), str) or not payload[key]:
            failures.append(f"{label} {key} must be a non-empty string")

validate_payload(summary, "generated summary")
validate_payload(sample, "fixture sample")

for failed_sample_path, failed_sample in failed_samples:
    label = f"failed fixture sample {failed_sample_path.name}"
    missing = [key for key in required if key not in failed_sample]
    if missing:
        failures.append(f"{label} missing required fields: " + ", ".join(missing))
    extra = sorted(set(failed_sample) - set(properties))
    if extra:
        failures.append(f"{label} has fields not declared by schema: " + ", ".join(extra))
    if failed_sample.get("schema_version") != 1:
        failures.append(f"{label} schema_version must be 1")
    if failed_sample.get("status") != "failed":
        failures.append(f"{label} status must be failed")
    if not isinstance(failed_sample.get("failures"), list) or not failed_sample["failures"]:
        failures.append(f"{label} failures must be a non-empty array")
    for key in (
        "client_dir",
        "workflow_path",
        "install_summary_json",
        "install_summary_validator_json",
        "client_summary_json",
        "fixture_dir",
        "fixture_validation_json",
        "result_json",
        "agent_response_json",
    ):
        if not isinstance(failed_sample.get(key), str) or not failed_sample[key]:
            failures.append(f"{label} {key} must be a non-empty string")

for key in (
    "workflow_path",
    "install_summary_json",
    "install_summary_validator_json",
    "client_summary_json",
    "fixture_dir",
    "fixture_validation_json",
    "result_json",
    "agent_response_json",
):
    if not Path(summary[key]).exists():
        failures.append(f"generated summary {key} does not exist: {summary[key]}")

install_summary = json.loads(Path(summary["install_summary_json"]).read_text(encoding="utf-8"))
client_summary = json.loads(Path(summary["client_summary_json"]).read_text(encoding="utf-8"))
result_payload = json.loads(Path(summary["result_json"]).read_text(encoding="utf-8"))
agent_response = json.loads(Path(summary["agent_response_json"]).read_text(encoding="utf-8"))
if install_summary.get("decisions") != expected_decisions:
    failures.append("generated install summary decisions drifted")
if client_summary.get("decisions") != expected_decisions:
    failures.append("generated client summary decisions drifted")
if result_payload.get("decisions") != expected_decisions:
    failures.append("generated result JSON decisions drifted")
if agent_response.get("agent_next_step") != "ready":
    failures.append("generated agent response did not resolve agent_next_step=ready")

if failures:
    print("Agent decision client consumer smoke summary schema test failed:", file=sys.stderr)
    for failure in failures:
        print(f"- {failure}", file=sys.stderr)
    sys.exit(1)

print("Agent decision client consumer smoke summary schema test passed")
PY
