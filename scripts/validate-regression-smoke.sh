#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat >&2 <<'USAGE'
用法：scripts/validate-regression-smoke.sh

运行当前仓库维护的固定真实项目小回归矩阵。它面向发布前或改动后快速
验证测试反馈闭环，不替代完整 top-N 真实项目验证。

环境变量：
  TESTLOOP_REGRESSION_OUTPUT_DIR
                                    总输出目录。
                                    默认：/tmp/testloop-regression-smoke-<timestamp>
  TESTLOOP_REGRESSION_SKIP_JAVA
                                    true 时跳过 Java 样本。
  TESTLOOP_REGRESSION_SKIP_PY
                                    true 时跳过 Python 样本。
  TESTLOOP_VALIDATE_* / TESTLOOP_JAVA_REGRESSION_* / TESTLOOP_PY_REGRESSION_*
                                    透传给各语言样本脚本。
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
output_dir="${TESTLOOP_REGRESSION_OUTPUT_DIR:-/tmp/testloop-regression-smoke-$(date +%Y%m%d%H%M%S)}"

env_bool() {
  case "$(printf '%s' "${1:-}" | tr '[:upper:]' '[:lower:]')" in
    1|true|yes|y|on) return 0 ;;
    *) return 1 ;;
  esac
}

mkdir -p "$output_dir"

if ! env_bool "${TESTLOOP_REGRESSION_SKIP_JAVA:-}"; then
  echo "==> java regression samples"
  TESTLOOP_JAVA_REGRESSION_OUTPUT_DIR="$output_dir/java" \
    "$script_dir/validate-java-regression-samples.sh"
fi

if ! env_bool "${TESTLOOP_REGRESSION_SKIP_PY:-}"; then
  echo "==> python regression samples"
  TESTLOOP_PY_REGRESSION_OUTPUT_DIR="$output_dir/python" \
    "$script_dir/validate-py-regression-samples.sh"
fi

echo "regression_smoke_output_dir=$output_dir"
