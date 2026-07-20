#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"

cd "$repo_root"

python3 - <<'PY'
from pathlib import Path
import sys

workflow = Path(".github/workflows/ci.yml")
text = workflow.read_text(encoding="utf-8")

failures = []
for script in sorted(Path("test").glob("*_test.sh")):
    command = f"sh {script.as_posix()}"
    if command not in text:
        failures.append(f"{workflow}: missing CI command {command!r}")

if failures:
    print("CI workflow test failed:", file=sys.stderr)
    for failure in failures:
        print(f"- {failure}", file=sys.stderr)
    sys.exit(1)

print("CI workflow test passed")
PY
