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
    "TESTLOOP_REPORT_SUMMARY_JSON=/tmp/testloop-summary.json",
    "scripts/generate-verification-report.sh /tmp/testloop-mcp /tmp/testloop-report.md",
    "go run ./examples/verification-summary-decision-demo /tmp/testloop-summary.json",
    "actions/upload-artifact@v4",
    "if: always()",
    "/tmp/testloop-report.md",
    "/tmp/testloop-summary.json",
    "TESTLOOP_REPORT_PROJECT_COMMAND='go test ./...'",
    "TESTLOOP_REPORT_PROJECT_COMMAND='pnpm install --frozen-lockfile && pnpm build'",
    "./regression-smoke.md",
]

failures = []
for snippet in required_snippets:
    if snippet not in text:
        failures.append(f"{doc}: missing required snippet {snippet!r}")

command_paths = {
    "scripts/generate-verification-report.sh": Path("scripts/generate-verification-report.sh"),
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
