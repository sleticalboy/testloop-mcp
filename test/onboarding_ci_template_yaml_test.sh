#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"

cd "$repo_root"

python3 - <<'PY'
from pathlib import Path
import re
import subprocess
import sys

doc = Path("docs/onboarding-ci-template.md")
text = doc.read_text(encoding="utf-8")
blocks = re.findall(r"```yaml\n(.*?)\n```", text, flags=re.S)

if len(blocks) != 2:
    print(f"{doc}: expected exactly 2 complete yaml workflow examples, found {len(blocks)}", file=sys.stderr)
    sys.exit(1)

ruby_program = r'''
require "yaml"

begin
  data = YAML.load(STDIN.read)
rescue Psych::SyntaxError => e
  warn "invalid YAML: #{e.message}"
  exit 1
end

unless data.is_a?(Hash)
  warn "workflow must be a YAML mapping"
  exit 1
end

missing = []
missing << "name" unless data.key?("name")
missing << "on" unless data.key?("on") || data.key?(true)
missing << "jobs" unless data.key?("jobs")

jobs = data["jobs"]
if jobs.nil? || !jobs.is_a?(Hash) || !jobs.key?("onboarding")
  missing << "jobs.onboarding"
end

if missing.any?
  warn "workflow missing required keys: #{missing.join(", ")}"
  exit 1
end
'''

failures = []
seen_names = set()
for index, block in enumerate(blocks, start=1):
    result = subprocess.run(
        ["ruby", "-e", ruby_program],
        input=block,
        text=True,
        capture_output=True,
    )
    if result.returncode != 0:
        failures.append(f"block {index}: {result.stderr.strip() or 'ruby yaml validation failed'}")
        continue

    match = re.search(r"^name:\s*(.+)$", block, flags=re.M)
    if not match:
        failures.append(f"block {index}: missing workflow name line")
        continue
    seen_names.add(match.group(1).strip())

expected_names = {"testloop onboarding", "testloop web onboarding"}
missing_names = expected_names - seen_names
if missing_names:
    failures.append(f"{doc}: missing workflow names: {', '.join(sorted(missing_names))}")

if failures:
    print("onboarding CI template YAML test failed:", file=sys.stderr)
    for failure in failures:
        print(f"- {failure}", file=sys.stderr)
    sys.exit(1)

print(f"onboarding CI template YAML test passed ({len(blocks)} workflows)")
PY
