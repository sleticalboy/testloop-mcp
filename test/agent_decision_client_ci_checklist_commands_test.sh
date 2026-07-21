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
doc = Path("docs/agent-decision-client-ci-checklist.md")
text = doc.read_text(encoding="utf-8")
blocks = re.findall(r"```bash\n(.*?)\n```", text, flags=re.S)

failures = []
if len(blocks) != 3:
    failures.append(f"{doc}: expected exactly 3 bash command blocks, found {len(blocks)}")

client_dir = tmp_dir / "client"
client_dir.mkdir()
installer_path = repo_root / "scripts/install-agent-decision-client-ci-template.sh"

def run_shell(script: str, cwd: Path, env=None):
    merged_env = os.environ.copy()
    if env:
        merged_env.update(env)
    return subprocess.run(
        ["bash", "-eu", "-o", "pipefail", "-c", script],
        cwd=cwd,
        env=merged_env,
        text=True,
        capture_output=True,
    )

if not failures:
    install_block = blocks[0]
    if "curl -fsSL " not in install_block or "/absolute/path/to/client-repo" not in install_block:
        failures.append(f"{doc}: first bash block must be the installer curl command")
    else:
        local_install_block = install_block.replace(
            "curl -fsSL https://raw.githubusercontent.com/sleticalboy/testloop-mcp/main/scripts/install-agent-decision-client-ci-template.sh -o /tmp/install-testloop-agent-decision-ci.sh",
            f"cp {installer_path} /tmp/install-testloop-agent-decision-ci.sh",
        ).replace("/absolute/path/to/client-repo", str(client_dir))
        result = run_shell(local_install_block, repo_root)
        if result.returncode != 0:
            failures.append(
                "installer checklist command failed:\nSTDOUT:\n"
                + result.stdout
                + "\nSTDERR:\n"
                + result.stderr
            )

workflow = client_dir / ".github/workflows/testloop-agent-decision-contract.yml"
if not workflow.exists():
    failures.append(f"installer checklist command did not create workflow: {workflow}")
else:
    workflow_text = workflow.read_text(encoding="utf-8")
    for needle in [
        "repository: sleticalboy/testloop-mcp",
        "ref: v0.5.16",
        ".testloop-mcp/scripts/showcase-agent-decision-client-ci.sh --json",
    ]:
        if needle not in workflow_text:
            failures.append(f"generated workflow missing {needle!r}")

if not failures:
    helper = client_dir / ".testloop-mcp"
    if not helper.exists():
        helper.symlink_to(repo_root, target_is_directory=True)
    contract_block = blocks[1]
    if contract_block.strip() != ".testloop-mcp/scripts/showcase-agent-decision-client-ci.sh --json":
        failures.append(f"{doc}: second bash block must be the contract command")
    else:
        env = {
            "TESTLOOP_AGENT_DECISION_CLIENT_DIR": str(tmp_dir / "contract-client"),
        }
        result = run_shell(contract_block, client_dir, env=env)
        if result.returncode != 0:
            failures.append(
                "contract checklist command failed:\nSTDOUT:\n"
                + result.stdout
                + "\nSTDERR:\n"
                + result.stderr
            )
        else:
            try:
                payload = json.loads(result.stdout)
            except json.JSONDecodeError as exc:
                failures.append(f"contract command did not emit JSON: {exc}")
            else:
                if payload.get("status") != "passed" or payload.get("fixture_count") != 8:
                    failures.append("contract command emitted unexpected summary")

if not failures:
    showcase_block = blocks[2]
    if showcase_block.strip() != "scripts/showcase-agent-decision-client-ci-template-install.sh --json":
        failures.append(f"{doc}: third bash block must be the install showcase command")
    else:
        env = {
            "TESTLOOP_AGENT_DECISION_CI_INSTALLER_PATH": str(installer_path),
            "TESTLOOP_AGENT_DECISION_CI_CLIENT_DIR": str(tmp_dir / "showcase-client"),
            "TESTLOOP_AGENT_DECISION_CI_HELPER_DIR": str(repo_root),
        }
        result = run_shell(showcase_block, repo_root, env=env)
        if result.returncode != 0:
            failures.append(
                "install showcase checklist command failed:\nSTDOUT:\n"
                + result.stdout
                + "\nSTDERR:\n"
                + result.stderr
            )
        else:
            try:
                payload = json.loads(result.stdout)
            except json.JSONDecodeError as exc:
                failures.append(f"install showcase command did not emit JSON: {exc}")
            else:
                if payload.get("schema_version") != 1:
                    failures.append("install showcase schema_version must be 1")
                if payload.get("status") != "passed":
                    failures.append("install showcase status must be passed")
                if payload.get("fixture_count") != 8:
                    failures.append("install showcase fixture_count must be 8")

if failures:
    print("Agent decision client CI checklist command test failed:", file=sys.stderr)
    for failure in failures:
        print(f"- {failure}", file=sys.stderr)
    sys.exit(1)

print("Agent decision client CI checklist command test passed")
PY
