#!/usr/bin/env sh
set -eu

tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/testloop-preflight-report-test.XXXXXX")"
trap 'rm -rf "$tmp_dir"' EXIT

passed_json="$tmp_dir/passed.json"
cat > "$passed_json" <<'JSON'
{"ok":true,"missing_count":0,"missing":[],"checks":[]}
JSON

passed_report="$(scripts/render-regression-preflight-report.py "$passed_json")"
printf '%s\n' "$passed_report" | grep -F "## Regression Smoke 前置检查" >/dev/null
printf '%s\n' "$passed_report" | grep -F -- "- 状态：通过" >/dev/null
printf '%s\n' "$passed_report" | grep -F "scripts/validate-regression-smoke.sh" >/dev/null

missing_json="$tmp_dir/missing.json"
cat > "$missing_json" <<'JSON'
{
  "ok": false,
  "missing_count": 3,
  "missing": [
    {"status":"missing","kind":"command","label":"mvn","value":"mvn"},
    {"status":"missing","kind":"dir","label":"Java Commons Lang","value":"/tmp/missing-lang"},
    {"status":"missing","kind":"file","label":"JS mcp-hub repair tasks","value":"/repo/testdata/js-mcp-hub/repair-tasks.jsonl"}
  ],
  "checks": []
}
JSON

missing_report="$(scripts/render-regression-preflight-report.py "$missing_json")"
printf '%s\n' "$missing_report" | grep -F -- "- 状态：未通过" >/dev/null
printf '%s\n' "$missing_report" | grep -F "### 缺失命令" >/dev/null
printf '%s\n' "$missing_report" | grep -F '`mvn`: `mvn`' >/dev/null
printf '%s\n' "$missing_report" | grep -F "### 缺失目录" >/dev/null
printf '%s\n' "$missing_report" | grep -F '`Java Commons Lang`: `/tmp/missing-lang`' >/dev/null
printf '%s\n' "$missing_report" | grep -F "### 缺失 JSONL fixture" >/dev/null
printf '%s\n' "$missing_report" | grep -F '`JS mcp-hub repair tasks`: `/repo/testdata/js-mcp-hub/repair-tasks.jsonl`' >/dev/null

stdin_report="$(scripts/render-regression-preflight-report.py - < "$missing_json")"
printf '%s\n' "$stdin_report" | grep -F "TESTLOOP_REGRESSION_PREFLIGHT_FORMAT=json" >/dev/null

echo "regression preflight report test passed"
