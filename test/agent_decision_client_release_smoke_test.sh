#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT INT TERM

cd "$repo_root"

summary_json="${tmp_dir}/release-smoke-summary.json"
TESTLOOP_AGENT_DECISION_RELEASE_INSTALLER_URL="file://${repo_root}/scripts/install-agent-decision-client-ci-template.sh" \
  scripts/showcase-agent-decision-client-release-smoke.sh --json > "$summary_json"

python3 - \
  "$summary_json" \
  docs/fixtures/agent-decision-client-release-smoke-summary.schema.json \
  docs/fixtures/agent-decision-client-release-smoke-summary/passed.json <<'PY'
from pathlib import Path
import json
import sys

summary_path = Path(sys.argv[1])
schema_path = Path(sys.argv[2])
sample_path = Path(sys.argv[3])
summary = json.loads(summary_path.read_text(encoding="utf-8"))
schema = json.loads(schema_path.read_text(encoding="utf-8"))
sample = json.loads(sample_path.read_text(encoding="utf-8"))
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
    if payload.get("schema_version") != 1:
        failures.append(f"{label} schema_version must be 1")
    if payload.get("status") != "passed":
        failures.append(f"{label} status must be passed")
    if payload.get("release_ref") != "v0.5.21":
        failures.append(f"{label} release_ref must be v0.5.21")
    installer_url = payload.get("installer_url")
    if label == "fixture sample":
        if installer_url != "https://raw.githubusercontent.com/sleticalboy/testloop-mcp/v0.5.21/scripts/install-agent-decision-client-ci-template.sh":
            failures.append(f"{label} installer_url must point at v0.5.21 raw installer")
    elif not isinstance(installer_url, str) or not installer_url.startswith("file://"):
        failures.append(f"{label} installer_url must point at local file URL in regression test")
    if payload.get("helper_refs") != {"install": "v0.5.21", "consumer": "v0.5.21"}:
        failures.append(f"{label} helper_refs drifted")
    if payload.get("fixture_count") != 8:
        failures.append(f"{label} fixture_count must be 8")
    if payload.get("decisions") != expected_decisions:
        failures.append(f"{label} decisions drifted")
    if payload.get("agent_next_steps") != {"client": "ready", "consumer": "ready"}:
        failures.append(f"{label} agent_next_steps drifted")
    if payload.get("failures") != []:
        failures.append(f"{label} failures must be empty")
    for key in (
        "install_summary_json",
        "client_summary_json",
        "client_response_json",
        "consumer_summary_json",
        "consumer_agent_response_json",
    ):
        if not isinstance(payload.get(key), str) or not payload[key]:
            failures.append(f"{label} {key} must be a non-empty string")

validate_payload(summary, "generated summary")
validate_payload(sample, "fixture sample")

for key in (
    "install_summary_json",
    "client_summary_json",
    "client_response_json",
    "consumer_summary_json",
    "consumer_agent_response_json",
):
    if not Path(summary[key]).exists():
        failures.append(f"generated summary {key} does not exist: {summary[key]}")

client_response = json.loads(Path(summary["client_response_json"]).read_text(encoding="utf-8"))
consumer_response = json.loads(Path(summary["consumer_agent_response_json"]).read_text(encoding="utf-8"))
if client_response.get("agent_next_step") != "ready":
    failures.append("client response agent_next_step must be ready")
if consumer_response.get("agent_next_step") != "ready":
    failures.append("consumer response agent_next_step must be ready")

if failures:
    print("Agent decision client release smoke test failed:", file=sys.stderr)
    for failure in failures:
        print(f"- {failure}", file=sys.stderr)
    sys.exit(1)

print("Agent decision client release smoke test passed")
PY
