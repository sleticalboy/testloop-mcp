#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT INT TERM

cd "$repo_root"

python3 - "$tmp_dir" <<'PY'
from pathlib import Path
import json
import subprocess
import sys

tmp_dir = Path(sys.argv[1])
summary_sample = Path("docs/fixtures/agent-decision-client-ci-summary/passed.json")
schema_path = Path("docs/fixtures/agent-decision-client-ci-response.schema.json")
fixtures_dir = Path("docs/fixtures/agent-decision-client-ci-response")
renderer = ["node", "scripts/render-agent-decision-client-ci-response.mjs", "--json"]
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
failures = []

schema = json.loads(schema_path.read_text(encoding="utf-8"))
if schema.get("additionalProperties") is not False:
    failures.append("schema must reject additional properties")

required = set(schema.get("required", []))
properties = set(schema.get("properties", {}))
if not required:
    failures.append("schema must define required fields")
if not required <= properties:
    failures.append("schema required fields must be declared in properties")


def validate_shape(payload, label):
    missing = sorted(required - set(payload))
    if missing:
        failures.append(f"{label}: missing required fields {missing}")
    extra = sorted(set(payload) - properties)
    if extra:
        failures.append(f"{label}: extra fields {extra}")
    evidence = payload.get("evidence")
    if not isinstance(evidence, dict):
        failures.append(f"{label}: evidence must be an object")
        return
    for key in [
        "client_dir",
        "fixture_dir",
        "result_json",
        "result_schema",
        "fixture_count",
        "decisions",
        "validator_exit_code",
    ]:
        if key not in evidence:
            failures.append(f"{label}: evidence missing {key}")
    if not isinstance(payload.get("failures"), list):
        failures.append(f"{label}: failures must be an array")


def comparable(payload):
    cloned = json.loads(json.dumps(payload))
    cloned.pop("summary_json", None)
    return cloned


def write_summary(name, mutate):
    payload = json.loads(summary_sample.read_text(encoding="utf-8"))
    mutate(payload)
    path = tmp_dir / f"{name}.json"
    path.write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")
    return path


def render(path):
    completed = subprocess.run(
        [*renderer, str(path)],
        cwd=Path.cwd(),
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=False,
    )
    if completed.stdout.strip() == "":
        failures.append(f"renderer produced empty stdout for {path}")
        return {}
    payload = json.loads(completed.stdout)
    if (payload.get("status") == "passed") != (completed.returncode == 0):
        failures.append(f"renderer exit code mismatch for {path}: {completed.returncode}")
    return payload


cases = {
    "passed": summary_sample,
    "validator-failed": write_summary(
        "validator-failed",
        lambda payload: (
            payload.__setitem__("status", "failed"),
            payload.__setitem__("validator_exit_code", 1),
            payload.__setitem__("failures", ["npm test failed"]),
        ),
    ),
    "fixture-drift": write_summary(
        "fixture-drift",
        lambda payload: (
            payload.__setitem__("fixture_count", 7),
            payload.__setitem__("decisions", expected_decisions[:-1]),
        ),
    ),
}

for name, summary_path in cases.items():
    fixture_path = fixtures_dir / f"{name}.json"
    fixture = json.loads(fixture_path.read_text(encoding="utf-8"))
    generated = render(summary_path)
    validate_shape(fixture, f"fixture {name}")
    validate_shape(generated, f"generated {name}")
    if comparable(generated) != comparable(fixture):
        failures.append(
            f"{name}: generated response does not match fixture\n"
            f"generated={json.dumps(comparable(generated), ensure_ascii=False, sort_keys=True)}\n"
            f"fixture={json.dumps(comparable(fixture), ensure_ascii=False, sort_keys=True)}"
        )

passed = json.loads((fixtures_dir / "passed.json").read_text(encoding="utf-8"))
if passed["evidence"]["decisions"] != expected_decisions:
    failures.append("passed fixture decisions drifted")
if passed["evidence"]["validator_exit_code"] != 0:
    failures.append("passed fixture validator_exit_code must be 0")

if failures:
    print("Agent decision client CI response fixtures test failed:", file=sys.stderr)
    for failure in failures:
        print(f"- {failure}", file=sys.stderr)
    sys.exit(1)

print("Agent decision client CI response fixtures test passed")
PY
