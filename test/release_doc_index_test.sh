#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"

cd "$repo_root"

python3 - <<'PY'
from pathlib import Path
import sys

readme = Path("README.md")
text = readme.read_text(encoding="utf-8")

required_entries = {
    "Agent 工作流": "./docs/agent-workflow.md",
    "Agent 结构化契约": "./docs/agent-contract.md",
    "Agent Action 决策表": "./docs/agent-action-guide.md",
    "validate_coverage_task 结构化返回样例": "./docs/validate-coverage-task-samples.md",
    "真实结构化 fixture": "./docs/fixtures.md",
    "客户端集成说明": "./docs/client-integration.md",
    "MCP 客户端契约测试说明": "./docs/mcp-client-contract-tests.md",
    "首跑诊断": "./docs/first-run-diagnostics.md",
    "首跑诊断失败样例": "./docs/first-run-failures.md",
    "用户项目验收报告": "./docs/verification-report.md",
    "Onboarding CI 外部项目演练": "./docs/onboarding-ci-external-dry-run.md",
    "Onboarding CI 复制模板": "./docs/onboarding-ci-template.md",
    "Onboarding CI 失败排查": "./docs/onboarding-ci-failure-triage.md",
    "真实接入案例模板": "./docs/real-integration-cases.md",
    "验收 summary 失败分流样例": "./docs/verification-summary-failures.md",
    "验收报告 CI 集成": "./docs/verification-ci.md",
}

required_commands = [
    "go run ./examples/agent-decision-demo",
    "go run ./examples/mcp-client-demo",
    "go run ./examples/verification-summary-decision-demo",
    "scripts/doctor-first-run.sh",
    "scripts/generate-verification-report.sh",
    "scripts/run-onboarding-ci.sh",
    "scripts/showcase-onboarding-ci-external-project.sh",
    "scripts/showcase-agent-onboarding-report.sh",
    "scripts/showcase-onboarding.sh",
    "scripts/verify-client-setup.sh",
    "scripts/verify-mcp-process-smoke.sh",
]

failures = []
for label, target in required_entries.items():
    if target not in text:
        failures.append(f"{readme}: missing release doc index target {target!r} ({label})")
        continue
    path = Path(target)
    if not path.exists():
        failures.append(f"{readme}: release doc index target does not exist: {target}")

for command in required_commands:
    if command not in text:
        failures.append(f"{readme}: missing release doc index command {command!r}")

if failures:
    print("release doc index test failed:", file=sys.stderr)
    for failure in failures:
        print(f"- {failure}", file=sys.stderr)
    sys.exit(1)

print(f"release doc index test passed ({len(required_entries)} docs, {len(required_commands)} commands)")
PY
