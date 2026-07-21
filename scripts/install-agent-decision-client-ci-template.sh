#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: scripts/install-agent-decision-client-ci-template.sh [options] [client-dir]

Install the testloop Agent decision contract GitHub Actions workflow into an
external MCP client repository.

Options:
  --version REF          testloop-mcp helper ref. Default: v<main.go appVersion>,
                         or a built-in stable ref when run as a standalone script.
  --workflow-path PATH   Workflow path under client-dir.
                         Default: .github/workflows/testloop-agent-decision-contract.yml
  --force               Overwrite an existing workflow file.
  --dry-run             Print the target path and helper ref without writing.
  -h, --help            Show this help.

Examples:
  scripts/install-agent-decision-client-ci-template.sh /path/to/client
  scripts/install-agent-decision-client-ci-template.sh --version v0.5.17 /path/to/client
USAGE
}

fail() {
  printf 'error: %s\n' "$*" >&2
  exit 1
}

repo_root="$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
default_helper_ref="v0.5.17"
client_dir="."
workflow_path=".github/workflows/testloop-agent-decision-contract.yml"
helper_ref="${TESTLOOP_AGENT_DECISION_CI_VERSION:-}"
force=0
dry_run=0

while [[ "$#" -gt 0 ]]; do
  case "$1" in
    -h|--help)
      usage
      exit 0
      ;;
    --version)
      [[ "$#" -ge 2 ]] || fail "--version requires a value"
      helper_ref="$2"
      shift 2
      ;;
    --workflow-path)
      [[ "$#" -ge 2 ]] || fail "--workflow-path requires a value"
      workflow_path="$2"
      shift 2
      ;;
    --force)
      force=1
      shift
      ;;
    --dry-run)
      dry_run=1
      shift
      ;;
    --*)
      usage >&2
      exit 2
      ;;
    *)
      if [[ "$client_dir" != "." ]]; then
        usage >&2
        exit 2
      fi
      client_dir="$1"
      shift
      ;;
  esac
done

if [[ -z "$helper_ref" ]]; then
  if [[ -f "$repo_root/main.go" ]]; then
    app_version="$(sed -n 's/^const appVersion = "\([^"]*\)"/\1/p' "$repo_root/main.go" | head -n 1)"
  else
    app_version=""
  fi
  if [[ -n "$app_version" ]]; then
    helper_ref="v${app_version}"
  else
    helper_ref="$default_helper_ref"
  fi
fi

[[ "$helper_ref" != *[$'\n\r\t ']* ]] || fail "helper ref must not contain whitespace"
[[ "$workflow_path" != /* ]] || fail "--workflow-path must be relative to client-dir"
[[ "$workflow_path" != *..* ]] || fail "--workflow-path must not contain .."
[[ -d "$client_dir" ]] || fail "client dir must be an existing directory: $client_dir"

target_path="${client_dir%/}/${workflow_path}"
[[ ! -d "$target_path" ]] || fail "workflow path must not be a directory: $target_path"
if [[ -e "$target_path" && "$force" != "1" ]]; then
  fail "workflow already exists: $target_path; pass --force to overwrite"
fi

printf 'agent_decision_client_ci_template_ref=%s\n' "$helper_ref"
printf 'agent_decision_client_ci_template_path=%s\n' "$target_path"
if [[ "$dry_run" = "1" ]]; then
  printf 'agent_decision_client_ci_template_status=dry-run\n'
  exit 0
fi

mkdir -p "$(dirname "$target_path")"
cat > "$target_path" <<YAML
name: testloop agent decision contract

on:
  workflow_dispatch:
  pull_request:

jobs:
  agent-decision-contract:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-node@v4
        with:
          node-version: 22

      - name: Checkout testloop-mcp fixture helpers
        uses: actions/checkout@v4
        with:
          repository: sleticalboy/testloop-mcp
          ref: ${helper_ref}
          path: .testloop-mcp

      - name: Verify Agent decision fixture contract
        run: |
          TESTLOOP_AGENT_DECISION_CLIENT_DIR=/tmp/testloop-agent-decision-client \\
            .testloop-mcp/scripts/showcase-agent-decision-client-ci.sh --json \\
            | tee /tmp/testloop-agent-decision-client-summary.json

      - name: Upload Agent decision result
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: testloop-agent-decision-contract
          path: |
            /tmp/testloop-agent-decision-client-summary.json
            /tmp/testloop-agent-decision-client/agent-decision-fixtures-result.json
            /tmp/testloop-agent-decision-client/testloop-agent-decision-fixtures/package.json
            /tmp/testloop-agent-decision-client/testloop-agent-decision-fixtures/docs/fixtures/agent-decision-fixtures.json
YAML

printf 'agent_decision_client_ci_template_status=written\n'
