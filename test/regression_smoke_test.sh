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

out="${tmp_dir}/help.out"
run_expect_code 0 "$out" bash "${repo_root}/scripts/validate-regression-smoke.sh" --help
assert_contains "$out" "scripts/validate-regression-smoke.sh"
assert_contains "$out" "TESTLOOP_REGRESSION_OUTPUT_DIR"

out="${tmp_dir}/args.out"
run_expect_code 2 "$out" bash "${repo_root}/scripts/validate-regression-smoke.sh" extra
assert_contains "$out" "scripts/validate-regression-smoke.sh"

output_file="${tmp_dir}/output-file"
printf 'not a directory\n' > "$output_file"
out="${tmp_dir}/output-file.out"
run_expect_code 1 "$out" env \
  TESTLOOP_REGRESSION_OUTPUT_DIR="$output_file" \
  TESTLOOP_REGRESSION_SKIP_PREFLIGHT=true \
  TESTLOOP_REGRESSION_SKIP_JAVA=true \
  TESTLOOP_REGRESSION_SKIP_JS=true \
  TESTLOOP_REGRESSION_SKIP_PY=true \
  bash "${repo_root}/scripts/validate-regression-smoke.sh"
assert_contains "$out" "output path must be a directory"

output_dir="${tmp_dir}/artifacts"
out="${tmp_dir}/skipped.out"
run_expect_code 0 "$out" env \
  TESTLOOP_REGRESSION_OUTPUT_DIR="$output_dir" \
  TESTLOOP_REGRESSION_SKIP_PREFLIGHT=true \
  TESTLOOP_REGRESSION_SKIP_JAVA=true \
  TESTLOOP_REGRESSION_SKIP_JS=true \
  TESTLOOP_REGRESSION_SKIP_PY=true \
  bash "${repo_root}/scripts/validate-regression-smoke.sh"
assert_contains "$out" "regression_smoke_output_dir=$output_dir"

echo "regression smoke test passed"
