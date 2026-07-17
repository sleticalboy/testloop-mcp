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
  bash -n "${repo_root}/scripts/showcase-go-public-project.sh"
  bash -n "${repo_root}/scripts/showcase-js-public-project.sh"
}

test_go_showcase_help_and_args() {
  out="${tmp_dir}/go-help.out"
  run_expect_code 0 "$out" bash "${repo_root}/scripts/showcase-go-public-project.sh" --help
  assert_contains "$out" "Usage: scripts/showcase-go-public-project.sh [output-jsonl]"
  assert_contains "$out" "TESTLOOP_SHOWCASE_GO_REF"

  out="${tmp_dir}/go-args.out"
  run_expect_code 2 "$out" bash "${repo_root}/scripts/showcase-go-public-project.sh" one two
  assert_contains "$out" "Usage: scripts/showcase-go-public-project.sh [output-jsonl]"
}

test_js_showcase_help_args_and_missing_pnpm() {
  out="${tmp_dir}/js-help.out"
  run_expect_code 0 "$out" bash "${repo_root}/scripts/showcase-js-public-project.sh" --help
  assert_contains "$out" "Usage: scripts/showcase-js-public-project.sh [output-jsonl]"
  assert_contains "$out" "TESTLOOP_SHOWCASE_JS_REF"

  out="${tmp_dir}/js-args.out"
  run_expect_code 2 "$out" bash "${repo_root}/scripts/showcase-js-public-project.sh" one two
  assert_contains "$out" "Usage: scripts/showcase-js-public-project.sh [output-jsonl]"

  mkdir -p "${tmp_dir}/empty-path"
  out="${tmp_dir}/js-missing-pnpm.out"
  run_expect_code 1 "$out" env PATH="${tmp_dir}/empty-path" "$bash_bin" "${repo_root}/scripts/showcase-js-public-project.sh"
  assert_contains "$out" "error: pnpm is required for this showcase"
}

test_showcase_scripts_are_valid_bash
test_go_showcase_help_and_args
test_js_showcase_help_args_and_missing_pnpm

echo "showcase script tests passed"
