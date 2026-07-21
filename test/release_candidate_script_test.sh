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

script="${repo_root}/scripts/verify-release-candidate.sh"

bash -n "$script"

help_out="${tmp_dir}/help.out"
run_expect_code 0 "$help_out" bash "$script" --help
assert_contains "$help_out" "Usage: scripts/verify-release-candidate.sh TAG"
assert_contains "$help_out" "TESTLOOP_RELEASE_CANDIDATE_DIST_DIR"

missing_out="${tmp_dir}/missing.out"
run_expect_code 2 "$missing_out" bash "$script"
assert_contains "$missing_out" "Usage: scripts/verify-release-candidate.sh TAG"

bad_tag_out="${tmp_dir}/bad-tag.out"
run_expect_code 2 "$bad_tag_out" bash "$script" 0.5.14
assert_contains "$bad_tag_out" "TAG must look like vMAJOR.MINOR.PATCH"

assert_contains "$script" "find scripts test -name '*.sh' -print0 | xargs -0 -n1 bash -n"
assert_contains "$script" "require_command node"
assert_contains "$script" "require_command npm"
assert_contains "$script" "go test ./..."
assert_contains "$script" "find test -maxdepth 1 -name '*_test.sh'"
assert_contains "$script" "verify agent decision fixture export package"
assert_contains "$script" 'node scripts/export-agent-decision-fixtures.mjs "$agent_decision_fixture_dir"'
assert_contains "$script" 'npm test --silent > "$agent_decision_fixture_json"'
assert_contains "$script" "verify agent decision release response client export package"
assert_contains "$script" 'node scripts/export-agent-decision-release-response-client.mjs "$agent_decision_release_response_client_dir"'
assert_contains "$script" '(cd "$agent_decision_release_response_client_dir" && npm test --silent)'
assert_contains "$script" 'go build -o "$mcp_binary" .'
assert_contains "$script" 'go build -o "$testgen_binary" ./cmd/testgen'
assert_contains "$script" '"$mcp_binary" --version'
assert_contains "$script" '"$mcp_binary" --help'
assert_contains "$script" '"$testgen_binary" --help'
assert_contains "$script" 'scripts/package-release-asset.sh "$tag" "$asset" "$goos" "$goarch"'
assert_contains "$script" 'checksum_check "$checksum_file"'
assert_contains "$script" 'verify_archive_contents "$archive"'
assert_contains "$script" "git diff --check"
assert_contains "$script" "release_candidate_status=passed"

echo "release candidate script test passed"
