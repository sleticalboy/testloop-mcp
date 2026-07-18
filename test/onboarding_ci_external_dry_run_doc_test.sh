#!/usr/bin/env sh
set -eu

python3 - <<'PY'
from pathlib import Path

doc = Path("docs/onboarding-ci-external-dry-run.md")
text = doc.read_text(encoding="utf-8")

required = [
    "scripts/showcase-onboarding-ci-external-project.sh",
    "TESTLOOP_MCP_COMMAND=/tmp/testloop-mcp",
    "external_onboarding_status=passed",
    "/tmp/testloop-external-onboarding/artifacts/verification-report.md",
    "/tmp/testloop-external-onboarding/artifacts/verification-summary.json",
    "/tmp/testloop-external-onboarding/artifacts/agent-decision.txt",
    "agent_next_step=ready",
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
