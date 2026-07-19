#!/usr/bin/env sh
set -eu

python3 - <<'PY'
from pathlib import Path

doc = Path("docs/adopter-verification-guide.md")
text = doc.read_text(encoding="utf-8")

required = [
    "brew install testloop-mcp",
    "testloop-mcp --version",
    "TESTLOOP_FIRST_RUN_EXPECT_VERSION=0.5.12",
    "scripts/doctor-first-run.sh",
    "first_run_agent_next_step=ready",
    "scripts/run-first-run-ci.sh",
    "scripts/run-onboarding-ci.sh",
    "TESTLOOP_MCP_VERSION=v0.5.12",
    "PATH",
    "bootstrap 通过但本机 `PATH` 仍是旧版本",
    "pnpm install --frozen-lockfile && pnpm build",
    "verification-report.md",
    "verification-summary.json",
    "agent-decision.txt",
    "first-run-context.txt",
    "agent-response.txt",
    "first-run.log",
    "if: always()",
    "go run ./examples/agent-response-manifest-demo",
    "docs/fixtures/agent-response-artifact-manifest.json",
    "agent-response-artifact-manifest.schema.json",
    "fallback_order",
    "agent_next_step=ready",
    "fix-installation",
    "inspect-mcp-transport",
    "inspect-agent-demo",
    "inspect-user-project",
    "scripts/showcase-first-run-ci-external-project.sh",
    "scripts/showcase-onboarding-ci-external-project.sh",
    "TESTLOOP_EXTERNAL_FIRST_RUN_PROJECT_TYPE=all",
    "TESTLOOP_EXTERNAL_ONBOARDING_PROJECT_TYPE=all",
]

missing = [item for item in required if item not in text]
if missing:
    print("adopter verification guide doc test failed:")
    for item in missing:
        print(f"- missing {item}")
    raise SystemExit(1)

for path in [
    Path("docs/quickstart.md"),
    Path("docs/verification-ci.md"),
    Path("docs/first-run-diagnostics.md"),
    Path("docs/first-run-ci-template.md"),
    Path("docs/onboarding-ci-template.md"),
    Path("docs/mcp-client-contract-tests.md"),
    Path("docs/fixtures.md"),
    Path("docs/fixtures/agent-response-artifact-manifest.json"),
    Path("docs/fixtures/agent-response-artifact-manifest.schema.json"),
]:
    if not path.exists():
        print(f"missing referenced file: {path}")
        raise SystemExit(1)

print("adopter verification guide doc test passed")
PY
