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

output_dir="${tmp_dir}/output-dir"
mkdir -p "$output_dir"
project_dir="${tmp_dir}/project"
mkdir -p "$project_dir"

out="${tmp_dir}/go-help.out"
run_expect_code 0 "$out" bash "${repo_root}/scripts/validate-go-coverage-top-tasks.sh" --help
assert_contains "$out" "TESTLOOP_VALIDATE_GO_COVERPROFILE"

out="${tmp_dir}/js-help.out"
run_expect_code 0 "$out" bash "${repo_root}/scripts/validate-js-coverage-top-tasks.sh" --help
assert_contains "$out" "TESTLOOP_VALIDATE_JS_COVERAGE_FILE"

out="${tmp_dir}/py-help.out"
run_expect_code 0 "$out" bash "${repo_root}/scripts/validate-py-coverage-top-tasks.sh" --help
assert_contains "$out" "TESTLOOP_VALIDATE_PY_COVERAGE_FILE"

out="${tmp_dir}/go.out"
run_expect_code 1 "$out" bash "${repo_root}/scripts/validate-go-coverage-top-tasks.sh" "$project_dir" 1 "$output_dir"
assert_contains "$out" "output path must not be a directory"

out="${tmp_dir}/js.out"
run_expect_code 1 "$out" bash "${repo_root}/scripts/validate-js-coverage-top-tasks.sh" "$project_dir" vitest 1 "$output_dir"
assert_contains "$out" "output path must not be a directory"

out="${tmp_dir}/java.out"
run_expect_code 1 "$out" bash "${repo_root}/scripts/validate-java-coverage-top-tasks.sh" "$project_dir" 1 "$output_dir"
assert_contains "$out" "output path must not be a directory"

out="${tmp_dir}/py.out"
run_expect_code 1 "$out" bash "${repo_root}/scripts/validate-py-coverage-top-tasks.sh" "$project_dir" 1 "$output_dir"
assert_contains "$out" "output path must not be a directory"

echo "validate coverage top tasks output test passed"
