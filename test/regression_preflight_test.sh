#!/usr/bin/env sh
set -eu

help_output="$(scripts/validate-regression-preflight.sh --help 2>&1)"
printf '%s\n' "$help_output" | grep -F "scripts/validate-regression-preflight.sh" >/dev/null
printf '%s\n' "$help_output" | grep -F "不执行覆盖率、测试生成或真实项目测试" >/dev/null

TESTLOOP_REGRESSION_SKIP_JAVA=true \
TESTLOOP_REGRESSION_SKIP_JS=true \
TESTLOOP_REGRESSION_SKIP_PY=true \
  scripts/validate-regression-preflight.sh | grep -F "regression preflight passed" >/dev/null

missing_output="$(
  TESTLOOP_REGRESSION_SKIP_JS=true \
  TESTLOOP_REGRESSION_SKIP_PY=true \
  TESTLOOP_JAVA_REGRESSION_LANG_DIR=/tmp/testloop-missing-java-lang-fixture \
  scripts/validate-regression-preflight.sh 2>&1 || true
)"
printf '%s\n' "$missing_output" | grep -F "missing: Java Commons Lang directory: /tmp/testloop-missing-java-lang-fixture" >/dev/null
printf '%s\n' "$missing_output" | grep -F "regression preflight failed:" >/dev/null

echo "regression preflight test passed"
