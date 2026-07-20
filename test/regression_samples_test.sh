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

for script in \
  scripts/validate-java-regression-samples.sh \
  scripts/validate-js-regression-samples.sh \
  scripts/validate-py-regression-samples.sh
do
  out="${tmp_dir}/$(basename "$script").help.out"
  run_expect_code 0 "$out" bash "${repo_root}/${script}" --help
  assert_contains "$out" "$script"
done

output_file="${tmp_dir}/output-file"
printf 'not a directory\n' > "$output_file"

out="${tmp_dir}/java.out"
run_expect_code 1 "$out" env TESTLOOP_JAVA_REGRESSION_OUTPUT_DIR="$output_file" bash "${repo_root}/scripts/validate-java-regression-samples.sh"
assert_contains "$out" "output path must be a directory"

out="${tmp_dir}/js.out"
run_expect_code 1 "$out" env TESTLOOP_JS_REGRESSION_OUTPUT_DIR="$output_file" bash "${repo_root}/scripts/validate-js-regression-samples.sh"
assert_contains "$out" "output path must be a directory"

out="${tmp_dir}/py.out"
run_expect_code 1 "$out" env TESTLOOP_PY_REGRESSION_OUTPUT_DIR="$output_file" bash "${repo_root}/scripts/validate-py-regression-samples.sh"
assert_contains "$out" "output path must be a directory"

echo "regression samples test passed"
