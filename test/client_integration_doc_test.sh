#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"

cd "$repo_root"

python3 - <<'PY'
from pathlib import Path
import re
import sys

doc = Path("docs/client-integration.md")
text = doc.read_text(encoding="utf-8")

required_snippets = [
    "go run ./examples/agent-decision-demo",
    "./fixtures/agent-decision-fixtures.json",
    "./fixtures/agent-decision-fixtures.schema.json",
    "fixtures[].expected_decision",
    "failed/manual_review_*",
    "./fixtures/validate-coverage-task-ready.json",
    "./fixtures/validate-coverage-task-manual-review-internal.json",
    "./fixtures/validate-coverage-task-apply-fix-suggestions.json",
    "./fixtures/validate-coverage-task-needs-better-input.json",
    "./fixtures/real-project-agent-loop/laoxia-server-go-utils.json",
    "./fixtures/real-project-agent-loop/mcp-hub-vitest-repair.json",
    "./fixtures/real-project-agent-loop/haoy-apk-station-py-environment.json",
    "./fixtures/real-project-agent-loop/haoy-apk-station-py-external-service.json",
    "real-project-agent-loop/*.json",
    "regression_note",
    "redaction_note",
    "docs/fixtures/first-run-artifacts/user-project-smoke-failed/",
    "docs/fixtures/onboarding-artifacts/user-project-smoke-failed/",
    "sh scripts/render-first-run-agent-response.sh",
    "sh scripts/render-onboarding-agent-response.sh",
    "sh scripts/verify-agent-artifact.sh",
    "sh scripts/verify-agent-artifact.sh \\",
    "--json",
    "node scripts/validate-agent-decision-fixtures.mjs --json \\",
    "node scripts/export-agent-decision-fixtures.mjs /tmp/testloop-agent-decision-fixtures",
    "scripts/showcase-agent-decision-client-ci.sh",
    "scripts/showcase-agent-decision-client-ci.sh --json",
    "node scripts/render-agent-decision-client-ci-response.mjs /path/to/testloop-agent-decision-client-summary.json",
    "inspect-client-validator",
    "inspect-agent-decision-client-summary",
    "./agent-decision-client-ci-template.md",
    "node scripts/validate-agent-decision-client-ci-install-summary.mjs /path/to/install-summary.json",
    "scripts/showcase-agent-decision-client-consumer-smoke.sh --json",
    "node scripts/validate-agent-decision-client-consumer-smoke-summary.mjs /path/to/consumer-smoke-summary.json",
    "node scripts/render-agent-decision-client-consumer-response.mjs /path/to/consumer-smoke-summary.json",
    "node scripts/export-agent-decision-release-response-client.mjs /tmp/testloop-release-response-client",
    "scripts/install-agent-decision-release-response-client.sh /absolute/path/to/client-repo",
    "node scripts/validate-agent-decision-release-response-client-install-summary.mjs /path/to/install-summary.json",
    "./fixtures/agent-decision-release-response-client-install-summary.schema.json",
    "./fixtures/agent-decision-release-response-client-install-summary/passed.json",
    "scripts/showcase-agent-decision-client-release-response-ci.sh --json",
    "scripts/showcase-release-response-adopter.sh --json",
    "node scripts/validate-release-response-adopter-summary.mjs /path/to/release-response-adopter-summary.json",
    "./fixtures/release-response-adopter-summary.schema.json",
    "./fixtures/release-response-adopter-summary/passed.json",
    "./fixtures/release-response-adopter-summary/invalid-response.json",
    "../examples/release-response-adopter/README.md",
    ".github/workflows/testloop-release-response-contract.yml",
    "agent_response_json",
    "agent_next_step",
    "inspect-consumer-smoke-validator",
    "inspect-agent-decision-fixtures",
    "inspect-consumer-smoke-summary",
    "./fixtures/agent-decision-client-ci-template-install-summary/passed.json",
    "./fixtures/agent-decision-client-consumer-smoke-summary/passed.json",
    "./fixtures/agent-decision-client-consumer-smoke-summary/validator-failed.json",
    "./fixtures/agent-decision-client-consumer-smoke-summary/fixture-drift.json",
    "agent_decision_client_status=passed",
    "agent_decision_fixture_count=8",
    "validator_exit_code",
    "最小决策 fixture 包",
    "package.json",
    "npm test --silent",
    "fixture_count",
    "decisions[]",
    "fixtures[]",
    "failures[]",
    "client_expectation",
    "JSON Schema 工具链",
    "agent_artifact_manifest_status=passed",
    "agent_artifact_status=passed",
    "./fixtures/agent-response-artifact-manifest.json",
    "./fixtures/agent-response-artifact-manifest.schema.json",
    "./fixtures/verification-summary.schema.json",
    "./fixtures/dual-project-summary.schema.json",
    "./fixtures/dual-project-summary/laoxia-passed.json",
    "local_summary_schema=verification-summary.schema.json",
    "sections[].signals.action",
    "response_action=inspect-user-project",
    "artifact_count=2",
    "artifacts[].section_signals",
    "verification-summary-decision-demo",
    "go run ./examples/agent-response-manifest-demo",
    "go run ./examples/first-run-agent-response-demo",
    "first_run_agent_next_step=inspect-user-project",
    "agent_next_step=inspect-user-project",
    "failed_section=用户项目 smoke",
    "structuredContent",
    "content[0].text",
]

failures = []
for snippet in required_snippets:
    if snippet not in text:
        failures.append(f"{doc}: missing required snippet {snippet!r}")

command_paths = {
    "go run ./examples/agent-decision-demo": Path("examples/agent-decision-demo/main.go"),
    "go run ./examples/agent-response-manifest-demo": Path("examples/agent-response-manifest-demo/main.go"),
    "go run ./examples/first-run-agent-response-demo": Path("examples/first-run-agent-response-demo/main.go"),
    "sh scripts/render-first-run-agent-response.sh": Path("scripts/render-first-run-agent-response.sh"),
    "sh scripts/render-onboarding-agent-response.sh": Path("scripts/render-onboarding-agent-response.sh"),
    "sh scripts/verify-agent-artifact.sh": Path("scripts/verify-agent-artifact.sh"),
    "scripts/showcase-agent-decision-client-ci.sh": Path("scripts/showcase-agent-decision-client-ci.sh"),
    "node scripts/render-agent-decision-client-ci-response.mjs": Path("scripts/render-agent-decision-client-ci-response.mjs"),
    "scripts/showcase-agent-decision-client-consumer-smoke.sh": Path("scripts/showcase-agent-decision-client-consumer-smoke.sh"),
    "node scripts/validate-agent-decision-client-consumer-smoke-summary.mjs": Path("scripts/validate-agent-decision-client-consumer-smoke-summary.mjs"),
    "node scripts/render-agent-decision-client-consumer-response.mjs": Path("scripts/render-agent-decision-client-consumer-response.mjs"),
    "scripts/install-agent-decision-release-response-client.sh": Path("scripts/install-agent-decision-release-response-client.sh"),
    "scripts/showcase-release-response-adopter.sh": Path("scripts/showcase-release-response-adopter.sh"),
    "node scripts/validate-release-response-adopter-summary.mjs": Path("scripts/validate-release-response-adopter-summary.mjs"),
    "node scripts/validate-agent-decision-release-response-client-install-summary.mjs": Path("scripts/validate-agent-decision-release-response-client-install-summary.mjs"),
}
for command, path in command_paths.items():
    if command in text and not path.exists():
        failures.append(f"{doc}: command {command!r} points to missing {path}")

fixture_links = re.findall(r"\]\((\./fixtures/[^)]+\.json)\)", text)
for raw_link in fixture_links:
    path = doc.parent / raw_link
    if not path.exists():
        failures.append(f"{doc}: fixture link points to missing {raw_link}")

artifact_links = re.findall(r"\]\((\./first-run-agent-artifact-demo\.md)\)", text)
for raw_link in artifact_links:
    path = doc.parent / raw_link
    if not path.exists():
        failures.append(f"{doc}: artifact demo link points to missing {raw_link}")

if failures:
    print("client integration doc test failed:", file=sys.stderr)
    for failure in failures:
        print(f"- {failure}", file=sys.stderr)
    sys.exit(1)

print(f"client integration doc test passed ({len(fixture_links)} fixture links)")
PY
