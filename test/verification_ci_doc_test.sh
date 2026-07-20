#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"

cd "$repo_root"

python3 - <<'PY'
from pathlib import Path
import sys

doc = Path("docs/verification-ci.md")
text = doc.read_text(encoding="utf-8")

required_snippets = [
    "scripts/run-onboarding-ci.sh",
    "sh scripts/verify-agent-artifact.sh onboarding /tmp/testloop-onboarding",
    "Artifact verification",
    "bash /tmp/testloop-onboarding-ci.sh 'go test ./...'",
    "TESTLOOP_ONBOARDING_OUTPUT_DIR=/tmp/testloop-onboarding",
    "./first-run-ci-template.md",
    "first-run-context.txt",
    "first-run.log",
    "/tmp/testloop-onboarding/verification-report.md",
    "/tmp/testloop-onboarding/verification-summary.json",
    "/tmp/testloop-onboarding/verification-summary.schema.json",
    "/tmp/testloop-onboarding/agent-decision.txt",
    "TESTLOOP_REPORT_SUMMARY_JSON=/tmp/testloop-summary.json",
    "scripts/generate-verification-report.sh /tmp/testloop-mcp /tmp/testloop-report.md",
    "go run ./examples/verification-summary-decision-demo /tmp/testloop-summary.json",
    "actions/upload-artifact@v4",
    "if: always()",
    "bash /tmp/testloop-onboarding-ci.sh 'pnpm install --frozen-lockfile && pnpm build'",
    "TESTLOOP_ONBOARDING_OUTPUT_DIR=/tmp/testloop-web-onboarding",
    "/tmp/testloop-web-onboarding/verification-summary.schema.json",
    "./onboarding-ci-template.md",
    "./onboarding-ci-failure-triage.md",
    "./regression-smoke.md",
]

failures = []
for snippet in required_snippets:
    if snippet not in text:
        failures.append(f"{doc}: missing required snippet {snippet!r}")

command_paths = {
    "scripts/run-onboarding-ci.sh": Path("scripts/run-onboarding-ci.sh"),
    "scripts/showcase-agent-onboarding-report.sh": Path("scripts/showcase-agent-onboarding-report.sh"),
    "scripts/generate-verification-report.sh": Path("scripts/generate-verification-report.sh"),
    "scripts/run-first-run-ci.sh": Path("scripts/run-first-run-ci.sh"),
    "sh scripts/verify-agent-artifact.sh": Path("scripts/verify-agent-artifact.sh"),
    "go run ./examples/verification-summary-decision-demo": Path("examples/verification-summary-decision-demo/main.go"),
}
for command, path in command_paths.items():
    if command in text and not path.exists():
        failures.append(f"{doc}: command {command!r} points to missing {path}")

if failures:
    print("verification CI doc test failed:", file=sys.stderr)
    for failure in failures:
        print(f"- {failure}", file=sys.stderr)
    sys.exit(1)

print("verification CI doc test passed")
PY
