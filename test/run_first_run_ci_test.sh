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

run_expect_code() {
  want="$1"
  output="$2"
  shift 2
  set +e
  "$@" > "$output" 2>&1
  code=$?
  set -e
  if [ "$code" -ne "$want" ]; then
    echo "expected exit code $want, got $code: $*" >&2
    echo "--- $output ---" >&2
    cat "$output" >&2
    exit 1
  fi
}

fake_binary="${tmp_dir}/testloop-mcp"
cat >"$fake_binary" <<'SH'
#!/usr/bin/env sh
case "${1:-}" in
  --version)
    echo "testloop-mcp 0.5.15"
    ;;
  *)
    echo "fake testloop-mcp"
    ;;
esac
SH
chmod +x "$fake_binary"

project_dir="${tmp_dir}/project"
mkdir -p "$project_dir"
out="${tmp_dir}/first-run-ci.out"
output_dir="${tmp_dir}/artifacts"
step_summary="${tmp_dir}/step-summary.md"

project_file="${tmp_dir}/project-file"
printf 'not a directory\n' > "$project_file"
run_expect_code 1 "$out" env \
  TESTLOOP_FIRST_RUN_PROJECT_DIR="$project_file" \
  bash "${repo_root}/scripts/run-first-run-ci.sh" 'echo smoke'
assert_contains "$out" "project path must be a directory"

output_file="${tmp_dir}/output-file"
printf 'not a directory\n' > "$output_file"
run_expect_code 1 "$out" env \
  TESTLOOP_FIRST_RUN_PROJECT_DIR="$project_dir" \
  TESTLOOP_FIRST_RUN_OUTPUT_DIR="$output_file" \
  bash "${repo_root}/scripts/run-first-run-ci.sh" 'echo smoke'
assert_contains "$out" "output path must be a directory"

run_expect_code 0 "$out" env \
  TESTLOOP_MCP_REPO_DIR="$repo_root" \
  TESTLOOP_MCP_COMMAND="$fake_binary" \
  TESTLOOP_MCP_VERSION=v0.5.15 \
  TESTLOOP_FIRST_RUN_PROJECT_DIR="$project_dir" \
  TESTLOOP_FIRST_RUN_OUTPUT_DIR="$output_dir" \
  TESTLOOP_REPORT_SKIP_BASIC=true \
  TESTLOOP_REPORT_SKIP_PROCESS_SMOKE=true \
  TESTLOOP_REPORT_SKIP_AGENT_DEMO=true \
  GITHUB_STEP_SUMMARY="$step_summary" \
  bash "${repo_root}/scripts/run-first-run-ci.sh" 'printf "project smoke ok\n"'

assert_contains "$out" "testloop_mcp_repo=$repo_root"
assert_contains "$out" "testloop_mcp_binary=$fake_binary"
assert_contains "$out" "testloop_first_run_output_dir=$output_dir"
assert_contains "$out" "testloop_project_dir=$project_dir"
assert_contains "$out" "testloop_project_command=printf \"project smoke ok\\n\""
assert_contains "$out" "first_run_status=passed"
assert_contains "$out" "first_run_agent_next_step=ready"
assert_contains "$out" "first_run_context=$output_dir/first-run-context.txt"
assert_contains "$out" "agent_artifact_status=passed"
assert_contains "$out" "artifact_kind=first-run"
assert_contains "$out" "response_action=ready"
assert_contains "$output_dir/agent-response.txt" "结论：testloop-mcp 接入链路通过"
assert_contains "$output_dir/agent-response.txt" "- first_run_agent_next_step=ready"
assert_contains "$output_dir/agent-response.txt" "- first_run_status=passed"
assert_contains "$output_dir/agent-response.txt" "- first_run_failed_count=0"
assert_contains "$output_dir/verification-report.md" "project smoke ok"
assert_contains "$output_dir/verification-summary.json" '"overall_status": "passed"'
assert_contains "$output_dir/verification-summary.json" '"signals": {'
assert_contains "$output_dir/verification-summary.json" '"action": "manual_review"'
assert_contains "$output_dir/verification-summary.schema.json" '"title": "testloop-mcp verification summary"'
assert_contains "$output_dir/agent-decision.txt" "agent_next_step=ready"
assert_contains "$output_dir/agent-decision.txt" "section_signal=独立 CLI 生成动作 smoke action=manual_review"
assert_contains "$output_dir/first-run-context.txt" "first_run_agent_next_step=ready"
assert_contains "$output_dir/first-run.log" "onboarding_report=$output_dir/verification-report.md"
assert_contains "$output_dir/agent-response.txt" "- section_signal=独立 CLI 生成动作 smoke action=manual_review"
assert_contains "$step_summary" "## testloop-mcp first-run"
assert_contains "$step_summary" 'Status: `passed`'
assert_contains "$step_summary" 'first_run_agent_next_step: `ready`'
assert_contains "$step_summary" "Agent context: \`$output_dir/first-run-context.txt\`"
assert_contains "$step_summary" "Agent response: \`$output_dir/agent-response.txt\`"
assert_contains "$step_summary" 'Artifact verification: `passed`'

failed_output_dir="${tmp_dir}/failed-artifacts"
failed_step_summary="${tmp_dir}/failed-step-summary.md"
run_expect_code 1 "$out" env \
  TESTLOOP_MCP_REPO_DIR="$repo_root" \
  TESTLOOP_MCP_COMMAND="$fake_binary" \
  TESTLOOP_FIRST_RUN_PROJECT_DIR="$project_dir" \
  TESTLOOP_FIRST_RUN_OUTPUT_DIR="$failed_output_dir" \
  TESTLOOP_REPORT_SKIP_BASIC=true \
  TESTLOOP_REPORT_SKIP_PROCESS_SMOKE=true \
  TESTLOOP_REPORT_SKIP_AGENT_DEMO=true \
  GITHUB_STEP_SUMMARY="$failed_step_summary" \
  bash "${repo_root}/scripts/run-first-run-ci.sh" 'echo project failed; exit 7'

assert_contains "$out" "first_run_status=failed"
assert_contains "$out" "first_run_agent_next_step=inspect-user-project"
assert_contains "$out" "agent_artifact_status=passed"
assert_contains "$out" "artifact_kind=first-run"
assert_contains "$out" "response_action=inspect-user-project"
assert_contains "$failed_output_dir/verification-report.md" "project failed"
assert_contains "$failed_output_dir/verification-summary.json" '"action": "manual_review"'
assert_contains "$failed_output_dir/verification-summary.schema.json" '"title": "testloop-mcp verification summary"'
assert_contains "$failed_output_dir/first-run-context.txt" "first_run_agent_next_step=inspect-user-project"
assert_contains "$failed_output_dir/agent-decision.txt" "section_signal=独立 CLI 生成动作 smoke action=manual_review"
assert_contains "$failed_output_dir/agent-response.txt" "结论：testloop-mcp 接入链路本身是通的，失败发生在用户项目 smoke。"
assert_contains "$failed_output_dir/agent-response.txt" "- first_run_status=failed"
assert_contains "$failed_output_dir/agent-response.txt" "- first_run_failed_count=1"
assert_contains "$failed_output_dir/agent-response.txt" "- failed_section=用户项目 smoke"
assert_contains "$failed_output_dir/agent-response.txt" "- exit_code=7"
assert_contains "$failed_output_dir/agent-response.txt" "- section_signal=独立 CLI 生成动作 smoke action=manual_review"
assert_contains "$failed_step_summary" 'Status: `failed`'
assert_contains "$failed_step_summary" 'first_run_agent_next_step: `inspect-user-project`'
assert_contains "$failed_step_summary" "Agent response: \`$failed_output_dir/agent-response.txt\`"
assert_contains "$failed_step_summary" 'Artifact verification: `passed`'

run_expect_code 0 "$out" bash "${repo_root}/scripts/run-first-run-ci.sh" --help
assert_contains "$out" "Usage: scripts/run-first-run-ci.sh [project-smoke-command]"

fake_repo="${tmp_dir}/fake-testloop-repo"
mkdir -p "$fake_repo/scripts"
cat >"$fake_repo/scripts/install.sh" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
mkdir -p "$TESTLOOP_MCP_INSTALL_DIR"
cat > "$TESTLOOP_MCP_INSTALL_DIR/testloop-mcp" <<'BIN'
#!/usr/bin/env sh
echo "testloop-mcp 9.9.9"
BIN
chmod +x "$TESTLOOP_MCP_INSTALL_DIR/testloop-mcp"
SH
chmod +x "$fake_repo/scripts/install.sh"

cat >"$fake_repo/scripts/doctor-first-run.sh" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
binary="${1:?binary required}"
echo "helper_binary=$binary"
"$binary" --version
echo "helper_expect_version=${TESTLOOP_FIRST_RUN_EXPECT_VERSION:-}"
echo "helper_project_dir=${TESTLOOP_FIRST_RUN_PROJECT_DIR:-}"
echo "helper_project_command=${TESTLOOP_FIRST_RUN_PROJECT_COMMAND:-}"
SH
chmod +x "$fake_repo/scripts/doctor-first-run.sh"

install_dir="${tmp_dir}/install-bin"
run_expect_code 0 "$out" env \
  TESTLOOP_MCP_REPO_DIR="$fake_repo" \
  TESTLOOP_MCP_VERSION=v9.9.9 \
  TESTLOOP_MCP_INSTALL_DIR="$install_dir" \
  TESTLOOP_FIRST_RUN_PROJECT_DIR="$project_dir" \
  bash "${repo_root}/scripts/run-first-run-ci.sh" 'echo smoke'

assert_contains "$out" "testloop_mcp_binary=$install_dir/testloop-mcp"
assert_contains "$out" "helper_binary=$install_dir/testloop-mcp"
assert_contains "$out" "testloop-mcp 9.9.9"
assert_contains "$out" "helper_expect_version=9.9.9"
assert_contains "$out" "helper_project_command=echo smoke"

dir_binary_out="${tmp_dir}/dir-binary.out"
run_expect_code 1 "$dir_binary_out" env \
  TESTLOOP_MCP_REPO_DIR="$repo_root" \
  TESTLOOP_MCP_COMMAND="$repo_root" \
  TESTLOOP_FIRST_RUN_PROJECT_DIR="$project_dir" \
  bash "${repo_root}/scripts/run-first-run-ci.sh" 'echo smoke'
assert_contains "$dir_binary_out" "binary must be an executable file"

fake_git_bin="${tmp_dir}/fake-git-bin"
mkdir -p "$fake_git_bin"
cat >"$fake_git_bin/git" <<'SH'
#!/usr/bin/env sh
log="${TEST_FAKE_GIT_LOG:?TEST_FAKE_GIT_LOG required}"
echo "$*" >> "$log"
dest=""
for arg in "$@"; do
  dest="$arg"
done
mkdir -p "$dest/scripts"
cat > "$dest/scripts/install.sh" <<'INSTALL'
#!/usr/bin/env bash
set -euo pipefail
mkdir -p "$TESTLOOP_MCP_INSTALL_DIR"
cat > "$TESTLOOP_MCP_INSTALL_DIR/testloop-mcp" <<'BIN'
#!/usr/bin/env sh
echo "testloop-mcp 8.8.8"
BIN
chmod +x "$TESTLOOP_MCP_INSTALL_DIR/testloop-mcp"
INSTALL
chmod +x "$dest/scripts/install.sh"
cat > "$dest/scripts/doctor-first-run.sh" <<'DOCTOR'
#!/usr/bin/env bash
set -euo pipefail
echo "cloned_helper_expect_version=${TESTLOOP_FIRST_RUN_EXPECT_VERSION:-}"
DOCTOR
chmod +x "$dest/scripts/doctor-first-run.sh"
SH
chmod +x "$fake_git_bin/git"

git_log="${tmp_dir}/fake-git.log"
clone_install_dir="${tmp_dir}/clone-install-bin"
copied_bootstrap="${tmp_dir}/copied-run-first-run-ci.sh"
cp "${repo_root}/scripts/run-first-run-ci.sh" "$copied_bootstrap"
chmod +x "$copied_bootstrap"
run_expect_code 0 "$out" env \
  PATH="$fake_git_bin:$PATH" \
  TEST_FAKE_GIT_LOG="$git_log" \
  TESTLOOP_MCP_REPO_URL=https://example.invalid/testloop-mcp.git \
  TESTLOOP_MCP_VERSION=v8.8.8 \
  TESTLOOP_MCP_INSTALL_DIR="$clone_install_dir" \
  TESTLOOP_FIRST_RUN_PROJECT_DIR="$project_dir" \
  bash "$copied_bootstrap" 'echo smoke'

assert_contains "$git_log" "--branch main"
assert_contains "$out" "testloop_mcp_binary=$clone_install_dir/testloop-mcp"
assert_contains "$out" "cloned_helper_expect_version=8.8.8"

echo "run first-run CI test passed"
