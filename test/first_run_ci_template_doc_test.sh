#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
cd "$repo_root"

python3 - <<'PY'
from pathlib import Path
import sys

doc = Path("docs/first-run-ci-template.md")
text = doc.read_text(encoding="utf-8")

required = [
    "name: testloop first run",
    "name: testloop web first run",
    "curl -fsSL https://raw.githubusercontent.com/sleticalboy/testloop-mcp/main/scripts/run-first-run-ci.sh -o /tmp/testloop-first-run-ci.sh",
    "TESTLOOP_MCP_VERSION=v0.5.15",
    "TESTLOOP_MCP_REPO_REF",
    "helper checkout 默认使用 `main`",
    "sh scripts/verify-agent-artifact.sh first-run /tmp/testloop-first-run",
    "Artifact verification",
    "agent_artifact_status=passed",
    "TESTLOOP_FIRST_RUN_OUTPUT_DIR=/tmp/testloop-first-run",
    "TESTLOOP_FIRST_RUN_OUTPUT_DIR=/tmp/testloop-web-first-run",
    "TESTLOOP_FIRST_RUN_PROJECT_DIR=\"$PWD\"",
    "bash /tmp/testloop-first-run-ci.sh 'go test ./...'",
    "bash /tmp/testloop-first-run-ci.sh 'pnpm install --frozen-lockfile && pnpm build'",
    "actions/upload-artifact@v4",
    "if: always()",
    "/tmp/testloop-first-run/verification-report.md",
    "/tmp/testloop-first-run/verification-summary.json",
    "/tmp/testloop-first-run/verification-summary.schema.json",
    "/tmp/testloop-first-run/agent-decision.txt",
    "/tmp/testloop-first-run/first-run-context.txt",
    "/tmp/testloop-first-run/agent-response.txt",
    "/tmp/testloop-first-run/first-run.log",
    "/tmp/testloop-web-first-run/verification-report.md",
    "/tmp/testloop-web-first-run/verification-summary.json",
    "/tmp/testloop-web-first-run/verification-summary.schema.json",
    "/tmp/testloop-web-first-run/agent-decision.txt",
    "/tmp/testloop-web-first-run/first-run-context.txt",
    "/tmp/testloop-web-first-run/agent-response.txt",
    "/tmp/testloop-web-first-run/first-run.log",
    "Agent 四段回复草稿",
    "./first-run-failures.md",
    "./onboarding-ci-template.md",
]

failures = [f"{doc}: missing required snippet {item!r}" for item in required if item not in text]
for path in [
    Path("scripts/run-first-run-ci.sh"),
    Path("scripts/doctor-first-run.sh"),
    Path("scripts/verify-agent-artifact.sh"),
    Path("docs/first-run-failures.md"),
    Path("docs/onboarding-ci-template.md"),
]:
    if not path.exists():
        failures.append(f"{doc}: referenced path does not exist: {path}")

if failures:
    print("first-run CI template doc test failed:", file=sys.stderr)
    for failure in failures:
        print(f"- {failure}", file=sys.stderr)
    sys.exit(1)

print("first-run CI template doc test passed")
PY
