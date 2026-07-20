#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: scripts/showcase-laoxia-scaffold-report.sh [testloop-mcp-binary]

Run a dual-stack laoxia scaffold report for the Go server and Vue web project.
The script writes:
  - server/verification-report.md
  - server/verification-summary.json
  - web/verification-report.md
  - web/verification-summary.json
  - laoxia-summary.json

Environment:
  TESTLOOP_LAOXIA_OUTPUT_DIR      Output dir. Default: /tmp/testloop-laoxia-scaffold
  TESTLOOP_LAOXIA_SUMMARY_JSON    Optional combined summary JSON path.
  TESTLOOP_LAOXIA_SERVER_DIR      Go server project dir.
  TESTLOOP_LAOXIA_WEB_DIR         Vue web project dir.
  TESTLOOP_LAOXIA_SERVER_COMMAND  Server smoke command. Default: go test ./...
  TESTLOOP_LAOXIA_WEB_COMMAND     Web smoke command. Default: pnpm install --frozen-lockfile && pnpm build:prod

All TESTLOOP_LAOXIA_* variables are forwarded to the shared dual-project helper.
USAGE
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

if [[ "$#" -gt 1 ]]; then
  usage >&2
  exit 2
fi

repo_root="$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"

exec env \
  TESTLOOP_PAIR_PREFIX=laoxia \
  TESTLOOP_PAIR_OUTPUT_DIR="${TESTLOOP_LAOXIA_OUTPUT_DIR:-/tmp/testloop-laoxia-scaffold}" \
  TESTLOOP_PAIR_SUMMARY_JSON="${TESTLOOP_LAOXIA_SUMMARY_JSON:-${TESTLOOP_LAOXIA_OUTPUT_DIR:-/tmp/testloop-laoxia-scaffold}/laoxia-summary.json}" \
  TESTLOOP_PAIR_FIRST_NAME=server \
  TESTLOOP_PAIR_FIRST_DIR="${TESTLOOP_LAOXIA_SERVER_DIR:-/Users/binlee/code/free-works/laoxia-scaffold-v1.0.0/car-admin-server}" \
  TESTLOOP_PAIR_FIRST_COMMAND="${TESTLOOP_LAOXIA_SERVER_COMMAND:-go test ./...}" \
  TESTLOOP_PAIR_FIRST_TITLE="${TESTLOOP_LAOXIA_SERVER_TITLE:-laoxia car-admin-server 接入验收报告}" \
  TESTLOOP_PAIR_SECOND_NAME=web \
  TESTLOOP_PAIR_SECOND_DIR="${TESTLOOP_LAOXIA_WEB_DIR:-/Users/binlee/code/free-works/laoxia-scaffold-v1.0.0/car-admin-web}" \
  TESTLOOP_PAIR_SECOND_COMMAND="${TESTLOOP_LAOXIA_WEB_COMMAND:-pnpm install --frozen-lockfile && pnpm build:prod}" \
  TESTLOOP_PAIR_SECOND_TITLE="${TESTLOOP_LAOXIA_WEB_TITLE:-laoxia car-admin-web 接入验收报告}" \
  "$repo_root/scripts/showcase-dual-project-report.sh" "$@"
