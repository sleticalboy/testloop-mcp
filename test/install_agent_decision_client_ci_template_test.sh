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

assert_not_exists() {
  path="$1"
  if [ -e "$path" ]; then
    echo "expected path not to exist: $path" >&2
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

script="${repo_root}/scripts/install-agent-decision-client-ci-template.sh"
client_dir="${tmp_dir}/client"
mkdir -p "$client_dir"

python3 - "$repo_root" "$script" <<'PY'
from pathlib import Path
import re
import sys

repo_root = Path(sys.argv[1])
script = Path(sys.argv[2])
main_go = (repo_root / "main.go").read_text(encoding="utf-8")
script_text = script.read_text(encoding="utf-8")
version = re.search(r'^const appVersion = "([^"]+)"', main_go, flags=re.M)
default_ref = re.search(r'^default_helper_ref="([^"]+)"', script_text, flags=re.M)
if not version or not default_ref:
    print("failed to read appVersion or default_helper_ref", file=sys.stderr)
    sys.exit(1)
want = f"v{version.group(1)}"
if default_ref.group(1) != want:
    print(f"default_helper_ref = {default_ref.group(1)!r}, want {want!r}", file=sys.stderr)
    sys.exit(1)
PY

out="${tmp_dir}/help.out"
run_expect_code 0 "$out" bash "$script" --help
assert_contains "$out" "Usage: scripts/install-agent-decision-client-ci-template.sh"
assert_contains "$out" "--version REF"
assert_contains "$out" "--workflow-path PATH"
assert_contains "$out" "--dry-run"

out="${tmp_dir}/dry-run.out"
run_expect_code 0 "$out" bash "$script" --dry-run "$client_dir"
assert_contains "$out" "agent_decision_client_ci_template_ref=v0.5.19"
assert_contains "$out" "agent_decision_client_ci_template_status=dry-run"
assert_not_exists "${client_dir}/.github/workflows/testloop-agent-decision-contract.yml"

out="${tmp_dir}/write.out"
run_expect_code 0 "$out" bash "$script" "$client_dir"
workflow="${client_dir}/.github/workflows/testloop-agent-decision-contract.yml"
assert_contains "$out" "agent_decision_client_ci_template_status=written"
assert_contains "$workflow" "name: testloop agent decision contract"
assert_contains "$workflow" "repository: sleticalboy/testloop-mcp"
assert_contains "$workflow" "ref: v0.5.19"
assert_contains "$workflow" ".testloop-mcp/scripts/showcase-agent-decision-client-ci.sh --json"
assert_contains "$workflow" "tee /tmp/testloop-agent-decision-client-summary.json"
assert_contains "$workflow" "Render Agent decision response"
assert_contains "$workflow" ".testloop-mcp/scripts/render-agent-decision-client-ci-response.mjs"
assert_contains "$workflow" "tee /tmp/testloop-agent-decision-client-response.json"
assert_contains "$workflow" "/tmp/testloop-agent-decision-client-response.json"

out="${tmp_dir}/exists.out"
run_expect_code 1 "$out" bash "$script" "$client_dir"
assert_contains "$out" "workflow already exists"

out="${tmp_dir}/force.out"
run_expect_code 0 "$out" bash "$script" --force --version v9.9.9 "$client_dir"
assert_contains "$workflow" "ref: v9.9.9"

custom_client_dir="${tmp_dir}/custom-client"
mkdir -p "$custom_client_dir"
out="${tmp_dir}/custom.out"
run_expect_code 0 "$out" bash "$script" --workflow-path .github/workflows/custom.yml "$custom_client_dir"
custom_workflow="${custom_client_dir}/.github/workflows/custom.yml"
assert_contains "$custom_workflow" "ref: v0.5.19"

out="${tmp_dir}/bad-path.out"
run_expect_code 1 "$out" bash "$script" --workflow-path ../bad.yml "$custom_client_dir"
assert_contains "$out" "--workflow-path must not contain .."

standalone_dir="${tmp_dir}/standalone"
standalone_client_dir="${tmp_dir}/standalone-client"
mkdir -p "$standalone_dir" "$standalone_client_dir"
cp "$script" "${standalone_dir}/install-agent-decision-client-ci-template.sh"
out="${tmp_dir}/standalone.out"
run_expect_code 0 "$out" bash "${standalone_dir}/install-agent-decision-client-ci-template.sh" "$standalone_client_dir"
assert_contains "$out" "agent_decision_client_ci_template_ref=v0.5.19"
assert_contains "${standalone_client_dir}/.github/workflows/testloop-agent-decision-contract.yml" "ref: v0.5.19"

ruby -e 'require "yaml"; data = YAML.load_file(ARGV.fetch(0)); raise "missing jobs" unless data["jobs"] || data[true]' "$custom_workflow"

python3 - "${repo_root}/docs/agent-decision-client-ci-template.md" "$custom_workflow" <<'PY'
from pathlib import Path
import re
import sys

doc = Path(sys.argv[1])
workflow = Path(sys.argv[2])
blocks = re.findall(r"```yaml\n(.*?)\n```", doc.read_text(encoding="utf-8"), flags=re.S)
if len(blocks) != 1:
    print(f"{doc}: expected exactly 1 yaml block, found {len(blocks)}", file=sys.stderr)
    sys.exit(1)
if blocks[0].strip() != workflow.read_text(encoding="utf-8").strip():
    print("generated workflow does not match documented YAML template", file=sys.stderr)
    sys.exit(1)
PY

echo "install Agent decision client CI template test passed"
