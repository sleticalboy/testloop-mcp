#!/usr/bin/env sh
set -eu

python3 - <<'PY'
from pathlib import Path

doc = Path("docs/first-run-ci-external-dry-run.md")
text = doc.read_text(encoding="utf-8")

required = [
    "scripts/showcase-first-run-ci-external-project.sh",
    "scripts/run-first-run-ci.sh",
    "TESTLOOP_MCP_COMMAND=/tmp/testloop-mcp",
    "TESTLOOP_EXTERNAL_FIRST_RUN_PROJECT_TYPE=node",
    "TESTLOOP_EXTERNAL_FIRST_RUN_PROJECT_TYPE=all",
    "pnpm install --frozen-lockfile && pnpm build",
    "external_first_run_status=passed",
    "external_first_run_node_status=passed",
    "/tmp/testloop-external-first-run/artifacts/verification-report.md",
    "/tmp/testloop-external-first-run/artifacts/verification-summary.json",
    "/tmp/testloop-external-first-run/artifacts/agent-decision.txt",
    "/tmp/testloop-external-first-run/artifacts/first-run-context.txt",
    "/tmp/testloop-external-first-run/artifacts/first-run.log",
    "agent_next_step=ready",
    "first_run_agent_next_step=ready",
    "pnpm",
]

missing = [item for item in required if item not in text]
if missing:
    print("first-run CI external dry-run doc test failed:")
    for item in missing:
        print(f"- missing {item}")
    raise SystemExit(1)

for path in [
    Path("scripts/showcase-first-run-ci-external-project.sh"),
    Path("scripts/run-first-run-ci.sh"),
]:
    if not path.exists():
        print(f"missing referenced file: {path}")
        raise SystemExit(1)

print("first-run CI external dry-run doc test passed")
PY
