#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
cd "$repo_root"

python3 - <<'PY'
from pathlib import Path
import sys

doc = Path("docs/first-run-failures.md")
fixture_dir = Path("docs/fixtures/first-run")
expected = {
    "fix-installation.txt": "fix-installation",
    "inspect-mcp-transport.txt": "inspect-mcp-transport",
    "inspect-agent-demo.txt": "inspect-agent-demo",
    "inspect-showcase.txt": "inspect-showcase",
    "inspect-user-project.txt": "inspect-user-project",
}

text = doc.read_text(encoding="utf-8")
failures = []
for name, action in expected.items():
    path = fixture_dir / name
    if not path.exists():
        failures.append(f"missing fixture: {path}")
        continue
    content = path.read_text(encoding="utf-8")
    required = [
        "testloop-mcp first-run diagnostic context",
        "first_run_status=failed",
        "first_run_failed_count=1",
        f"first_run_agent_next_step={action}",
        "first_run_report=/tmp/testloop-",
        "first_run_summary_json=/tmp/testloop-",
        "first_run_decision=/tmp/testloop-",
        "first_run_log=/tmp/testloop-",
        "Suggested prompt:",
        "不要直接改生成测试",
    ]
    for snippet in required:
        if snippet not in content:
            failures.append(f"{path}: missing {snippet!r}")
    if name not in text:
        failures.append(f"{doc}: missing fixture name {name}")
    if f"`{action}`" not in text:
        failures.append(f"{doc}: missing action `{action}`")

extra = sorted(path.name for path in fixture_dir.glob("*.txt") if path.name not in expected)
if extra:
    failures.append("unexpected first-run fixtures: " + ", ".join(extra))

if failures:
    print("first-run failure fixtures test failed:", file=sys.stderr)
    for failure in failures:
        print(f"- {failure}", file=sys.stderr)
    sys.exit(1)

print("first-run failure fixtures test passed")
PY
