#!/usr/bin/env sh
set -eu

help_output="$(scripts/validate-regression-preflight.sh --help 2>&1)"
printf '%s\n' "$help_output" | grep -F "scripts/validate-regression-preflight.sh" >/dev/null
printf '%s\n' "$help_output" | grep -F "不执行覆盖率、测试生成或真实项目测试" >/dev/null

TESTLOOP_REGRESSION_SKIP_JAVA=true \
TESTLOOP_REGRESSION_SKIP_JS=true \
TESTLOOP_REGRESSION_SKIP_PY=true \
  scripts/validate-regression-preflight.sh | grep -F "regression preflight passed" >/dev/null

json_ok="$(
  TESTLOOP_REGRESSION_SKIP_JAVA=true \
  TESTLOOP_REGRESSION_SKIP_JS=true \
  TESTLOOP_REGRESSION_SKIP_PY=true \
  TESTLOOP_REGRESSION_PREFLIGHT_FORMAT=json \
    scripts/validate-regression-preflight.sh
)"
python3 - "$json_ok" <<'PY'
import json
import sys

payload = json.loads(sys.argv[1])
if payload["ok"] is not True:
    raise SystemExit(f"ok={payload['ok']}, want true")
if payload["missing_count"] != 0:
    raise SystemExit(f"missing_count={payload['missing_count']}, want 0")
if payload["missing"] != []:
    raise SystemExit(f"missing={payload['missing']}, want []")
PY

missing_output="$(
  TESTLOOP_REGRESSION_SKIP_JS=true \
  TESTLOOP_REGRESSION_SKIP_PY=true \
  TESTLOOP_JAVA_REGRESSION_LANG_DIR=/tmp/testloop-missing-java-lang-fixture \
  scripts/validate-regression-preflight.sh 2>&1 || true
)"
printf '%s\n' "$missing_output" | grep -F "missing: Java Commons Lang directory: /tmp/testloop-missing-java-lang-fixture" >/dev/null
printf '%s\n' "$missing_output" | grep -F "regression preflight failed:" >/dev/null

json_missing="$(
  TESTLOOP_REGRESSION_SKIP_JS=true \
  TESTLOOP_REGRESSION_SKIP_PY=true \
  TESTLOOP_REGRESSION_PREFLIGHT_FORMAT=json \
  TESTLOOP_JAVA_REGRESSION_LANG_DIR=/tmp/testloop-missing-java-lang-fixture \
    scripts/validate-regression-preflight.sh || true
)"
python3 - "$json_missing" <<'PY'
import json
import sys

payload = json.loads(sys.argv[1])
if payload["ok"] is not False:
    raise SystemExit(f"ok={payload['ok']}, want false")
if payload["missing_count"] < 1:
    raise SystemExit(f"missing_count={payload['missing_count']}, want >= 1")
expected = {
    "status": "missing",
    "kind": "dir",
    "label": "Java Commons Lang",
    "value": "/tmp/testloop-missing-java-lang-fixture",
}
if expected not in payload["missing"]:
    raise SystemExit(f"missing does not include {expected}: {payload['missing']}")
PY

echo "regression preflight test passed"
