#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"

cd "$repo_root"

python3 - <<'PY'
from pathlib import Path
import sys

doc = Path("docs/agent-decision-client-ci-template.md")
text = doc.read_text(encoding="utf-8")

required = [
    "name: testloop agent decision contract",
    ".github/workflows/testloop-agent-decision-contract.yml",
    "scripts/install-agent-decision-client-ci-template.sh /absolute/path/to/client-repo",
    "scripts/install-agent-decision-client-ci-template.sh --version v0.5.18 /absolute/path/to/client-repo",
    "scripts/install-agent-decision-client-ci-template.sh --dry-run /absolute/path/to/client-repo",
    "https://raw.githubusercontent.com/sleticalboy/testloop-mcp/main/scripts/install-agent-decision-client-ci-template.sh",
    "bash /tmp/install-testloop-agent-decision-ci.sh /absolute/path/to/client-repo",
    "scripts/showcase-agent-decision-client-ci-template-install.sh --json",
    "node scripts/validate-agent-decision-client-ci-install-summary.mjs /path/to/install-summary.json",
    "scripts/showcase-agent-decision-client-consumer-smoke.sh --json",
    "node scripts/validate-agent-decision-client-consumer-smoke-summary.mjs /path/to/consumer-smoke-summary.json",
    "node scripts/render-agent-decision-client-consumer-response.mjs /tmp/testloop-agent-decision-consumer-smoke-summary.json",
    "agent_response_json",
    "docs/fixtures/agent-decision-client-consumer-smoke-summary/validator-failed.json",
    "docs/fixtures/agent-decision-client-consumer-smoke-summary/fixture-drift.json",
    "agent_next_step=ready",
    "agent_next_step=inspect-consumer-smoke-validator",
    "agent_next_step=inspect-agent-decision-fixtures",
    "./fixtures/agent-decision-client-ci-template-install-summary.schema.json",
    "./fixtures/agent-decision-client-ci-template-install-summary/passed.json",
    "./fixtures/agent-decision-client-consumer-smoke-summary.schema.json",
    "./fixtures/agent-decision-client-consumer-smoke-summary/passed.json",
    "--force",
    "actions/checkout@v4",
    "actions/setup-node@v4",
    "repository: sleticalboy/testloop-mcp",
    "ref: v0.5.18",
    "path: .testloop-mcp",
    "scripts/showcase-agent-decision-client-ci.sh",
    "scripts/showcase-agent-decision-client-ci.sh --json",
    "scripts/render-agent-decision-client-ci-response.mjs",
    "tee /tmp/testloop-agent-decision-client-summary.json",
    "tee /tmp/testloop-agent-decision-client-response.json",
    "TESTLOOP_AGENT_DECISION_CLIENT_DIR=/tmp/testloop-agent-decision-client",
    "actions/upload-artifact@v4",
    "if: always()",
    "/tmp/testloop-agent-decision-client-summary.json",
    "/tmp/testloop-agent-decision-client-response.json",
    "/tmp/testloop-agent-decision-client/agent-decision-fixtures-result.json",
    "/tmp/testloop-agent-decision-client/testloop-agent-decision-fixtures/package.json",
    "/tmp/testloop-agent-decision-client/testloop-agent-decision-fixtures/docs/fixtures/agent-decision-fixtures.json",
    "sh test/agent_decision_client_ci_template_dry_run_test.sh",
    ".testloop-mcp/scripts/showcase-agent-decision-client-ci.sh --json | tee",
    "testloop-agent-decision-client-summary.json",
    '"status": "passed"',
    '"fixture_count": 8',
    '"decisions": ["accept", "accept", "accept", "manual-review", "manual-review", "manual-review", "apply-repair", "needs-better-input"]',
    "validator_exit_code",
    "failures[]",
    "./client-integration.md",
    "./mcp-client-contract-tests.md",
]

failures = [f"{doc}: missing required snippet {item!r}" for item in required if item not in text]
for path in [
    Path("scripts/showcase-agent-decision-client-ci.sh"),
    Path("scripts/render-agent-decision-client-ci-response.mjs"),
    Path("scripts/install-agent-decision-client-ci-template.sh"),
    Path("scripts/showcase-agent-decision-client-ci-template-install.sh"),
    Path("scripts/validate-agent-decision-client-ci-install-summary.mjs"),
    Path("scripts/showcase-agent-decision-client-consumer-smoke.sh"),
    Path("scripts/validate-agent-decision-client-consumer-smoke-summary.mjs"),
    Path("scripts/render-agent-decision-client-consumer-response.mjs"),
    Path("docs/fixtures/agent-decision-client-consumer-smoke-summary/validator-failed.json"),
    Path("docs/fixtures/agent-decision-client-consumer-smoke-summary/fixture-drift.json"),
    Path("docs/fixtures/agent-decision-client-ci-template-install-summary.schema.json"),
    Path("docs/fixtures/agent-decision-client-ci-template-install-summary/passed.json"),
    Path("docs/fixtures/agent-decision-client-consumer-smoke-summary.schema.json"),
    Path("docs/fixtures/agent-decision-client-consumer-smoke-summary/passed.json"),
    Path("test/agent_decision_client_ci_template_dry_run_test.sh"),
    Path("docs/client-integration.md"),
    Path("docs/mcp-client-contract-tests.md"),
]:
    if not path.exists():
        failures.append(f"{doc}: referenced path does not exist: {path}")

if failures:
    print("Agent decision client CI template doc test failed:", file=sys.stderr)
    for failure in failures:
        print(f"- {failure}", file=sys.stderr)
    sys.exit(1)

print("Agent decision client CI template doc test passed")
PY
