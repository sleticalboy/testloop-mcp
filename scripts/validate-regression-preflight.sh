#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat >&2 <<'USAGE'
用法：scripts/validate-regression-preflight.sh

在运行固定 regression smoke 前做快速前置检查。该脚本只检查目录、静态
JSONL fixture 和常用命令是否存在，不执行覆盖率、测试生成或真实项目测试。

环境变量：
  TESTLOOP_REGRESSION_SKIP_JAVA
                                    true 时跳过 Java 检查。
  TESTLOOP_REGRESSION_SKIP_JS
                                    true 时跳过 JS 检查。
  TESTLOOP_REGRESSION_SKIP_PY
                                    true 时跳过 Python 检查。
  TESTLOOP_REGRESSION_PREFLIGHT_FORMAT
                                    输出格式：text 或 json。
                                    默认：text
  TESTLOOP_JAVA_REGRESSION_* / TESTLOOP_JS_REGRESSION_* / TESTLOOP_PY_REGRESSION_*
                                    与各语言 regression 样本脚本保持一致。
USAGE
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

if [[ $# -ne 0 ]]; then
  usage
  exit 2
fi

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/.." && pwd)"
failures=0
output_format="${TESTLOOP_REGRESSION_PREFLIGHT_FORMAT:-text}"
checks_file="$(mktemp "${TMPDIR:-/tmp}/testloop-preflight-checks.XXXXXX")"
trap 'rm -f "$checks_file"' EXIT

case "$output_format" in
  text|json)
    ;;
  *)
    echo "unsupported TESTLOOP_REGRESSION_PREFLIGHT_FORMAT: $output_format (expected text or json)" >&2
    exit 2
    ;;
esac

env_bool() {
  case "$(printf '%s' "${1:-}" | tr '[:upper:]' '[:lower:]')" in
    1|true|yes|y|on) return 0 ;;
    *) return 1 ;;
  esac
}

report_missing() {
  local message="$1"
  if [[ "$output_format" == "text" ]]; then
    echo "missing: $message" >&2
  fi
  failures=$((failures + 1))
}

record_check() {
  local status="$1"
  local kind="$2"
  local label="$3"
  local value="$4"
  printf '%s\t%s\t%s\t%s\n' "$status" "$kind" "$label" "$value" >> "$checks_file"
}

log_section() {
  local label="$1"
  if [[ "$output_format" == "text" ]]; then
    echo "preflight: $label"
  fi
}

check_dir() {
  local label="$1"
  local path="$2"
  if [[ ! -d "$path" ]]; then
    record_check "missing" "dir" "$label" "$path"
    report_missing "$label directory: $path"
  else
    record_check "ok" "dir" "$label" "$path"
  fi
}

check_file() {
  local label="$1"
  local path="$2"
  if [[ ! -f "$path" ]]; then
    record_check "missing" "file" "$label" "$path"
    report_missing "$label file: $path"
  else
    record_check "ok" "file" "$label" "$path"
  fi
}

check_command() {
  local command_name="$1"
  if ! command -v "$command_name" >/dev/null 2>&1; then
    record_check "missing" "command" "$command_name" "$command_name"
    report_missing "command: $command_name"
  else
    record_check "ok" "command" "$command_name" "$(command -v "$command_name")"
  fi
}

check_java() {
  local lang_dir="${TESTLOOP_JAVA_REGRESSION_LANG_DIR:-/tmp/testloop-commons-lang}"
  local codec_dir="${TESTLOOP_JAVA_REGRESSION_CODEC_DIR:-/tmp/testloop-commons-codec}"
  local rocketmq_dir="${TESTLOOP_JAVA_REGRESSION_ROCKETMQ_DIR:-/Users/binlee/code/free-works/haoying/rocketmq-clients/java}"
  local lang_ready_tasks="${TESTLOOP_JAVA_REGRESSION_LANG_READY_TASKS_FILE:-$repo_root/testdata/java-commons-lang/ready-hit-tasks.jsonl}"
  local lang_manual_tasks="${TESTLOOP_JAVA_REGRESSION_LANG_MANUAL_TASKS_FILE:-$repo_root/testdata/java-commons-lang/manual-internal-tasks.jsonl}"
  local codec_unreachable_tasks="${TESTLOOP_JAVA_REGRESSION_CODEC_UNREACHABLE_TASKS_FILE:-$repo_root/testdata/java-commons-codec/unreachable-tasks.jsonl}"
  local rocketmq_statuschecker_tasks="${TESTLOOP_JAVA_REGRESSION_ROCKETMQ_STATUSCHECKER_TASKS_FILE:-$repo_root/testdata/java-rocketmq-statuschecker/statuschecker-tasks.jsonl}"

  log_section java
  check_command go
  check_command python3
  check_command mvn
  check_dir "Java Commons Lang" "$lang_dir"
  check_dir "Java Commons Codec" "$codec_dir"
  check_dir "Java RocketMQ" "$rocketmq_dir"
  check_file "Java Commons Lang ready tasks" "$lang_ready_tasks"
  check_file "Java Commons Lang manual tasks" "$lang_manual_tasks"
  check_file "Java Commons Codec unreachable tasks" "$codec_unreachable_tasks"
  check_file "Java RocketMQ StatusChecker tasks" "$rocketmq_statuschecker_tasks"
}

check_js() {
  local ip2region_dir="${TESTLOOP_JS_REGRESSION_IP2REGION_DIR:-/Users/binlee/code/open-source/ip2region/binding/javascript}"
  local ip2region_tasks="${TESTLOOP_JS_REGRESSION_IP2REGION_TASKS_FILE:-$repo_root/testdata/js-ip2region/ready-hit-tasks.jsonl}"
  local no_runtime_dir="${TESTLOOP_JS_REGRESSION_NO_RUNTIME_DIR:-$repo_root/testdata/js-no-runtime}"
  local no_runtime_tasks="${TESTLOOP_JS_REGRESSION_NO_RUNTIME_TASKS_FILE:-$repo_root/testdata/js-no-runtime/no-runtime-tasks.jsonl}"
  local internal_dir="${TESTLOOP_JS_REGRESSION_INTERNAL_DIR:-$repo_root/testdata/js-internal}"
  local internal_tasks="${TESTLOOP_JS_REGRESSION_INTERNAL_TASKS_FILE:-$repo_root/testdata/js-internal/internal-tasks.jsonl}"
  local mcp_hub_dir="${TESTLOOP_JS_REGRESSION_MCP_HUB_DIR:-/Users/binlee/code/open-source/mcp-hub}"
  local mcp_hub_repair_tasks="${TESTLOOP_JS_REGRESSION_MCP_HUB_REPAIR_TASKS_FILE:-$repo_root/testdata/js-mcp-hub/repair-tasks.jsonl}"
  local mcp_hub_env_tasks="${TESTLOOP_JS_REGRESSION_MCP_HUB_ENV_TASKS_FILE:-$repo_root/testdata/js-mcp-hub/env-tasks.jsonl}"
  local mcp_hub_devwatcher_tasks="${TESTLOOP_JS_REGRESSION_MCP_HUB_DEVWATCHER_TASKS_FILE:-$repo_root/testdata/js-mcp-hub/devwatcher-tasks.jsonl}"
  local mcp_hub_sse_tasks="${TESTLOOP_JS_REGRESSION_MCP_HUB_SSE_TASKS_FILE:-$repo_root/testdata/js-mcp-hub/sse-tasks.jsonl}"
  local mcp_hub_workspace_tasks="${TESTLOOP_JS_REGRESSION_MCP_HUB_WORKSPACE_TASKS_FILE:-$repo_root/testdata/js-mcp-hub/workspace-tasks.jsonl}"

  log_section js
  check_command go
  check_command node
  check_command npx
  check_dir "JS ip2region" "$ip2region_dir"
  check_dir "JS no-runtime fixture" "$no_runtime_dir"
  check_dir "JS internal fixture" "$internal_dir"
  check_dir "JS mcp-hub" "$mcp_hub_dir"
  check_file "JS ip2region tasks" "$ip2region_tasks"
  check_file "JS no-runtime tasks" "$no_runtime_tasks"
  check_file "JS internal tasks" "$internal_tasks"
  check_file "JS mcp-hub repair tasks" "$mcp_hub_repair_tasks"
  check_file "JS mcp-hub env tasks" "$mcp_hub_env_tasks"
  check_file "JS mcp-hub DevWatcher tasks" "$mcp_hub_devwatcher_tasks"
  check_file "JS mcp-hub SSE tasks" "$mcp_hub_sse_tasks"
  check_file "JS mcp-hub workspace tasks" "$mcp_hub_workspace_tasks"
}

check_py() {
  local click_dir="${TESTLOOP_PY_REGRESSION_CLICK_DIR:-/tmp/testloop-click-sample}"
  local click_tasks="${TESTLOOP_PY_REGRESSION_CLICK_TASKS_FILE:-$repo_root/testdata/py-click/ready-hit-tasks.jsonl}"
  local internal_dir="${TESTLOOP_PY_REGRESSION_INTERNAL_DIR:-$repo_root/testdata/py-internal}"
  local internal_tasks="${TESTLOOP_PY_REGRESSION_INTERNAL_TASKS_FILE:-$repo_root/testdata/py-internal/internal-tasks.jsonl}"
  local apk_station_dir="${TESTLOOP_PY_REGRESSION_APK_STATION_DIR:-/Users/binlee/code/free-works/haoy-apk-station/backend}"
  local apk_station_tasks="${TESTLOOP_PY_REGRESSION_APK_STATION_TASKS_FILE:-$repo_root/testdata/py-haoy-apk-station/environment-tasks.jsonl}"
  local apk_station_external_tasks="${TESTLOOP_PY_REGRESSION_APK_STATION_EXTERNAL_TASKS_FILE:-$repo_root/testdata/py-haoy-apk-station/external-service-tasks.jsonl}"
  local apk_station_database_tasks="${TESTLOOP_PY_REGRESSION_APK_STATION_DATABASE_TASKS_FILE:-$repo_root/testdata/py-haoy-apk-station/database-tasks.jsonl}"

  log_section python
  check_command go
  check_command python3
  check_command uv
  check_dir "Python Click" "$click_dir"
  check_dir "Python internal fixture" "$internal_dir"
  check_dir "Python haoy-apk-station" "$apk_station_dir"
  check_file "Python Click tasks" "$click_tasks"
  check_file "Python internal tasks" "$internal_tasks"
  check_file "Python haoy-apk-station environment tasks" "$apk_station_tasks"
  check_file "Python haoy-apk-station external-service tasks" "$apk_station_external_tasks"
  check_file "Python haoy-apk-station database tasks" "$apk_station_database_tasks"
}

if ! env_bool "${TESTLOOP_REGRESSION_SKIP_JAVA:-}"; then
  check_java
fi

if ! env_bool "${TESTLOOP_REGRESSION_SKIP_JS:-}"; then
  check_js
fi

if ! env_bool "${TESTLOOP_REGRESSION_SKIP_PY:-}"; then
  check_py
fi

emit_json_summary() {
  python3 - "$checks_file" "$failures" <<'PY'
import json
import sys

checks_path, failures = sys.argv[1], int(sys.argv[2])
checks = []
with open(checks_path, "r", encoding="utf-8") as handle:
    for line in handle:
        line = line.rstrip("\n")
        if not line:
            continue
        status, kind, label, value = line.split("\t", 3)
        checks.append({
            "status": status,
            "kind": kind,
            "label": label,
            "value": value,
        })

payload = {
    "ok": failures == 0,
    "missing_count": failures,
    "missing": [item for item in checks if item["status"] == "missing"],
    "checks": checks,
}
print(json.dumps(payload, ensure_ascii=False, sort_keys=True))
PY
}

if [[ "$output_format" == "json" ]]; then
  emit_json_summary
fi

if [[ "$failures" -gt 0 ]]; then
  if [[ "$output_format" == "text" ]]; then
    echo "regression preflight failed: $failures missing requirement(s)" >&2
  fi
  exit 1
fi

if [[ "$output_format" == "text" ]]; then
  echo "regression preflight passed"
fi
