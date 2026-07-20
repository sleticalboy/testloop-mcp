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

out="${tmp_dir}/verify.out"
err="${tmp_dir}/verify.err"

sh "${repo_root}/scripts/verify-agent-artifact.sh" \
  first-run \
  "${repo_root}/docs/fixtures/first-run-artifacts/user-project-smoke-failed" > "$out"

assert_contains "$out" "agent_artifact_status=passed"
assert_contains "$out" "artifact_kind=first-run"
assert_contains "$out" "summary_schema=verification-summary.schema.json"
assert_contains "$out" "overall_status=failed"
assert_contains "$out" "failed_count=1"
assert_contains "$out" "decision_action=inspect-user-project"
assert_contains "$out" "response_action=inspect-user-project"
assert_contains "$out" "failed_section=用户项目 smoke"
assert_contains "$out" "exit_code=7"
assert_contains "$out" "section_signal=独立 CLI 生成动作 smoke action=manual_review"
assert_contains "$out" "required_files=7"

sh "${repo_root}/scripts/verify-agent-artifact.sh" \
  onboarding \
  "${repo_root}/docs/fixtures/onboarding-artifacts/user-project-smoke-failed" > "$out"

assert_contains "$out" "agent_artifact_status=passed"
assert_contains "$out" "artifact_kind=onboarding"
assert_contains "$out" "decision_action=inspect-user-project"
assert_contains "$out" "response_action=inspect-user-project"
assert_contains "$out" "required_files=5"

sh "${repo_root}/scripts/verify-agent-artifact.sh" \
  manifest \
  "${repo_root}/docs/fixtures/agent-response-artifact-manifest.json" > "$out"

assert_contains "$out" "agent_artifact_manifest_status=passed"
assert_contains "$out" "manifest_schema_version=1"
assert_contains "$out" "artifact_count=2"
assert_contains "$out" "1. artifact_kind=first-run expected_action=inspect-user-project"
assert_contains "$out" "2. artifact_kind=onboarding expected_action=inspect-user-project"
assert_contains "$out" "required_files=7"
assert_contains "$out" "required_files=5"

sh "${repo_root}/scripts/verify-agent-artifact.sh" \
  --json \
  first-run \
  "${repo_root}/docs/fixtures/first-run-artifacts/user-project-smoke-failed" > "$out"

assert_contains "$out" '"status": "passed"'
assert_contains "$out" '"artifact_kind": "first-run"'
assert_contains "$out" '"response_action": "inspect-user-project"'
assert_contains "$out" '"failed_section": "用户项目 smoke"'
assert_contains "$out" '"required_files": 7'

sh "${repo_root}/scripts/verify-agent-artifact.sh" \
  --json \
  manifest \
  "${repo_root}/docs/fixtures/agent-response-artifact-manifest.json" > "$out"

assert_contains "$out" '"status": "passed"'
assert_contains "$out" '"manifest_schema_version": 1'
assert_contains "$out" '"artifact_count": 2'
assert_contains "$out" '"artifact_kind": "first-run"'
assert_contains "$out" '"artifact_kind": "onboarding"'

run_expect_code 0 "$out" sh "${repo_root}/scripts/verify-agent-artifact.sh" --help
assert_contains "$out" "Usage: scripts/verify-agent-artifact.sh [--json] <first-run|onboarding> <artifact-dir>"
assert_contains "$out" "scripts/verify-agent-artifact.sh [--json] manifest <agent-response-artifact-manifest.json>"

run_expect_code 2 "$err" sh "${repo_root}/scripts/verify-agent-artifact.sh" first-run
assert_contains "$err" "Usage: scripts/verify-agent-artifact.sh [--json] <first-run|onboarding> <artifact-dir>"

bad_missing_schema="${tmp_dir}/missing-schema"
cp -R "${repo_root}/docs/fixtures/onboarding-artifacts/user-project-smoke-failed" "$bad_missing_schema"
rm "$bad_missing_schema/verification-summary.schema.json"
run_expect_code 1 "$err" sh "${repo_root}/scripts/verify-agent-artifact.sh" onboarding "$bad_missing_schema"
assert_contains "$err" "missing required file verification-summary.schema.json"

bad_decision="${tmp_dir}/bad-decision"
cp -R "${repo_root}/docs/fixtures/onboarding-artifacts/user-project-smoke-failed" "$bad_decision"
printf 'agent_next_step=ready\n' > "$bad_decision/agent-decision.txt"
run_expect_code 1 "$err" sh "${repo_root}/scripts/verify-agent-artifact.sh" onboarding "$bad_decision"
assert_contains "$err" 'agent-decision.txt agent_next_step="ready", want "inspect-user-project"'

bad_response="${tmp_dir}/bad-response"
cp -R "${repo_root}/docs/fixtures/onboarding-artifacts/user-project-smoke-failed" "$bad_response"
perl -0pi -e 's/- section_signal=独立 CLI 生成动作 smoke action=manual_review\n//' "$bad_response/agent-response.txt"
run_expect_code 1 "$err" sh "${repo_root}/scripts/verify-agent-artifact.sh" onboarding "$bad_response"
assert_contains "$err" "agent-response.txt missing - section_signal=独立 CLI 生成动作 smoke action=manual_review"

bad_first_run_response="${tmp_dir}/bad-first-run-response"
cp -R "${repo_root}/docs/fixtures/first-run-artifacts/user-project-smoke-failed" "$bad_first_run_response"
perl -0pi -e 's/- first_run_failed_count=1\n//' "$bad_first_run_response/agent-response.txt"
run_expect_code 1 "$err" sh "${repo_root}/scripts/verify-agent-artifact.sh" first-run "$bad_first_run_response"
assert_contains "$err" "agent-response.txt missing first_run_failed_count=1"

bad_manifest="${tmp_dir}/bad-manifest.json"
cp "${repo_root}/docs/fixtures/agent-response-artifact-manifest.json" "$bad_manifest"
perl -0pi -e 's/"expected_action": "inspect-user-project"/"expected_action": "ready"/' "$bad_manifest"
run_expect_code 1 "$err" sh "${repo_root}/scripts/verify-agent-artifact.sh" manifest "$bad_manifest"
assert_contains "$err" 'expected_action="ready", want "inspect-user-project"'

echo "agent artifact verify test passed"
