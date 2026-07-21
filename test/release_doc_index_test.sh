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
    "接入方一页式验证指南": "./docs/adopter-verification-guide.md",
    "Agent 工作流": "./docs/agent-workflow.md",
    "Agent 结构化契约": "./docs/agent-contract.md",
    "Agent Action 决策表": "./docs/agent-action-guide.md",
    "validate_coverage_task 结构化返回样例": "./docs/validate-coverage-task-samples.md",
    "真实结构化 fixture": "./docs/fixtures.md",
    "客户端集成说明": "./docs/client-integration.md",
    "MCP 客户端契约测试说明": "./docs/mcp-client-contract-tests.md",
    "Agent 决策客户端 CI 模板": "./docs/agent-decision-client-ci-template.md",
    "首跑诊断": "./docs/first-run-diagnostics.md",
    "首跑诊断 CI 复制模板": "./docs/first-run-ci-template.md",
    "首跑诊断失败样例": "./docs/first-run-failures.md",
    "CI 失败后交给 Agent": "./docs/ci-agent-triage.md",
    "Agent response artifact contract": "./docs/agent-response-artifact-contract.md",
    "first-run Agent 回复格式": "./docs/first-run-agent-response.md",
    "first-run artifact Agent 消费演示": "./docs/first-run-agent-artifact-demo.md",
    "用户项目验收报告": "./docs/verification-report.md",
    "Onboarding CI 外部项目演练": "./docs/onboarding-ci-external-dry-run.md",
    "Onboarding CI 复制模板": "./docs/onboarding-ci-template.md",
    "Onboarding CI 失败排查": "./docs/onboarding-ci-failure-triage.md",
    "真实接入案例模板": "./docs/real-integration-cases.md",
    "验收 summary 失败分流样例": "./docs/verification-summary-failures.md",
    "验收报告 CI 集成": "./docs/verification-ci.md",
}

required_commands = [
    "curl -fsSL https://raw.githubusercontent.com/sleticalboy/testloop-mcp/main/scripts/run-first-run-ci.sh",
    "curl -fsSL https://raw.githubusercontent.com/sleticalboy/testloop-mcp/main/scripts/run-onboarding-ci.sh",
    "go run ./examples/agent-decision-demo",
    "go run ./examples/agent-response-manifest-demo",
    "go run ./examples/first-run-agent-response-demo",
    "go run ./examples/mcp-client-demo",
    "go run ./examples/verification-summary-decision-demo",
    "node scripts/validate-agent-decision-fixtures.mjs",
    "node scripts/validate-agent-decision-fixtures.mjs --json",
    "node scripts/export-agent-decision-fixtures.mjs",
    "scripts/showcase-agent-decision-client-ci.sh",
    "scripts/showcase-agent-decision-client-ci.sh --json",
    "scripts/showcase-agent-decision-client-release-smoke.sh --json",
    "scripts/showcase-agent-decision-client-release-response-smoke.sh --json",
    "scripts/showcase-agent-decision-client-release-response-ci.sh --json",
    "scripts/install-agent-decision-release-response-client.sh",
    "node scripts/validate-agent-decision-release-response-client-install-summary.mjs",
    "node scripts/export-agent-decision-release-response-client.mjs",
    "node scripts/validate-agent-decision-client-ci-install-summary.mjs",
    "npm test --silent",
    "scripts/doctor-first-run.sh",
    "sh scripts/render-first-run-agent-response.sh",
    "sh scripts/render-onboarding-agent-response.sh",
    "sh scripts/verify-agent-artifact.sh",
    "sh scripts/verify-agent-artifact.sh manifest docs/fixtures/agent-response-artifact-manifest.json",
    "--json",
    "scripts/run-first-run-ci.sh",
    "scripts/generate-verification-report.sh",
    "scripts/run-onboarding-ci.sh",
    "scripts/showcase-dual-project-report.sh",
    "scripts/showcase-laoxia-scaffold-report.sh",
    "scripts/showcase-onboarding-ci-external-project.sh",
    "scripts/showcase-agent-onboarding-report.sh",
    "scripts/showcase-onboarding.sh",
    "scripts/verify-client-setup.sh",
    "scripts/verify-mcp-process-smoke.sh",
    "scripts/verify-release-candidate.sh",
]

required_phrases = [
    "用户项目接入：直接复制",
    "复制 first-run bootstrap",
    "复制 onboarding bootstrap",
    "GitHub Actions 最小片段",
    ".github/workflows/testloop-first-run.yml",
    "actions/upload-artifact@v4",
    "CI 失败后交给 Agent",
    "Agent response artifact contract",
    "agent-response artifact manifest",
    "Artifact verification",
    "离线校验必备文件",
    "docs/fixtures/agent-response-artifact-manifest.json",
    "agent-response-artifact-manifest.schema.json",
    "manifest_schema_version=1",
    "artifact_count=2",
    "kind=first-run action_field=first_run_agent_next_step expected_action=inspect-user-project",
    "kind=onboarding action_field=agent_next_step expected_action=inspect-user-project",
    "first-run Agent 回复格式",
    "first-run 失败 artifact fixture",
    "./docs/fixtures/first-run-artifacts/user-project-smoke-failed/",
    "first-run-context.txt",
    "onboarding artifact",
    "summary schema",
    "七件套",
    "五件套",
    "真实 server / web 实跑记录",
    "JSON 输出会包含",
    "decisions[]",
    "failures[]",
    "最小决策 fixture 包",
    "package.json",
    "client_expectation",
    "agent_decision_client_status=passed",
    "agent_decision_fixture_count=8",
    "release tag raw installer",
    "Agent 决策 release response 客户端接入",
    "testloop-release-response-client",
    "agent-decision-release-response-client-install-summary.schema.json",
    "GitHub Actions",
    "validator_exit_code",
    ".github/workflows/testloop-agent-decision-contract.yml",
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

for phrase in required_phrases:
    if phrase not in text:
        failures.append(f"{readme}: missing release doc index phrase {phrase!r}")

if failures:
    print("release doc index test failed:", file=sys.stderr)
    for failure in failures:
        print(f"- {failure}", file=sys.stderr)
    sys.exit(1)

print(f"release doc index test passed ({len(required_entries)} docs, {len(required_commands)} commands)")
PY
