#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
cd "$repo_root"

python3 - <<'PY'
from pathlib import Path
import re
import subprocess
import sys

doc = Path("README.md")
text = doc.read_text(encoding="utf-8")
blocks = re.findall(r"```yaml\n(.*?)\n```", text, flags=re.S)
target_blocks = [block for block in blocks if "name: testloop first-run smoke" in block]

if len(target_blocks) != 1:
    print(f"{doc}: expected exactly 1 README first-run workflow snippet, found {len(target_blocks)}", file=sys.stderr)
    sys.exit(1)

block = target_blocks[0]

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
if jobs.nil? || !jobs.is_a?(Hash) || !jobs.key?("first-run")
  missing << "jobs.first-run"
end

if missing.any?
  warn "workflow missing required keys: #{missing.join(", ")}"
  exit 1
end
'''

result = subprocess.run(["ruby", "-e", ruby_program], input=block, text=True, capture_output=True)
if result.returncode != 0:
    print("README CI snippet YAML test failed:", file=sys.stderr)
    print(result.stderr.strip() or "ruby yaml validation failed", file=sys.stderr)
    sys.exit(1)

required = [
    "actions/checkout@v4",
    "actions/setup-go@v5",
    "actions/upload-artifact@v4",
    "scripts/run-first-run-ci.sh",
    "TESTLOOP_MCP_VERSION=v0.5.21",
    "TESTLOOP_FIRST_RUN_OUTPUT_DIR=/tmp/testloop-first-run",
    "TESTLOOP_FIRST_RUN_PROJECT_DIR=\"$PWD\"",
    "go test ./...",
    "if: always()",
    "verification-report.md",
    "verification-summary.json",
    "verification-summary.schema.json",
    "agent-decision.txt",
    "first-run-context.txt",
    "agent-response.txt",
    "first-run.log",
]

missing = [item for item in required if item not in block]
if missing:
    print("README CI snippet test failed:", file=sys.stderr)
    for item in missing:
        print(f"- missing {item}", file=sys.stderr)
    sys.exit(1)

required_text = [
    "./docs/fixtures/agent-response-artifact-manifest.json",
    "./docs/fixtures/agent-response-artifact-manifest.schema.json",
    "./docs/fixtures/verification-summary.schema.json",
    "./docs/fixtures/dual-project-summary.schema.json",
    "./docs/fixtures/first-run-artifacts/user-project-smoke-failed/",
    "node scripts/render-agent-decision-client-ci-response.mjs /path/to/testloop-agent-decision-client-summary.json",
    "scripts/showcase-agent-decision-client-adopter.sh --json",
    "node scripts/validate-agent-decision-client-adopter-summary.mjs /path/to/agent-decision-client-adopter-summary.json",
    "agent-decision-client-adopter-summary.schema.json",
    "./examples/agent-decision-client-adopter/README.md",
    "node scripts/render-agent-decision-client-consumer-response.mjs /path/to/consumer-smoke-summary.json",
    "scripts/showcase-release-response-adopter.sh --json",
    "node scripts/validate-release-response-adopter-artifact-verification.mjs",
    "node scripts/validate-release-response-adopter-summary.mjs /path/to/release-response-adopter-summary.json",
    "release-response-adopter-summary.schema.json",
    "invalid-response.json",
    "read-testloop-release-response-summary.mjs",
    "testloop_release_response_summary_next_step",
    "./examples/release-response-adopter/README.md",
    "agent_response_json",
    "基础客户端 CI response",
    "agent_next_step=ready",
    "./docs/fixtures/agent-decision-client-consumer-smoke-summary/validator-failed.json",
    "./docs/fixtures/agent-decision-client-consumer-smoke-summary/fixture-drift.json",
]

missing_text = [item for item in required_text if item not in text]
if missing_text:
    print("README CI artifact text test failed:", file=sys.stderr)
    for item in missing_text:
        print(f"- missing {item}", file=sys.stderr)
    sys.exit(1)

print("README CI snippet test passed")
PY
