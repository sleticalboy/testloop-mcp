#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT INT TERM
bash_bin="$(command -v bash)"

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

assert_exit_code() {
  want="$1"
  got="$2"
  context="$3"
  if [ "$got" -ne "$want" ]; then
    echo "expected exit code $want, got $got: $context" >&2
    exit 1
  fi
}

run_expect_code() {
  want="$1"
  out="$2"
  shift 2
  set +e
  "$@" > "$out" 2>&1
  code=$?
  set -e
  assert_exit_code "$want" "$code" "$*"
}

test_showcase_scripts_are_valid_bash() {
  bash -n "${repo_root}/scripts/showcase-onboarding.sh"
  bash -n "${repo_root}/scripts/doctor-first-run.sh"
  bash -n "${repo_root}/scripts/run-first-run-ci.sh"
  bash -n "${repo_root}/scripts/showcase-onboarding-ci-external-project.sh"
  bash -n "${repo_root}/scripts/showcase-first-run-ci-external-project.sh"
  bash -n "${repo_root}/scripts/showcase-go-public-project.sh"
  bash -n "${repo_root}/scripts/showcase-js-public-project.sh"
  bash -n "${repo_root}/scripts/showcase-dual-project-report.sh"
  bash -n "${repo_root}/scripts/showcase-laoxia-scaffold-report.sh"
  bash -n "${repo_root}/scripts/showcase-agent-decision-client-ci.sh"
  bash -n "${repo_root}/scripts/install-agent-decision-client-ci-template.sh"
  bash -n "${repo_root}/scripts/showcase-agent-decision-client-ci-template-install.sh"
  python3 -m py_compile "${repo_root}/scripts/summarize-showcase-output.py"
}

test_onboarding_showcase_help_and_args() {
  out="${tmp_dir}/onboarding-help.out"
  run_expect_code 0 "$out" bash "${repo_root}/scripts/showcase-onboarding.sh" --help
  assert_contains "$out" "Usage: scripts/showcase-onboarding.sh [testloop-mcp-binary]"
  assert_contains "$out" "TESTLOOP_MCP_VERIFY_EXPECT_VERSION"

  out="${tmp_dir}/onboarding-args.out"
  run_expect_code 2 "$out" bash "${repo_root}/scripts/showcase-onboarding.sh" one two
  assert_contains "$out" "Usage: scripts/showcase-onboarding.sh [testloop-mcp-binary]"
}

test_external_onboarding_help_and_args() {
  out="${tmp_dir}/external-onboarding-help.out"
  run_expect_code 0 "$out" bash "${repo_root}/scripts/showcase-onboarding-ci-external-project.sh" --help
  assert_contains "$out" "Usage: scripts/showcase-onboarding-ci-external-project.sh"
  assert_contains "$out" "TESTLOOP_EXTERNAL_ONBOARDING_WORKDIR"
  assert_contains "$out" "TESTLOOP_EXTERNAL_ONBOARDING_PROJECT_TYPE"
  assert_contains "$out" "TESTLOOP_MCP_COMMAND"

  out="${tmp_dir}/external-onboarding-args.out"
  run_expect_code 2 "$out" bash "${repo_root}/scripts/showcase-onboarding-ci-external-project.sh" extra
  assert_contains "$out" "Usage: scripts/showcase-onboarding-ci-external-project.sh"

  out="${tmp_dir}/external-onboarding-type.out"
  run_expect_code 1 "$out" env TESTLOOP_EXTERNAL_ONBOARDING_PROJECT_TYPE=bad bash "${repo_root}/scripts/showcase-onboarding-ci-external-project.sh"
  assert_contains "$out" "unsupported TESTLOOP_EXTERNAL_ONBOARDING_PROJECT_TYPE"

  output_file="${tmp_dir}/external-onboarding-output-file"
  printf 'not a directory\n' > "$output_file"
  out="${tmp_dir}/external-onboarding-output-file.out"
  run_expect_code 1 "$out" env \
    TESTLOOP_EXTERNAL_ONBOARDING_WORKDIR="${tmp_dir}/external-onboarding-workdir" \
    TESTLOOP_EXTERNAL_ONBOARDING_OUTPUT_DIR="$output_file" \
    bash "${repo_root}/scripts/showcase-onboarding-ci-external-project.sh"
  assert_contains "$out" "output path must be a directory"
}

test_external_first_run_help_and_args() {
  out="${tmp_dir}/external-first-run-help.out"
  run_expect_code 0 "$out" bash "${repo_root}/scripts/showcase-first-run-ci-external-project.sh" --help
  assert_contains "$out" "Usage: scripts/showcase-first-run-ci-external-project.sh"
  assert_contains "$out" "TESTLOOP_EXTERNAL_FIRST_RUN_WORKDIR"
  assert_contains "$out" "TESTLOOP_EXTERNAL_FIRST_RUN_PROJECT_TYPE"
  assert_contains "$out" "TESTLOOP_MCP_COMMAND"

  out="${tmp_dir}/external-first-run-args.out"
  run_expect_code 2 "$out" bash "${repo_root}/scripts/showcase-first-run-ci-external-project.sh" extra
  assert_contains "$out" "Usage: scripts/showcase-first-run-ci-external-project.sh"

  out="${tmp_dir}/external-first-run-type.out"
  run_expect_code 1 "$out" env TESTLOOP_EXTERNAL_FIRST_RUN_PROJECT_TYPE=bad bash "${repo_root}/scripts/showcase-first-run-ci-external-project.sh"
  assert_contains "$out" "unsupported TESTLOOP_EXTERNAL_FIRST_RUN_PROJECT_TYPE"

  output_file="${tmp_dir}/external-first-run-output-file"
  printf 'not a directory\n' > "$output_file"
  out="${tmp_dir}/external-first-run-output-file.out"
  run_expect_code 1 "$out" env \
    TESTLOOP_EXTERNAL_FIRST_RUN_WORKDIR="${tmp_dir}/external-first-run-workdir" \
    TESTLOOP_EXTERNAL_FIRST_RUN_OUTPUT_DIR="$output_file" \
    bash "${repo_root}/scripts/showcase-first-run-ci-external-project.sh"
  assert_contains "$out" "output path must be a directory"
}

test_doctor_first_run_help_and_args() {
  out="${tmp_dir}/doctor-first-run-help.out"
  run_expect_code 0 "$out" bash "${repo_root}/scripts/doctor-first-run.sh" --help
  assert_contains "$out" "Usage: scripts/doctor-first-run.sh [testloop-mcp-binary]"
  assert_contains "$out" "TESTLOOP_FIRST_RUN_OUTPUT_DIR"
  assert_contains "$out" "first_run_agent_next_step"

  out="${tmp_dir}/doctor-first-run-args.out"
  run_expect_code 2 "$out" bash "${repo_root}/scripts/doctor-first-run.sh" one two
  assert_contains "$out" "Usage: scripts/doctor-first-run.sh [testloop-mcp-binary]"
}

test_run_first_run_ci_help_and_args() {
  out="${tmp_dir}/run-first-run-ci-help.out"
  run_expect_code 0 "$out" bash "${repo_root}/scripts/run-first-run-ci.sh" --help
  assert_contains "$out" "Usage: scripts/run-first-run-ci.sh [project-smoke-command]"
  assert_contains "$out" "TESTLOOP_FIRST_RUN_OUTPUT_DIR"
  assert_contains "$out" "first-run-context.txt"

  out="${tmp_dir}/run-first-run-ci-args.out"
  run_expect_code 2 "$out" bash "${repo_root}/scripts/run-first-run-ci.sh" one two
  assert_contains "$out" "Usage: scripts/run-first-run-ci.sh [project-smoke-command]"
}

test_go_showcase_help_and_args() {
  out="${tmp_dir}/go-help.out"
  run_expect_code 0 "$out" bash "${repo_root}/scripts/showcase-go-public-project.sh" --help
  assert_contains "$out" "Usage: scripts/showcase-go-public-project.sh [output-jsonl]"
  assert_contains "$out" "TESTLOOP_SHOWCASE_GO_REF"
  assert_contains "$out" "TESTLOOP_SHOWCASE_GO_PROJECT_DIR"
  assert_contains "$out" "TESTLOOP_SHOWCASE_GO_EXPECT_ACTIONS"
  assert_contains "$out" "TESTLOOP_SHOWCASE_GO_GIT_TIMEOUT"

  out="${tmp_dir}/go-args.out"
  run_expect_code 2 "$out" bash "${repo_root}/scripts/showcase-go-public-project.sh" one two
  assert_contains "$out" "Usage: scripts/showcase-go-public-project.sh [output-jsonl]"

  out_dir="${tmp_dir}/go-output-dir"
  mkdir -p "$out_dir"
  out="${tmp_dir}/go-output-dir.out"
  run_expect_code 1 "$out" bash "${repo_root}/scripts/showcase-go-public-project.sh" "$out_dir"
  assert_contains "$out" "output path must not be a directory"
}

test_go_showcase_git_timeout() {
  fake_bin="${tmp_dir}/go-fake-bin"
  mkdir -p "$fake_bin"
  cat > "${fake_bin}/git" <<'SH'
#!/usr/bin/env sh
sleep 5
SH
  chmod +x "${fake_bin}/git"

  out="${tmp_dir}/go-git-timeout.out"
  run_expect_code 124 "$out" env PATH="${fake_bin}:$PATH" TESTLOOP_SHOWCASE_GO_GIT_TIMEOUT=0.1 "$bash_bin" "${repo_root}/scripts/showcase-go-public-project.sh"
  assert_contains "$out" "error: command timed out after 0.1s: git clone"
}

test_js_showcase_help_args_and_missing_pnpm() {
  out="${tmp_dir}/js-help.out"
  run_expect_code 0 "$out" bash "${repo_root}/scripts/showcase-js-public-project.sh" --help
  assert_contains "$out" "Usage: scripts/showcase-js-public-project.sh [output-jsonl]"
  assert_contains "$out" "TESTLOOP_SHOWCASE_JS_REF"
  assert_contains "$out" "TESTLOOP_SHOWCASE_JS_PROJECT_DIR"
  assert_contains "$out" "TESTLOOP_SHOWCASE_JS_EXPECT_ACTIONS"
  assert_contains "$out" "TESTLOOP_SHOWCASE_JS_GIT_TIMEOUT"
  assert_contains "$out" "TESTLOOP_SHOWCASE_JS_SKIP_INSTALL"

  out="${tmp_dir}/js-args.out"
  run_expect_code 2 "$out" bash "${repo_root}/scripts/showcase-js-public-project.sh" one two
  assert_contains "$out" "Usage: scripts/showcase-js-public-project.sh [output-jsonl]"

  out_dir="${tmp_dir}/js-output-dir"
  mkdir -p "$out_dir"
  out="${tmp_dir}/js-output-dir.out"
  run_expect_code 1 "$out" bash "${repo_root}/scripts/showcase-js-public-project.sh" "$out_dir"
  assert_contains "$out" "output path must not be a directory"

  mkdir -p "${tmp_dir}/empty-path"
  out="${tmp_dir}/js-missing-pnpm.out"
  run_expect_code 1 "$out" env PATH="${tmp_dir}/empty-path" "$bash_bin" "${repo_root}/scripts/showcase-js-public-project.sh"
  assert_contains "$out" "error: pnpm is required for this showcase"
}

test_js_showcase_git_timeout() {
  fake_bin="${tmp_dir}/js-fake-bin"
  mkdir -p "$fake_bin"
  cat > "${fake_bin}/git" <<'SH'
#!/usr/bin/env sh
sleep 5
SH
  cat > "${fake_bin}/pnpm" <<'SH'
#!/usr/bin/env sh
exit 0
SH
  chmod +x "${fake_bin}/git" "${fake_bin}/pnpm"

  out="${tmp_dir}/js-git-timeout.out"
  run_expect_code 124 "$out" env PATH="${fake_bin}:$PATH" TESTLOOP_SHOWCASE_JS_GIT_TIMEOUT=0.1 "$bash_bin" "${repo_root}/scripts/showcase-js-public-project.sh"
  assert_contains "$out" "error: command timed out after 0.1s: git clone"
}

test_laoxia_scaffold_showcase_help_args_and_run() {
  out="${tmp_dir}/laoxia-help.out"
  run_expect_code 0 "$out" bash "${repo_root}/scripts/showcase-laoxia-scaffold-report.sh" --help
  assert_contains "$out" "Usage: scripts/showcase-laoxia-scaffold-report.sh [testloop-mcp-binary]"
  assert_contains "$out" "TESTLOOP_LAOXIA_OUTPUT_DIR"
  assert_contains "$out" "TESTLOOP_LAOXIA_WEB_COMMAND"

  out="${tmp_dir}/laoxia-args.out"
  run_expect_code 2 "$out" bash "${repo_root}/scripts/showcase-laoxia-scaffold-report.sh" one two
  assert_contains "$out" "Usage: scripts/showcase-laoxia-scaffold-report.sh [testloop-mcp-binary]"

  fake_binary="${tmp_dir}/laoxia-fake-bin"
  cat > "$fake_binary" <<'SH'
#!/usr/bin/env sh
case "${1:-}" in
  --version)
    echo "testloop-mcp 0.5.16"
    ;;
  *)
    echo "fake testloop-mcp"
    ;;
esac
SH
  chmod +x "$fake_binary"

  laoxia_root="${tmp_dir}/laoxia-root"
  server_dir="${laoxia_root}/car-admin-server"
  web_dir="${laoxia_root}/car-admin-web"
  mkdir -p "$server_dir" "$web_dir"

  success_out="${tmp_dir}/laoxia-success.out"
  output_dir="${tmp_dir}/laoxia-artifacts"
  run_expect_code 0 "$success_out" env \
    TESTLOOP_LAOXIA_OUTPUT_DIR="$output_dir" \
    TESTLOOP_LAOXIA_SERVER_DIR="$server_dir" \
    TESTLOOP_LAOXIA_WEB_DIR="$web_dir" \
    TESTLOOP_LAOXIA_SERVER_COMMAND='printf "server smoke ok\n"' \
    TESTLOOP_LAOXIA_WEB_COMMAND='printf "web smoke ok\n"' \
    TESTLOOP_REPORT_SKIP_BASIC=true \
    TESTLOOP_REPORT_SKIP_PROCESS_SMOKE=true \
    TESTLOOP_REPORT_SKIP_AGENT_DEMO=true \
    TESTLOOP_REPORT_SKIP_TESTGEN_SMOKE=true \
    bash "${repo_root}/scripts/showcase-laoxia-scaffold-report.sh" "$fake_binary"

  assert_contains "$success_out" "laoxia_output_dir=$output_dir"
  assert_contains "$success_out" "laoxia_summary_json=$output_dir/laoxia-summary.json"
  assert_contains "$success_out" "laoxia_summary_schema=$output_dir/dual-project-summary.schema.json"
  assert_contains "$success_out" "laoxia_server_report=$output_dir/server/verification-report.md"
  assert_contains "$success_out" "laoxia_server_summary=$output_dir/server/verification-summary.json"
  assert_contains "$success_out" "laoxia_server_status=passed"
  assert_contains "$success_out" "laoxia_web_report=$output_dir/web/verification-report.md"
  assert_contains "$success_out" "laoxia_web_summary=$output_dir/web/verification-summary.json"
  assert_contains "$success_out" "laoxia_web_status=passed"
  assert_contains "$success_out" "laoxia_status=passed"
  assert_contains "$output_dir/laoxia-summary.json" '"overall_status": "passed"'
  assert_contains "$output_dir/laoxia-summary.json" '"failed_count": 0'
  assert_contains "$output_dir/laoxia-summary.json" '"server": {'
  assert_contains "$output_dir/laoxia-summary.json" '"web": {'
  assert_contains "$output_dir/dual-project-summary.schema.json" '"title": "testloop-mcp dual project summary"'
  python3 - "$output_dir/laoxia-summary.json" <<'PY'
import json
import sys
from pathlib import Path

data = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
if data["server"]["summary"]["overall_status"] != "passed":
    raise SystemExit("expected server summary overall_status=passed")
if data["web"]["summary"]["overall_status"] != "passed":
    raise SystemExit("expected web summary overall_status=passed")
if data["server"]["summary"]["failed_count"] != 0:
    raise SystemExit("expected server summary failed_count=0")
if data["web"]["summary"]["failed_count"] != 0:
    raise SystemExit("expected web summary failed_count=0")
PY
  assert_contains "$output_dir/server/verification-report.md" "server smoke ok"
  assert_contains "$output_dir/web/verification-report.md" "web smoke ok"
  assert_contains "$output_dir/server/verification-summary.json" '"overall_status": "passed"'
  assert_contains "$output_dir/web/verification-summary.json" '"overall_status": "passed"'

  failed_output_dir="${tmp_dir}/laoxia-failed-artifacts"
  failure_out="${tmp_dir}/laoxia-failure.out"
  run_expect_code 1 "$failure_out" env \
    TESTLOOP_LAOXIA_OUTPUT_DIR="$failed_output_dir" \
    TESTLOOP_LAOXIA_SERVER_DIR="$server_dir" \
    TESTLOOP_LAOXIA_WEB_DIR="$web_dir" \
    TESTLOOP_LAOXIA_SERVER_COMMAND='printf "server smoke ok\n"' \
    TESTLOOP_LAOXIA_WEB_COMMAND='echo web failed; exit 7' \
    TESTLOOP_REPORT_SKIP_BASIC=true \
    TESTLOOP_REPORT_SKIP_PROCESS_SMOKE=true \
    TESTLOOP_REPORT_SKIP_AGENT_DEMO=true \
    TESTLOOP_REPORT_SKIP_TESTGEN_SMOKE=true \
    bash "${repo_root}/scripts/showcase-laoxia-scaffold-report.sh" "$fake_binary"

  assert_contains "$failure_out" "laoxia_server_status=passed"
  assert_contains "$failure_out" "laoxia_web_status=failed"
  assert_contains "$failure_out" "laoxia_status=failed"
  assert_contains "$failed_output_dir/laoxia-summary.json" '"overall_status": "failed"'
  assert_contains "$failed_output_dir/laoxia-summary.json" '"failed_count": 1'
  assert_contains "$failed_output_dir/web/verification-report.md" "web failed"
  assert_contains "$failed_output_dir/web/verification-summary.json" '"overall_status": "failed"'
}

test_dual_project_showcase_helper_directly() {
  fake_binary="${tmp_dir}/dual-fake-bin"
  cat > "$fake_binary" <<'SH'
#!/usr/bin/env sh
case "${1:-}" in
  --version)
    echo "testloop-mcp 0.5.16"
    ;;
  *)
    echo "fake testloop-mcp"
    ;;
esac
SH
  chmod +x "$fake_binary"

  pair_root="${tmp_dir}/pair-root"
  api_dir="${pair_root}/api"
  web_dir="${pair_root}/web"
  mkdir -p "$api_dir" "$web_dir"

  project_file="${tmp_dir}/pair-project-file"
  printf 'not a directory\n' > "$project_file"
  project_file_out="${tmp_dir}/pair-project-file.out"
  run_expect_code 1 "$project_file_out" env \
    TESTLOOP_PAIR_FIRST_DIR="$project_file" \
    TESTLOOP_PAIR_FIRST_COMMAND='echo api' \
    TESTLOOP_PAIR_SECOND_DIR="$web_dir" \
    TESTLOOP_PAIR_SECOND_COMMAND='echo web' \
    bash "${repo_root}/scripts/showcase-dual-project-report.sh" "$fake_binary"
  assert_contains "$project_file_out" "first project path must be a directory"

  output_file="${tmp_dir}/pair-output-file"
  printf 'not a directory\n' > "$output_file"
  output_file_out="${tmp_dir}/pair-output-file.out"
  run_expect_code 1 "$output_file_out" env \
    TESTLOOP_PAIR_OUTPUT_DIR="$output_file" \
    TESTLOOP_PAIR_FIRST_DIR="$api_dir" \
    TESTLOOP_PAIR_FIRST_COMMAND='echo api' \
    TESTLOOP_PAIR_SECOND_DIR="$web_dir" \
    TESTLOOP_PAIR_SECOND_COMMAND='echo web' \
    bash "${repo_root}/scripts/showcase-dual-project-report.sh" "$fake_binary"
  assert_contains "$output_file_out" "output path must be a directory"

  summary_dir="${tmp_dir}/pair-summary-dir"
  mkdir -p "$summary_dir"
  summary_dir_out="${tmp_dir}/pair-summary-dir.out"
  run_expect_code 1 "$summary_dir_out" env \
    TESTLOOP_PAIR_SUMMARY_JSON="$summary_dir" \
    TESTLOOP_PAIR_FIRST_DIR="$api_dir" \
    TESTLOOP_PAIR_FIRST_COMMAND='echo api' \
    TESTLOOP_PAIR_SECOND_DIR="$web_dir" \
    TESTLOOP_PAIR_SECOND_COMMAND='echo web' \
    bash "${repo_root}/scripts/showcase-dual-project-report.sh" "$fake_binary"
  assert_contains "$summary_dir_out" "summary JSON path must not be a directory"

  success_out="${tmp_dir}/pair-success.out"
  output_dir="${tmp_dir}/pair-artifacts"
  run_expect_code 0 "$success_out" env \
    TESTLOOP_PAIR_PREFIX=pair \
    TESTLOOP_PAIR_OUTPUT_DIR="$output_dir" \
    TESTLOOP_PAIR_FIRST_NAME=api \
    TESTLOOP_PAIR_FIRST_DIR="$api_dir" \
    TESTLOOP_PAIR_FIRST_COMMAND='printf "api smoke ok\n"' \
    TESTLOOP_PAIR_SECOND_NAME=web \
    TESTLOOP_PAIR_SECOND_DIR="$web_dir" \
    TESTLOOP_PAIR_SECOND_COMMAND='printf "web smoke ok\n"' \
    TESTLOOP_REPORT_SKIP_BASIC=true \
    TESTLOOP_REPORT_SKIP_PROCESS_SMOKE=true \
    TESTLOOP_REPORT_SKIP_AGENT_DEMO=true \
    TESTLOOP_REPORT_SKIP_TESTGEN_SMOKE=true \
    bash "${repo_root}/scripts/showcase-dual-project-report.sh" "$fake_binary"

  assert_contains "$success_out" "pair_output_dir=$output_dir"
  assert_contains "$success_out" "pair_summary_json=$output_dir/pair-summary.json"
  assert_contains "$success_out" "pair_summary_schema=$output_dir/dual-project-summary.schema.json"
  assert_contains "$success_out" "pair_api_report=$output_dir/api/verification-report.md"
  assert_contains "$success_out" "pair_api_status=passed"
  assert_contains "$success_out" "pair_web_report=$output_dir/web/verification-report.md"
  assert_contains "$success_out" "pair_web_status=passed"
  assert_contains "$success_out" "pair_status=passed"
  assert_contains "$output_dir/pair-summary.json" '"overall_status": "passed"'
  assert_contains "$output_dir/pair-summary.json" '"failed_count": 0'
  assert_contains "$output_dir/pair-summary.json" '"api": {'
  assert_contains "$output_dir/pair-summary.json" '"web": {'
  assert_contains "$output_dir/dual-project-summary.schema.json" '"title": "testloop-mcp dual project summary"'

  python3 - "$output_dir/pair-summary.json" <<'PY'
import json
import sys
from pathlib import Path

data = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
if data["api"]["summary"]["overall_status"] != "passed":
    raise SystemExit("expected api summary overall_status=passed")
if data["web"]["summary"]["overall_status"] != "passed":
    raise SystemExit("expected web summary overall_status=passed")
if data["api"]["summary"]["failed_count"] != 0:
    raise SystemExit("expected api summary failed_count=0")
if data["web"]["summary"]["failed_count"] != 0:
    raise SystemExit("expected web summary failed_count=0")
PY
  assert_contains "$output_dir/api/verification-report.md" "api smoke ok"
  assert_contains "$output_dir/web/verification-report.md" "web smoke ok"

  failure_out="${tmp_dir}/pair-failure.out"
  failed_output_dir="${tmp_dir}/pair-failed-artifacts"
  run_expect_code 1 "$failure_out" env \
    TESTLOOP_PAIR_PREFIX=pair \
    TESTLOOP_PAIR_OUTPUT_DIR="$failed_output_dir" \
    TESTLOOP_PAIR_FIRST_NAME=api \
    TESTLOOP_PAIR_FIRST_DIR="$api_dir" \
    TESTLOOP_PAIR_FIRST_COMMAND='printf "api smoke ok\n"' \
    TESTLOOP_PAIR_SECOND_NAME=web \
    TESTLOOP_PAIR_SECOND_DIR="$web_dir" \
    TESTLOOP_PAIR_SECOND_COMMAND='echo web failed; exit 7' \
    TESTLOOP_REPORT_SKIP_BASIC=true \
    TESTLOOP_REPORT_SKIP_PROCESS_SMOKE=true \
    TESTLOOP_REPORT_SKIP_AGENT_DEMO=true \
    TESTLOOP_REPORT_SKIP_TESTGEN_SMOKE=true \
    bash "${repo_root}/scripts/showcase-dual-project-report.sh" "$fake_binary"

  assert_contains "$failure_out" "pair_api_status=passed"
  assert_contains "$failure_out" "pair_web_status=failed"
  assert_contains "$failure_out" "pair_status=failed"
  assert_contains "$failed_output_dir/pair-summary.json" '"overall_status": "failed"'
  assert_contains "$failed_output_dir/pair-summary.json" '"failed_count": 1'
  assert_contains "$failed_output_dir/dual-project-summary.schema.json" '"title": "testloop-mcp dual project summary"'
  assert_contains "$failed_output_dir/web/verification-report.md" "web failed"

  both_failed_out="${tmp_dir}/pair-both-failed.out"
  both_failed_output_dir="${tmp_dir}/pair-both-failed-artifacts"
  run_expect_code 1 "$both_failed_out" env \
    TESTLOOP_PAIR_PREFIX=pair \
    TESTLOOP_PAIR_OUTPUT_DIR="$both_failed_output_dir" \
    TESTLOOP_PAIR_FIRST_NAME=api \
    TESTLOOP_PAIR_FIRST_DIR="$api_dir" \
    TESTLOOP_PAIR_FIRST_COMMAND='echo api failed; exit 5' \
    TESTLOOP_PAIR_SECOND_NAME=web \
    TESTLOOP_PAIR_SECOND_DIR="$web_dir" \
    TESTLOOP_PAIR_SECOND_COMMAND='echo web failed; exit 7' \
    TESTLOOP_REPORT_SKIP_BASIC=true \
    TESTLOOP_REPORT_SKIP_PROCESS_SMOKE=true \
    TESTLOOP_REPORT_SKIP_AGENT_DEMO=true \
    TESTLOOP_REPORT_SKIP_TESTGEN_SMOKE=true \
    bash "${repo_root}/scripts/showcase-dual-project-report.sh" "$fake_binary"

  assert_contains "$both_failed_out" "pair_api_status=failed"
  assert_contains "$both_failed_out" "pair_web_status=failed"
  assert_contains "$both_failed_out" "pair_status=failed"
  assert_contains "$both_failed_output_dir/pair-summary.json" '"overall_status": "failed"'
  assert_contains "$both_failed_output_dir/pair-summary.json" '"failed_count": 2'
  assert_contains "$both_failed_output_dir/dual-project-summary.schema.json" '"title": "testloop-mcp dual project summary"'
  assert_contains "$both_failed_output_dir/api/verification-report.md" "api failed"
  assert_contains "$both_failed_output_dir/web/verification-report.md" "web failed"
}

test_laoxia_scaffold_showcase_rejects_directory_summary_json() {
  fake_binary="${tmp_dir}/laoxia-summary-fake-bin"
  cat > "$fake_binary" <<'SH'
#!/usr/bin/env sh
case "${1:-}" in
  --version)
    echo "testloop-mcp 0.5.16"
    ;;
  *)
    echo "fake testloop-mcp"
    ;;
esac
SH
  chmod +x "$fake_binary"

  laoxia_root="${tmp_dir}/laoxia-summary-root"
  server_dir="${laoxia_root}/car-admin-server"
  web_dir="${laoxia_root}/car-admin-web"
  summary_dir="${tmp_dir}/laoxia-summary-dir"
  mkdir -p "$server_dir" "$web_dir" "$summary_dir"

  out="${tmp_dir}/laoxia-summary-dir.out"
  run_expect_code 1 "$out" env \
    TESTLOOP_LAOXIA_SUMMARY_JSON="$summary_dir" \
    TESTLOOP_LAOXIA_SERVER_DIR="$server_dir" \
    TESTLOOP_LAOXIA_WEB_DIR="$web_dir" \
    TESTLOOP_LAOXIA_SERVER_COMMAND='echo server' \
    TESTLOOP_LAOXIA_WEB_COMMAND='echo web' \
    bash "${repo_root}/scripts/showcase-laoxia-scaffold-report.sh" "$fake_binary"
  assert_contains "$out" "summary JSON path must not be a directory"
}

test_dual_project_showcase_rejects_directory_binary_path() {
  out="${tmp_dir}/pair-binary-dir.out"
  run_expect_code 1 "$out" bash "${repo_root}/scripts/showcase-dual-project-report.sh" "$repo_root"
  assert_contains "$out" "binary must be an executable file"
}

test_showcase_scripts_are_valid_bash
test_onboarding_showcase_help_and_args
test_doctor_first_run_help_and_args
test_run_first_run_ci_help_and_args
test_external_onboarding_help_and_args
test_external_first_run_help_and_args
test_go_showcase_help_and_args
test_go_showcase_git_timeout
test_js_showcase_help_args_and_missing_pnpm
test_js_showcase_git_timeout
test_dual_project_showcase_helper_directly
test_laoxia_scaffold_showcase_rejects_directory_summary_json
test_laoxia_scaffold_showcase_help_args_and_run

echo "showcase script tests passed"
