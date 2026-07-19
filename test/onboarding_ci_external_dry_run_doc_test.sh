#!/usr/bin/env sh
set -eu

python3 - <<'PY'
from pathlib import Path

doc = Path("docs/onboarding-ci-external-dry-run.md")
text = doc.read_text(encoding="utf-8")

required = [
    "scripts/showcase-onboarding-ci-external-project.sh",
    "TESTLOOP_MCP_COMMAND=/tmp/testloop-mcp",
    "TESTLOOP_EXTERNAL_ONBOARDING_PROJECT_TYPE=node",
    "TESTLOOP_EXTERNAL_ONBOARDING_PROJECT_TYPE=all",
    "pnpm install --frozen-lockfile && pnpm build",
    "external_onboarding_status=passed",
    "external_onboarding_node_status=passed",
    "/tmp/testloop-external-onboarding/artifacts/verification-report.md",
    "/tmp/testloop-external-onboarding/artifacts/verification-summary.json",
    "/tmp/testloop-external-onboarding/artifacts/agent-decision.txt",
    "/tmp/testloop-external-onboarding/artifacts/agent-response.txt",
    "external_onboarding_agent_response=/tmp/testloop-external-onboarding/artifacts/agent-response.txt",
    "external_onboarding_node_agent_response=/tmp/testloop-external-onboarding/artifacts/agent-response.txt",
    "/tmp/testloop-external-onboarding/artifacts/go/agent-response.txt",
    "/tmp/testloop-external-onboarding/artifacts/node/agent-response.txt",
    "agent_next_step=ready",
    "pnpm",
]

missing = [item for item in required if item not in text]
if missing:
    print("onboarding CI external dry-run doc test failed:")
    for item in missing:
        print(f"- missing {item}")
    raise SystemExit(1)

for path in [
    Path("scripts/showcase-onboarding-ci-external-project.sh"),
    Path("scripts/run-onboarding-ci.sh"),
]:
    if not path.exists():
        print(f"missing referenced file: {path}")
        raise SystemExit(1)

print("onboarding CI external dry-run doc test passed")
PY
