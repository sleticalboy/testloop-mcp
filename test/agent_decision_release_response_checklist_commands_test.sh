#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT INT TERM

cd "$repo_root"

python3 - "$tmp_dir" <<'PY'
from pathlib import Path
import json
import os
import re
import subprocess
import sys

tmp_dir = Path(sys.argv[1])
repo_root = Path.cwd()
doc = Path("docs/agent-decision-release-response-checklist.md")
text = doc.read_text(encoding="utf-8")
blocks = re.findall(r"```bash\n(.*?)\n```", text, flags=re.S)

failures = []
if len(blocks) != 9:
    failures.append(f"{doc}: expected exactly 9 bash command blocks, found {len(blocks)}")

client_dir = tmp_dir / "client"
client_dir.mkdir()
install_summary = tmp_dir / "install-summary.json"
sample_summary = repo_root / "docs/fixtures/agent-decision-client-release-smoke-summary/passed.json"

def run_shell(script: str, cwd: Path):
    return subprocess.run(
        ["bash", "-eu", "-o", "pipefail", "-c", script],
        cwd=cwd,
        env=os.environ.copy(),
        text=True,
        capture_output=True,
    )

def fail_with_output(label: str, result):
    failures.append(
        f"{label} failed:\nSTDOUT:\n{result.stdout}\nSTDERR:\n{result.stderr}"
    )

if not failures:
    release_smoke_block = blocks[0].strip()
    if release_smoke_block != "scripts/showcase-agent-decision-client-release-smoke.sh --json > /tmp/testloop-release-smoke-summary.json":
        failures.append(f"{doc}: first bash block must generate release smoke summary")

if not failures:
    install_block = blocks[1]
    if "scripts/install-agent-decision-release-response-client.sh" not in install_block:
        failures.append(f"{doc}: second bash block must install release response client")
    else:
        local_install = (
            install_block
            .replace("/tmp/testloop-release-smoke-summary.json", str(sample_summary))
            .replace("/absolute/path/to/client-repo", str(client_dir))
            .replace("/tmp/testloop-release-response-install-summary.json", str(install_summary))
        )
        result = run_shell(local_install, repo_root)
        if result.returncode != 0:
            fail_with_output("release response install checklist command", result)

workflow = client_dir / ".github/workflows/testloop-release-response-contract.yml"
package_dir = client_dir / "testloop-release-response-client"
response_json = package_dir / "testloop-release-response.json"
if not workflow.exists():
    failures.append(f"installer did not create workflow: {workflow}")
if not response_json.exists():
    failures.append(f"installer did not create agent response json: {response_json}")
if not install_summary.exists():
    failures.append(f"installer did not create install summary: {install_summary}")

if not failures:
    payload = json.loads(install_summary.read_text(encoding="utf-8"))
    if payload.get("status") != "written":
        failures.append("install summary status must be written")
    if payload.get("release_ref") != "v0.5.21":
        failures.append("install summary release_ref must be v0.5.21")
    if payload.get("fixture_count") != 8:
        failures.append("install summary fixture_count must be 8")
    if payload.get("agent_next_step") != "ready":
        failures.append("install summary agent_next_step must be ready")
    if payload.get("npm_exit_code") != 0:
        failures.append("install summary npm_exit_code must be 0")
    if payload.get("failures") != []:
        failures.append("install summary failures must be empty")

if not failures:
    validator_block = blocks[2].strip().replace(
        "/tmp/testloop-release-response-install-summary.json",
        str(install_summary),
    )
    result = run_shell(validator_block, repo_root)
    if result.returncode != 0:
        fail_with_output("release response install summary validator checklist command", result)
    elif "agent_decision_release_response_client_install_summary_status=passed" not in result.stdout:
        failures.append("validator command did not emit passed status")

if not failures:
    local_npm_block = blocks[3].replace("/absolute/path/to/client-repo", str(client_dir))
    result = run_shell(local_npm_block, repo_root)
    if result.returncode != 0:
        fail_with_output("release response package npm checklist command", result)

if not failures:
    workflow_npm_block = blocks[4]
    if workflow_npm_block.strip() != "cd testloop-release-response-client\nnpm test --silent":
        failures.append(f"{doc}: fifth bash block must mirror workflow npm command")
    else:
        result = run_shell(workflow_npm_block, client_dir)
        if result.returncode != 0:
            fail_with_output("release response workflow npm checklist command", result)

if not failures:
    if blocks[5].strip() != "scripts/showcase-agent-decision-client-release-response-ci.sh --json":
        failures.append(f"{doc}: sixth bash block must be release response CI showcase")
    if blocks[6].strip() != "scripts/showcase-release-response-adopter.sh --json":
        failures.append(f"{doc}: seventh bash block must be release response adopter showcase")
    if blocks[7].strip() != "node scripts/export-agent-decision-release-response-client.mjs /tmp/testloop-release-response-client":
        failures.append(f"{doc}: eighth bash block must export release response client")
    if blocks[8].strip() != "scripts/verify-release-candidate.sh v0.5.21":
        failures.append(f"{doc}: ninth bash block must be release readiness command")

if failures:
    print("Agent decision release response checklist command test failed:", file=sys.stderr)
    for failure in failures:
        print(f"- {failure}", file=sys.stderr)
    sys.exit(1)

print("Agent decision release response checklist command test passed")
PY
