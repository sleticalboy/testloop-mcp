#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
manifest="${repo_root}/docs/fixtures/agent-response-artifact-manifest.json"

cd "$repo_root"

python3 - "$manifest" <<'PY'
import json
import sys
from pathlib import Path

manifest_path = Path(sys.argv[1])
data = json.loads(manifest_path.read_text(encoding="utf-8"))
schema_path = manifest_path.with_name("agent-response-artifact-manifest.schema.json")
index_text = Path("docs/fixtures.md").read_text(encoding="utf-8")

failures = []
if data.get("$schema") != "./agent-response-artifact-manifest.schema.json":
    failures.append("$schema must point to ./agent-response-artifact-manifest.schema.json")
if not schema_path.is_file():
    failures.append(f"missing manifest schema file: {schema_path}")
for snippet in (
    "agent-response-artifact-manifest.schema.json",
    "tools/agent_response_artifact_manifest_schema_test.go",
    "sh test/agent_response_manifest_demo_test.sh",
    "go test ./tools -run TestAgentResponseArtifactManifestSchema -count=1",
):
    if snippet not in index_text:
        failures.append(f"docs/fixtures.md missing maintenance snippet: {snippet}")

if data.get("schema_version") != 1:
    failures.append("schema_version must be 1")
if data.get("summary_schema") != "./verification-summary.schema.json":
    failures.append("summary_schema must point to ./verification-summary.schema.json")
summary_schema_path = manifest_path.with_name("verification-summary.schema.json")
if not summary_schema_path.is_file():
    failures.append(f"missing verification summary schema file: {summary_schema_path}")

artifacts = data.get("artifacts")
if not isinstance(artifacts, list) or not artifacts:
    failures.append("artifacts must be a non-empty list")
else:
    kinds = {artifact.get("kind") for artifact in artifacts}
    if kinds != {"first-run", "onboarding"}:
        failures.append(f"unexpected artifact kinds: {sorted(kinds)!r}")

for artifact in artifacts or []:
    kind = artifact.get("kind", "<missing>")
    directory = Path(artifact.get("directory", ""))
    if not directory.is_dir():
        failures.append(f"{kind}: missing directory {directory}")
        continue

    for key in ("agent_response", "decision", "summary", "report"):
        rel = artifact.get(key)
        if not rel:
            failures.append(f"{kind}: missing manifest key {key}")
            continue
        path = directory / rel
        if not path.is_file():
            failures.append(f"{kind}: missing file for {key}: {path}")

    for key in ("optional_context", "optional_log"):
        rel = artifact.get(key)
        if rel and not (directory / rel).is_file():
            failures.append(f"{kind}: missing optional file listed in manifest: {directory / rel}")

    response_path = directory / artifact.get("agent_response", "")
    decision_path = directory / artifact.get("decision", "")
    summary_path = directory / artifact.get("summary", "")
    if not response_path.is_file() or not decision_path.is_file() or not summary_path.is_file():
        continue

    response = response_path.read_text(encoding="utf-8")
    decision = decision_path.read_text(encoding="utf-8")
    summary = json.loads(summary_path.read_text(encoding="utf-8"))

    action_field = artifact.get("action_field")
    expected_action = artifact.get("expected_action")
    if f"{action_field}={expected_action}" not in response:
        failures.append(f"{kind}: response missing {action_field}={expected_action}")
    if "agent_next_step=inspect-user-project" not in decision:
        failures.append(f"{kind}: decision missing agent_next_step=inspect-user-project")
    if summary.get("overall_status") != "failed":
        failures.append(f"{kind}: summary overall_status must be failed")
    if summary.get("failed_count") != 1:
        failures.append(f"{kind}: summary failed_count must be 1")

    failed_sections = [
        section for section in summary.get("sections", [])
        if section.get("status") == "failed"
    ]
    if len(failed_sections) != 1:
        failures.append(f"{kind}: expected exactly one failed section")
    else:
        failed = failed_sections[0]
        if failed.get("name") != artifact.get("expected_failed_section"):
            failures.append(f"{kind}: failed section mismatch: {failed.get('name')!r}")
        if failed.get("exit_code") != artifact.get("expected_exit_code"):
            failures.append(f"{kind}: exit_code mismatch: {failed.get('exit_code')!r}")

    for field in artifact.get("required_response_fields", []):
        if f"- {field}=" not in response:
            failures.append(f"{kind}: response missing field {field}")

    fallback = artifact.get("fallback_order")
    if not isinstance(fallback, list) or not fallback or fallback[0] != "agent-response.txt":
        failures.append(f"{kind}: fallback_order must start with agent-response.txt")
    for rel in fallback or []:
        if not (directory / rel).is_file():
            failures.append(f"{kind}: fallback_order references missing file {rel}")

if failures:
    print("agent response artifact manifest test failed:", file=sys.stderr)
    for failure in failures:
        print(f"- {failure}", file=sys.stderr)
    raise SystemExit(1)

print(f"agent response artifact manifest test passed ({len(artifacts)} artifacts)")
PY
