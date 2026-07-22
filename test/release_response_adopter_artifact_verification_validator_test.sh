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

validator="scripts/validate-release-response-adopter-artifact-verification.mjs"

out="${tmp_dir}/validator.out"
node "$validator" > "$out"
assert_contains "$out" "release_response_adopter_artifact_verification_status=passed release_ref=v0.5.20"
assert_contains "$out" "release_response_adopter_artifact_status=passed"
assert_contains "$out" "release_response_adopter_artifact_verification_fixture_count=8"
assert_contains "$out" "release_response_adopter_artifact_verification_agent_next_step=ready"
assert_contains "$out" "release_response_adopter_artifact_verification_should_accept=true"
assert_contains "$out" "release_response_adopter_artifact_verification_required_files=6"

json_out="${tmp_dir}/validator.json"
node "$validator" --json > "$json_out"
python3 - "$json_out" <<'PY'
from pathlib import Path
import json
import sys

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
assert payload["status"] == "passed"
assert payload["release_ref"] == "v0.5.20"
assert payload["fixture_count"] == 8
assert payload["agent_next_step"] == "ready"
assert payload["should_accept"] is True
assert payload["required_files"] == 6
assert payload["failures"] == []
PY

runtime_artifact="${tmp_dir}/adopter-artifacts"
runtime_json="${tmp_dir}/runtime-verification.json"
TESTLOOP_RELEASE_RESPONSE_ADOPTER_REPO_DIR="${tmp_dir}/adopter-repo" \
TESTLOOP_RELEASE_RESPONSE_ADOPTER_ARTIFACT_DIR="$runtime_artifact" \
  scripts/showcase-release-response-adopter.sh --json > "${tmp_dir}/showcase.json"
node scripts/verify-release-response-adopter-artifact.mjs --json "$runtime_artifact" > "$runtime_json"
node "$validator" "$runtime_json" > "${tmp_dir}/runtime-validator.out"
assert_contains "${tmp_dir}/runtime-validator.out" "release_response_adopter_artifact_verification_status=passed release_ref=v0.5.20"

failed_fixture="docs/fixtures/release-response-adopter-artifact-verification/missing-summary-consumer.json"
if node "$validator" "$failed_fixture" > "${tmp_dir}/failed.out" 2>&1; then
  echo "expected validator to fail for failed release response adopter artifact verification fixture" >&2
  exit 1
fi
assert_contains "${tmp_dir}/failed.out" "status must be passed"
assert_contains "${tmp_dir}/failed.out" "agent_next_step must be ready"
assert_contains "${tmp_dir}/failed.out" "should_accept must be true"
assert_contains "${tmp_dir}/failed.out" "failures must be an empty array"
assert_contains "${tmp_dir}/failed.out" "files entry testloop-release-response-summary-consumer.json exists must be true"

if node "$validator" --json "$failed_fixture" > "${tmp_dir}/failed.json"; then
  echo "expected JSON validator to fail for failed release response adopter artifact verification fixture" >&2
  exit 1
fi
python3 - "${tmp_dir}/failed.json" <<'PY'
from pathlib import Path
import json
import sys

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
assert payload["status"] == "failed"
assert payload["release_ref"] == "v0.5.20"
assert payload["fixture_count"] == 8
assert payload["agent_next_step"] == "inspect-release-response-adopter-artifact"
assert payload["should_accept"] is False
assert payload["required_files"] == 6
assert any("missing required file testloop-release-response-summary-consumer.json" in item for item in payload["failures"])
assert any("agent_next_step must be ready" in item for item in payload["failures"])
PY

bad_result="${tmp_dir}/bad-result.json"
python3 - "$bad_result" <<'PY'
from pathlib import Path
import json
import sys

payload = json.loads(Path("docs/fixtures/release-response-adopter-artifact-verification/passed.json").read_text(encoding="utf-8"))
payload["release_ref"] = "v0.0.0"
payload["fixture_count"] = 7
payload["required_files"] = 5
payload["files"].pop()
payload["failures"] = ["boom"]
Path(sys.argv[1]).write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")
PY

if node "$validator" --json "$bad_result" > "${tmp_dir}/bad.json"; then
  echo "expected JSON validator to fail for invalid release response adopter artifact verification" >&2
  exit 1
fi
python3 - "${tmp_dir}/bad.json" <<'PY'
from pathlib import Path
import json
import sys

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
assert payload["status"] == "failed"
assert payload["release_ref"] == "v0.0.0"
assert payload["fixture_count"] == 7
assert payload["required_files"] == 5
assert "boom" in payload["failures"]
assert any("release_ref must be v0.5.20" in item for item in payload["failures"])
assert any("files length must be 6" in item for item in payload["failures"])
PY

echo "release response adopter artifact verification validator test passed"
