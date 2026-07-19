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
    "./fixtures/validate-coverage-task-ready.json",
    "./fixtures/validate-coverage-task-manual-review-internal.json",
    "./fixtures/validate-coverage-task-apply-fix-suggestions.json",
    "./fixtures/validate-coverage-task-needs-better-input.json",
    "docs/fixtures/first-run-artifacts/user-project-smoke-failed/",
    "sh scripts/render-first-run-agent-response.sh",
    "go run ./examples/first-run-agent-response-demo",
    "first_run_agent_next_step=inspect-user-project",
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
    "go run ./examples/first-run-agent-response-demo": Path("examples/first-run-agent-response-demo/main.go"),
    "sh scripts/render-first-run-agent-response.sh": Path("scripts/render-first-run-agent-response.sh"),
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
