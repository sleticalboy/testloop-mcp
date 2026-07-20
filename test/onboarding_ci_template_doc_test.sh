#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"

cd "$repo_root"

python3 - <<'PY'
from pathlib import Path
import sys

doc = Path("docs/onboarding-ci-template.md")
text = doc.read_text(encoding="utf-8")

required_snippets = [
    "name: testloop onboarding",
    "name: testloop web onboarding",
    "curl -fsSL https://raw.githubusercontent.com/sleticalboy/testloop-mcp/main/scripts/run-onboarding-ci.sh -o /tmp/testloop-onboarding-ci.sh",
    "TESTLOOP_MCP_VERSION=v0.5.15",
    "sh scripts/verify-agent-artifact.sh onboarding /tmp/testloop-onboarding",
    "Artifact verification",
    "TESTLOOP_ONBOARDING_OUTPUT_DIR=/tmp/testloop-onboarding",
    "TESTLOOP_ONBOARDING_OUTPUT_DIR=/tmp/testloop-web-onboarding",
    "TESTLOOP_ONBOARDING_PROJECT_DIR=\"$PWD\"",
    "bash /tmp/testloop-onboarding-ci.sh 'go test ./...'",
    "bash /tmp/testloop-onboarding-ci.sh 'pnpm install --frozen-lockfile && pnpm build'",
    "actions/upload-artifact@v4",
    "if: always()",
    "/tmp/testloop-onboarding/verification-report.md",
    "/tmp/testloop-onboarding/verification-summary.json",
    "/tmp/testloop-onboarding/verification-summary.schema.json",
    "/tmp/testloop-onboarding/agent-decision.txt",
    "/tmp/testloop-onboarding/agent-response.txt",
    "/tmp/testloop-web-onboarding/verification-report.md",
    "/tmp/testloop-web-onboarding/verification-summary.json",
    "/tmp/testloop-web-onboarding/verification-summary.schema.json",
    "/tmp/testloop-web-onboarding/agent-decision.txt",
    "/tmp/testloop-web-onboarding/agent-response.txt",
    "Agent 四段回复草稿",
    "agent_next_step=ready",
    "agent_artifact_status=passed",
    "./verification-ci.md",
    "./real-integration-cases.md",
]

failures = []
for snippet in required_snippets:
    if snippet not in text:
        failures.append(f"{doc}: missing required snippet {snippet!r}")

linked_paths = [
    Path("scripts/install.sh"),
    Path("scripts/run-onboarding-ci.sh"),
    Path("scripts/verify-agent-artifact.sh"),
    Path("scripts/showcase-agent-onboarding-report.sh"),
    Path("docs/verification-ci.md"),
    Path("docs/real-integration-cases.md"),
]
for path in linked_paths:
    if not path.exists():
        failures.append(f"{doc}: referenced path does not exist: {path}")

if failures:
    print("onboarding CI template doc test failed:", file=sys.stderr)
    for failure in failures:
        print(f"- {failure}", file=sys.stderr)
    sys.exit(1)

print("onboarding CI template doc test passed")
PY
