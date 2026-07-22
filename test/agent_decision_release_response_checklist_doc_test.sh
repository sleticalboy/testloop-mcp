#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"

cd "$repo_root"

python3 - <<'PY'
from pathlib import Path
import json
import re
import sys

doc = Path("docs/agent-decision-release-response-checklist.md")
text = doc.read_text(encoding="utf-8")

required = [
    "scripts/showcase-agent-decision-client-release-smoke.sh --json > /tmp/testloop-release-smoke-summary.json",
    "scripts/install-agent-decision-release-response-client.sh --summary-json /tmp/testloop-release-smoke-summary.json --json /absolute/path/to/client-repo > /tmp/testloop-release-response-install-summary.json",
    "node scripts/validate-agent-decision-release-response-client-install-summary.mjs /tmp/testloop-release-response-install-summary.json",
    "cd /absolute/path/to/client-repo/testloop-release-response-client",
    "npm test --silent",
    "testloop-release-response-client/",
    ".github/workflows/testloop-release-response-contract.yml",
    "testloop-release-response.json",
    "agent_next_step=ready",
    "agent-decision-release-response-client-install-summary.schema.json",
    "./fixtures/agent-decision-release-response-client-install-summary/passed.json",
    "agent-decision-client-release-response.schema.json",
    "inspect-release-installer",
    "inspect-release-client-response",
    "inspect-release-consumer-response",
    "inspect-agent-decision-fixtures",
    "inspect-release-smoke-summary",
    "scripts/showcase-agent-decision-client-release-response-ci.sh --json",
    "scripts/showcase-release-response-adopter.sh --json",
    "../examples/release-response-adopter/README.md",
    "node scripts/export-agent-decision-release-response-client.mjs /tmp/testloop-release-response-client",
    "scripts/verify-release-candidate.sh v0.5.20",
    "./agent-decision-release-response-client.md",
    "./client-integration.md",
    "./agent-decision-client-ci-checklist.md",
]

failures = [f"{doc}: missing required snippet {item!r}" for item in required if item not in text]
for path in [
    Path("scripts/showcase-agent-decision-client-release-smoke.sh"),
    Path("scripts/install-agent-decision-release-response-client.sh"),
    Path("scripts/validate-agent-decision-release-response-client-install-summary.mjs"),
    Path("scripts/showcase-agent-decision-client-release-response-ci.sh"),
    Path("scripts/showcase-release-response-adopter.sh"),
    Path("examples/release-response-adopter/README.md"),
    Path("examples/release-response-adopter/scripts/read-testloop-release-response.mjs"),
    Path("scripts/export-agent-decision-release-response-client.mjs"),
    Path("scripts/verify-release-candidate.sh"),
    Path("docs/fixtures/agent-decision-release-response-client-install-summary.schema.json"),
    Path("docs/fixtures/agent-decision-release-response-client-install-summary/passed.json"),
    Path("docs/fixtures/agent-decision-client-release-response.schema.json"),
    Path("docs/agent-decision-release-response-client.md"),
    Path("docs/client-integration.md"),
    Path("docs/agent-decision-client-ci-checklist.md"),
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
        expected = {
            "status": "written",
            "release_ref": "v0.5.20",
            "fixture_count": 8,
            "agent_next_step": "ready",
            "npm_exit_code": 0,
            "failures": [],
        }
        for key, want in expected.items():
            if payload.get(key) != want:
                failures.append(f"{doc}: json example {key} = {payload.get(key)!r}, want {want!r}")

if failures:
    print("Agent decision release response checklist doc test failed:", file=sys.stderr)
    for failure in failures:
        print(f"- {failure}", file=sys.stderr)
    sys.exit(1)

print("Agent decision release response checklist doc test passed")
PY
