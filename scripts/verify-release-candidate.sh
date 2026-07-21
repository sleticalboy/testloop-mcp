#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: scripts/verify-release-candidate.sh TAG

Runs the local release-candidate readiness gates used before cutting a tag.
The script does not update versions, create tags, publish GitHub Releases, or
touch Homebrew taps.

Example:
  scripts/verify-release-candidate.sh v0.5.15

Environment:
  TESTLOOP_RELEASE_CANDIDATE_TMP_DIR   Directory for candidate binaries. Default: /tmp
  TESTLOOP_RELEASE_CANDIDATE_DIST_DIR  Release asset dry-run directory. Default: /tmp/testloop-<tag>-candidate-dist
  TESTLOOP_RELEASE_CANDIDATE_ASSET     Asset suffix. Default: darwin_arm64
  TESTLOOP_RELEASE_CANDIDATE_GOOS      Build GOOS for the packaged asset. Default: darwin
  TESTLOOP_RELEASE_CANDIDATE_GOARCH    Build GOARCH for the packaged asset. Default: arm64
USAGE
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

if [[ "$#" -ne 1 ]]; then
  usage >&2
  exit 2
fi

tag="$1"
case "$tag" in
  v[0-9]*.[0-9]*.[0-9]*)
    ;;
  *)
    echo "error: TAG must look like vMAJOR.MINOR.PATCH, got: $tag" >&2
    exit 2
    ;;
esac

repo_root="$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
safe_tag="$(printf '%s' "$tag" | tr -c 'A-Za-z0-9._-' '-')"
tmp_dir="${TESTLOOP_RELEASE_CANDIDATE_TMP_DIR:-/tmp}"
dist_dir="${TESTLOOP_RELEASE_CANDIDATE_DIST_DIR:-/tmp/testloop-${safe_tag}-candidate-dist}"
asset="${TESTLOOP_RELEASE_CANDIDATE_ASSET:-darwin_arm64}"
goos="${TESTLOOP_RELEASE_CANDIDATE_GOOS:-darwin}"
goarch="${TESTLOOP_RELEASE_CANDIDATE_GOARCH:-arm64}"
mcp_binary="${tmp_dir}/testloop-mcp-${safe_tag}-candidate"
testgen_binary="${tmp_dir}/testloop-testgen-${safe_tag}-candidate"
agent_decision_fixture_dir="${tmp_dir}/testloop-agent-decision-fixtures-${safe_tag}"
agent_decision_fixture_json="${tmp_dir}/testloop-agent-decision-fixtures-${safe_tag}.json"
agent_decision_release_response_client_dir="${tmp_dir}/testloop-release-response-client-${safe_tag}"

step() {
  printf '==> %s\n' "$*"
}

require_command() {
  local command="$1"
  if ! command -v "$command" >/dev/null 2>&1; then
    echo "error: missing required command: $command" >&2
    exit 1
  fi
}

checksum_check() {
  local checksum_file="$1"
  local checksum_dir
  local checksum_name
  checksum_dir="$(dirname -- "$checksum_file")"
  checksum_name="$(basename -- "$checksum_file")"

  if command -v sha256sum >/dev/null 2>&1; then
    (cd "$checksum_dir" && sha256sum -c "$checksum_name")
  elif command -v shasum >/dev/null 2>&1; then
    (cd "$checksum_dir" && shasum -a 256 -c "$checksum_name")
  else
    echo "error: missing sha256sum or shasum" >&2
    exit 1
  fi
}

verify_archive_contents() {
  local archive="$1"
  local listing
  case "$archive" in
    *.zip)
      require_command unzip
      listing="$(unzip -Z1 "$archive" | sort)"
      ;;
    *.tar.gz)
      listing="$(tar -tzf "$archive" | sort)"
      ;;
    *)
      echo "error: unsupported archive type: $archive" >&2
      exit 1
      ;;
  esac

  grep -Fx './LICENSE' <<<"$listing" >/dev/null || grep -Fx 'LICENSE' <<<"$listing" >/dev/null
  grep -Fx './README.md' <<<"$listing" >/dev/null || grep -Fx 'README.md' <<<"$listing" >/dev/null
  grep -Fx './testloop-mcp' <<<"$listing" >/dev/null || grep -Fx 'testloop-mcp' <<<"$listing" >/dev/null || grep -Fx 'testloop-mcp.exe' <<<"$listing" >/dev/null
  grep -Fx './testloop-testgen' <<<"$listing" >/dev/null || grep -Fx 'testloop-testgen' <<<"$listing" >/dev/null || grep -Fx 'testloop-testgen.exe' <<<"$listing" >/dev/null
}

cd "$repo_root"

require_command go
require_command node
require_command npm

step "check shell syntax"
find scripts test -name '*.sh' -print0 | xargs -0 -n1 bash -n

step "run go tests"
go test ./...

step "run shell contract tests"
while IFS= read -r test_script; do
  sh "$test_script"
done < <(find test -maxdepth 1 -name '*_test.sh' -print | sort)

step "verify agent decision fixture export package"
rm -rf "$agent_decision_fixture_dir" "$agent_decision_fixture_json"
node scripts/export-agent-decision-fixtures.mjs "$agent_decision_fixture_dir"
(cd "$agent_decision_fixture_dir" && npm test --silent > "$agent_decision_fixture_json")

step "verify agent decision release response client export package"
rm -rf "$agent_decision_release_response_client_dir"
node scripts/export-agent-decision-release-response-client.mjs "$agent_decision_release_response_client_dir"
(cd "$agent_decision_release_response_client_dir" && npm test --silent)

step "build candidate binaries"
go build -o "$mcp_binary" .
go build -o "$testgen_binary" ./cmd/testgen

step "check version and help output"
"$mcp_binary" --version
"$mcp_binary" --help >/tmp/testloop-mcp-${safe_tag}-help.out 2>&1
"$testgen_binary" --help >/tmp/testloop-testgen-${safe_tag}-help.out 2>&1
grep -F "Usage of testloop-mcp" /tmp/testloop-mcp-${safe_tag}-help.out >/dev/null
grep -F "Usage: testgen" /tmp/testloop-testgen-${safe_tag}-help.out >/dev/null

step "package ${asset} release asset"
rm -rf "$dist_dir"
TESTLOOP_MCP_DIST_DIR="$dist_dir" scripts/package-release-asset.sh "$tag" "$asset" "$goos" "$goarch"

archive_ext="tar.gz"
if [[ "$goos" == "windows" ]]; then
  archive_ext="zip"
fi
archive="${dist_dir}/testloop-mcp_${tag}_${asset}.${archive_ext}"
checksum_file="${archive}.sha256"

step "verify checksum and archive contents"
checksum_check "$checksum_file"
verify_archive_contents "$archive"

step "check git diff whitespace"
git diff --check

printf 'release_candidate_status=passed\n'
printf 'tag=%s\n' "$tag"
printf 'asset=%s\n' "$asset"
printf 'dist_dir=%s\n' "$dist_dir"
