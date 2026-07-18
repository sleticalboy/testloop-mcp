#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT INT TERM

assert_contains() {
  file="$1"
  needle="$2"
  if ! grep -F -- "$needle" "$file" >/dev/null 2>&1; then
    echo "expected $file to contain: $needle" >&2
    echo "--- $file ---" >&2
    cat "$file" >&2
    exit 1
  fi
}

cd "$repo_root"

python3 - <<'PY'
from pathlib import Path
import json
import sys

doc = Path("docs/verification-summary-failures.md")
fixture_dir = Path("docs/fixtures/verification-summary")
expected = {
    "install-failed.json": "fix-installation",
    "mcp-transport-failed.json": "inspect-mcp-transport",
    "agent-demo-failed.json": "inspect-agent-demo",
    "showcase-failed.json": "inspect-showcase",
    "user-project-failed.json": "inspect-user-project",
}

text = doc.read_text(encoding="utf-8")
failures = []
for name, action in expected.items():
    path = fixture_dir / name
    if not path.exists():
        failures.append(f"missing fixture: {path}")
        continue
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except json.JSONDecodeError as exc:
        failures.append(f"{path}: invalid JSON at line {exc.lineno}, column {exc.colno}: {exc.msg}")
        continue
    if data.get("overall_status") != "failed":
        failures.append(f"{path}: overall_status must be failed")
    if data.get("failed_count") != 1:
        failures.append(f"{path}: failed_count must be 1")
    sections = data.get("sections")
    if not isinstance(sections, list) or not any(section.get("status") == "failed" for section in sections if isinstance(section, dict)):
        failures.append(f"{path}: must contain a failed section")
    if name not in text:
        failures.append(f"{doc}: missing fixture name {name}")
    if f"`{action}`" not in text:
        failures.append(f"{doc}: missing action `{action}` for {name}")

extra = sorted(path.name for path in fixture_dir.glob("*.json") if path.name not in expected)
if extra:
    failures.append("unexpected verification summary fixtures: " + ", ".join(extra))

if failures:
    print("verification summary failure fixture index failed:", file=sys.stderr)
    for failure in failures:
        print(f"- {failure}", file=sys.stderr)
    sys.exit(1)
PY

for case in \
  "install-failed.json:fix-installation" \
  "mcp-transport-failed.json:inspect-mcp-transport" \
  "agent-demo-failed.json:inspect-agent-demo" \
  "showcase-failed.json:inspect-showcase" \
  "user-project-failed.json:inspect-user-project"
do
  fixture="${case%%:*}"
  action="${case#*:}"
  out="${tmp_dir}/${fixture}.out"
  go run ./examples/verification-summary-decision-demo "docs/fixtures/verification-summary/${fixture}" > "$out"
  assert_contains "$out" "verification_summary: status=failed failed=1"
  assert_contains "$out" "agent_next_step=${action}"
  assert_contains "$out" "markdown_report=/tmp/testloop-"
done

echo "verification summary failure fixtures test passed"
