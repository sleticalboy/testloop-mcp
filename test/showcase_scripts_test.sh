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
  bash -n "${repo_root}/scripts/showcase-go-public-project.sh"
  bash -n "${repo_root}/scripts/showcase-js-public-project.sh"
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

test_showcase_scripts_are_valid_bash
test_onboarding_showcase_help_and_args
test_go_showcase_help_and_args
test_go_showcase_git_timeout
test_js_showcase_help_args_and_missing_pnpm
test_js_showcase_git_timeout

echo "showcase script tests passed"
