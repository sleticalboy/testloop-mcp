#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"

cd "$repo_root"

python3 - <<'PY'
from pathlib import Path
import json
import re
import sys

doc = Path("docs/agent-decision-client-ci-checklist.md")
text = doc.read_text(encoding="utf-8")

required = [
    "v0.5.21",
    "install-agent-decision-client-ci-template.sh",
    "https://raw.githubusercontent.com/sleticalboy/testloop-mcp/main/scripts/install-agent-decision-client-ci-template.sh",
    ".github/workflows/testloop-agent-decision-contract.yml",
    ".testloop-mcp/scripts/showcase-agent-decision-client-ci.sh --json",
    "/tmp/testloop-agent-decision-client-summary.json",
    "/tmp/testloop-agent-decision-client-response.json",
    "/tmp/testloop-agent-decision-client/agent-decision-fixtures-result.json",
    "testloop-agent-decision-client-response.json",
    "agent_next_step=inspect-client-validator",
    "agent_next_step=inspect-agent-decision-fixtures",
    "agent-decision-fixtures.json",
    "fixtures[].expected_decision",
    "passed/ready",
    "failed/apply_fix_suggestions",
    "failed/needs_better_input",
    "manual_review_*",
    "scripts/showcase-agent-decision-client-ci-template-install.sh --json",
    "./fixtures/agent-decision-client-ci-template-install-summary.schema.json",
    "./fixtures/agent-decision-client-ci-template-install-summary/passed.json",
    "node scripts/validate-agent-decision-client-ci-install-summary.mjs /path/to/install-summary.json",
    "scripts/showcase-agent-decision-client-consumer-smoke.sh --json",
    "agent_decision_client_consumer_smoke_status=passed",
    "client_summary_json",
    "client_summary_validator_json",
    "fixture_validation_json",
    "agent_response_json",
    "./fixtures/agent-decision-client-consumer-smoke-summary.schema.json",
    "./fixtures/agent-decision-client-consumer-smoke-summary/passed.json",
    "./fixtures/agent-decision-client-consumer-smoke-summary/client-summary-validator-failed.json",
    "./fixtures/agent-decision-client-consumer-smoke-summary/validator-failed.json",
    "./fixtures/agent-decision-client-consumer-smoke-summary/fixture-drift.json",
    "node scripts/validate-agent-decision-client-consumer-smoke-summary.mjs /path/to/consumer-smoke-summary.json",
    "node scripts/render-agent-decision-client-consumer-response.mjs /path/to/consumer-smoke-summary.json",
    "agent_next_step=ready",
    "inspect-consumer-smoke-validator",
    "inspect-agent-decision-fixtures",
    "inspect-consumer-smoke-summary",
    "./agent-decision-client-ci-template.md",
    "./client-integration.md",
    "./mcp-client-contract-tests.md",
]

failures = [f"{doc}: missing required snippet {item!r}" for item in required if item not in text]
for path in [
    Path("scripts/install-agent-decision-client-ci-template.sh"),
    Path("scripts/showcase-agent-decision-client-ci-template-install.sh"),
    Path("scripts/validate-agent-decision-client-ci-install-summary.mjs"),
    Path("scripts/showcase-agent-decision-client-consumer-smoke.sh"),
    Path("scripts/validate-agent-decision-client-consumer-smoke-summary.mjs"),
    Path("scripts/render-agent-decision-client-consumer-response.mjs"),
    Path("docs/fixtures/agent-decision-client-ci-template-install-summary.schema.json"),
    Path("docs/fixtures/agent-decision-client-ci-template-install-summary/passed.json"),
    Path("docs/fixtures/agent-decision-client-consumer-smoke-summary.schema.json"),
    Path("docs/fixtures/agent-decision-client-consumer-smoke-summary/passed.json"),
    Path("docs/fixtures/agent-decision-client-consumer-smoke-summary/client-summary-validator-failed.json"),
    Path("docs/fixtures/agent-decision-client-consumer-smoke-summary/validator-failed.json"),
    Path("docs/fixtures/agent-decision-client-consumer-smoke-summary/fixture-drift.json"),
    Path("docs/agent-decision-client-ci-template.md"),
    Path("docs/client-integration.md"),
    Path("docs/mcp-client-contract-tests.md"),
]:
    if not path.exists():
        failures.append(f"{doc}: referenced path does not exist: {path}")

blocks = re.findall(r"```json\n(.*?)\n```", text, flags=re.S)
if len(blocks) != 1:
    failures.append(f"{doc}: expected exactly 1 json example, found {len(blocks)}")
else:
    try:
        payload = json.loads(blocks[0])
    except json.JSONDecodeError as exc:
        failures.append(f"{doc}: invalid json example at line {exc.lineno}, column {exc.colno}: {exc.msg}")
    else:
        if payload.get("status") != "passed":
            failures.append(f"{doc}: json example status must be passed")
        if payload.get("fixture_count") != 8:
            failures.append(f"{doc}: json example fixture_count must be 8")
        if payload.get("failures") != []:
            failures.append(f"{doc}: json example failures must be empty")

if failures:
    print("Agent decision client CI checklist doc test failed:", file=sys.stderr)
    for failure in failures:
        print(f"- {failure}", file=sys.stderr)
    sys.exit(1)

print("Agent decision client CI checklist doc test passed")
PY
