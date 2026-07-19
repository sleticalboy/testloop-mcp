#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"

cd "$repo_root"

python3 - <<'PY'
from pathlib import Path
import sys

doc = Path("docs/onboarding-ci-failure-triage.md")
text = doc.read_text(encoding="utf-8")

required_snippets = [
    "$GITHUB_STEP_SUMMARY",
    "Status",
    "Failed sections",
    "agent_next_step",
    "Agent response",
    "verification-report.md",
    "verification-summary.json",
    "agent-decision.txt",
    "agent-response.txt",
    "Agent 四段回复草稿",
    "overall_status",
    "failed_count",
    "fix-installation",
    "inspect-mcp-transport",
    "inspect-agent-demo",
    "inspect-user-project",
    "inspect-showcase",
    "TESTLOOP_ONBOARDING_PROJECT_DIR=/path/to/project",
    "project-smoke-command='go test ./...'",
    "./onboarding-ci-template.md",
    "./verification-ci.md",
]

failures = []
for snippet in required_snippets:
    if snippet not in text:
        failures.append(f"{doc}: missing required snippet {snippet!r}")

linked_paths = [
    Path("docs/onboarding-ci-template.md"),
    Path("docs/verification-ci.md"),
]
for path in linked_paths:
    if not path.exists():
        failures.append(f"{doc}: referenced path does not exist: {path}")

if failures:
    print("onboarding CI failure triage doc test failed:", file=sys.stderr)
    for failure in failures:
        print(f"- {failure}", file=sys.stderr)
    sys.exit(1)

print("onboarding CI failure triage doc test passed")
PY
