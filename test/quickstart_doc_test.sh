#!/usr/bin/env sh
set -eu

python3 - <<'PY'
from pathlib import Path

doc = Path("docs/quickstart.md")
text = doc.read_text(encoding="utf-8")

required = [
    "brew install testloop-mcp",
    "testloop-mcp --version",
    "TESTLOOP_FIRST_RUN_EXPECT_VERSION=0.5.10",
    "scripts/doctor-first-run.sh",
    "first_run_agent_next_step=ready",
    "scripts/verify-client-setup.sh",
    "scripts/verify-mcp-process-smoke.sh",
    "testloop-mcp --print-config=codex",
    "testloop-mcp --print-config=claude",
    "testloop-mcp --print-config=cursor",
    "go run ./examples/mcp-client-demo",
    "scripts/showcase-agent-onboarding-report.sh",
    "go run ./examples/agent-response-manifest-demo",
    "docs/fixtures/agent-response-artifact-manifest.json",
    "agent-response-artifact-manifest.schema.json",
    "./adopter-verification-guide.md",
    "./mcp-client-contract-tests.md",
]

missing = [item for item in required if item not in text]
if missing:
    print("quickstart doc test failed:")
    for item in missing:
        print(f"- missing {item}")
    raise SystemExit(1)

for path in [
    Path("docs/installation.md"),
    Path("docs/first-run-diagnostics.md"),
    Path("docs/showcase-agent-loop.md"),
    Path("docs/verification-report.md"),
    Path("docs/fixtures/agent-response-artifact-manifest.json"),
    Path("docs/fixtures/agent-response-artifact-manifest.schema.json"),
    Path("docs/adopter-verification-guide.md"),
    Path("docs/mcp-client-contract-tests.md"),
    Path("examples/mcp-client-demo/main.go"),
    Path("examples/agent-response-manifest-demo/main.go"),
]:
    if not path.exists():
        print(f"missing referenced file: {path}")
        raise SystemExit(1)

print("quickstart doc test passed")
PY
