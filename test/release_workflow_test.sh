#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
workflow="${repo_root}/.github/workflows/release.yml"

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

assert_count() {
  file="$1"
  needle="$2"
  expected="$3"
  count="$(grep -F -- "$needle" "$file" | wc -l | tr -d ' ')"
  if [ "$count" != "$expected" ]; then
    echo "expected $file to contain $expected occurrences of: $needle" >&2
    echo "got $count" >&2
    exit 1
  fi
}

assert_contains "$workflow" "concurrency:"
assert_contains "$workflow" "group: release-artifacts-\${{ github.event_name == 'workflow_dispatch' && inputs.tag || github.ref_name }}"
assert_contains "$workflow" "ensure-release:"
assert_contains "$workflow" "name: Ensure GitHub Release"
assert_contains "$workflow" "outputs:"
assert_contains "$workflow" "tag: \${{ steps.resolve.outputs.tag }}"
assert_contains "$workflow" "needs: ensure-release"
assert_contains "$workflow" "TAG_NAME: \${{ needs.ensure-release.outputs.tag }}"
assert_contains "$workflow" "gh release upload \"\$TAG_NAME\" \"\$file\" --clobber"

assert_count "$workflow" "gh release create" 1
assert_count "$workflow" "name: Ensure GitHub Release exists" 1

python3 - "$workflow" <<'PY'
from pathlib import Path
import sys

text = Path(sys.argv[1]).read_text(encoding="utf-8")
failures = []

ensure_index = text.find("  ensure-release:")
build_index = text.find("  build:")
create_index = text.find("gh release create")
upload_index = text.find("gh release upload")
if ensure_index == -1:
    failures.append("missing ensure-release job")
if build_index == -1:
    failures.append("missing build job")
if create_index == -1:
    failures.append("missing gh release create command")
if upload_index == -1:
    failures.append("missing gh release upload command")
if not failures and not (ensure_index < build_index < upload_index):
    failures.append("build/upload job must run after ensure-release job")
if not failures and not (ensure_index < create_index < build_index):
    failures.append("gh release create must stay in ensure-release before matrix build")

build_block = text[build_index:] if build_index != -1 else ""
if "gh release create" in build_block:
    failures.append("matrix build job must not create GitHub Releases")
if "name: Ensure GitHub Release exists" in build_block:
    failures.append("matrix build job must not contain release creation step")

if failures:
    print("release workflow test failed:", file=sys.stderr)
    for failure in failures:
        print(f"- {failure}", file=sys.stderr)
    sys.exit(1)

print("release workflow test passed")
PY
