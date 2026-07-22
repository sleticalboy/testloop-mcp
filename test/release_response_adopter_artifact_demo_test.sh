#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT INT TERM

cd "$repo_root"

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
  expected="$1"
  out="$2"
  shift 2

  set +e
  "$@" > "$out" 2>&1
  code=$?
  set -e

  if [ "$code" -ne "$expected" ]; then
    echo "expected exit code $expected, got $code: $*" >&2
    echo "--- $out ---" >&2
    cat "$out" >&2
    exit 1
  fi
}

passed="docs/fixtures/release-response-adopter-artifact-verification/passed.json"
failed="docs/fixtures/release-response-adopter-artifact-verification/missing-summary-consumer.json"

passed_out="${tmp_dir}/passed.out"
go run ./examples/release-response-adopter-artifact-demo "$passed" > "$passed_out"
assert_contains "$passed_out" "artifact_verification_status=passed"
assert_contains "$passed_out" "schema_version=1"
assert_contains "$passed_out" "release_ref=v0.5.20"
assert_contains "$passed_out" "fixture_count=8"
assert_contains "$passed_out" "agent_next_step=ready"
assert_contains "$passed_out" "should_accept=true"
assert_contains "$passed_out" "required_files=6"
assert_contains "$passed_out" "missing_files="
assert_contains "$passed_out" "client_decision=accept"

failed_out="${tmp_dir}/failed.out"
go run ./examples/release-response-adopter-artifact-demo "$failed" > "$failed_out"
assert_contains "$failed_out" "artifact_verification_status=failed"
assert_contains "$failed_out" "agent_next_step=inspect-release-response-adopter-artifact"
assert_contains "$failed_out" "should_accept=false"
assert_contains "$failed_out" "missing_files=testloop-release-response-summary-consumer.json"
assert_contains "$failed_out" "client_decision=inspect-artifact"
assert_contains "$failed_out" "failures=missing required file testloop-release-response-summary-consumer.json"

run_expect_code 1 "${tmp_dir}/usage.err" go run ./examples/release-response-adopter-artifact-demo
assert_contains "${tmp_dir}/usage.err" "usage: go run ./examples/release-response-adopter-artifact-demo <artifact-verification.json>"

echo "release response adopter artifact demo test passed"
