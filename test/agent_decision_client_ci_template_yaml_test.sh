#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"

cd "$repo_root"

python3 - <<'PY'
from pathlib import Path
import re
import subprocess
import sys

doc = Path("docs/agent-decision-client-ci-template.md")
text = doc.read_text(encoding="utf-8")
blocks = re.findall(r"```yaml\n(.*?)\n```", text, flags=re.S)

if len(blocks) != 1:
    print(f"{doc}: expected exactly 1 complete yaml workflow example, found {len(blocks)}", file=sys.stderr)
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
if jobs.nil? || !jobs.is_a?(Hash) || !jobs.key?("agent-decision-contract")
  missing << "jobs.agent-decision-contract"
end

if missing.any?
  warn "workflow missing required keys: #{missing.join(", ")}"
  exit 1
end
'''

block = blocks[0]
result = subprocess.run(["ruby", "-e", ruby_program], input=block, text=True, capture_output=True)
failures = []
if result.returncode != 0:
    failures.append(result.stderr.strip() or "ruby yaml validation failed")

required = [
    "name: testloop agent decision contract",
    "repository: sleticalboy/testloop-mcp",
    "ref: v0.5.18",
    "path: .testloop-mcp",
    "actions/setup-node@v4",
    "TESTLOOP_AGENT_DECISION_CLIENT_DIR=/tmp/testloop-agent-decision-client",
    "set -euo pipefail",
    ".testloop-mcp/scripts/showcase-agent-decision-client-ci.sh --json",
    "tee /tmp/testloop-agent-decision-client-summary.json",
    "Render Agent decision response",
    ".testloop-mcp/scripts/render-agent-decision-client-ci-response.mjs",
    "tee /tmp/testloop-agent-decision-client-response.json",
    "actions/upload-artifact@v4",
    "/tmp/testloop-agent-decision-client-summary.json",
    "/tmp/testloop-agent-decision-client-response.json",
    "/tmp/testloop-agent-decision-client/agent-decision-fixtures-result.json",
]
for item in required:
    if item not in block:
        failures.append(f"workflow block missing required snippet {item!r}")

if "scripts/run-first-run-ci.sh" in block or "scripts/run-onboarding-ci.sh" in block:
    failures.append("Agent decision contract template must not use project first-run/onboarding bootstrap")

if failures:
    print("Agent decision client CI template YAML test failed:", file=sys.stderr)
    for failure in failures:
        print(f"- {failure}", file=sys.stderr)
    sys.exit(1)

print("Agent decision client CI template YAML test passed (1 workflow)")
PY
