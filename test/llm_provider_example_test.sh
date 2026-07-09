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

src_dir="${tmp_dir}/src"
mkdir -p "$src_dir"

cat > "${src_dir}/api.ts" <<'SRC'
import type { ExternalUser } from './types';

export async function loadUser(response: Response): Promise<ExternalUser> {
  return await response.json();
}
SRC

cat > "${src_dir}/types.ts" <<'SRC'
export interface ExternalUser {
  userId: number;
  email: string;
}
SRC

request_json="${tmp_dir}/request.json"
prompt_file="${tmp_dir}/prompt.md"
stdout_json="${tmp_dir}/stdout.json"

cat > "$request_json" <<JSON
{
  "source_file": "${src_dir}/api.ts",
  "context": {
    "language": "typescript",
    "framework": "vitest",
    "source_file": "${src_dir}/api.ts",
    "imports": ["import type { ExternalUser } from './types'"],
    "targets": [
      {
        "name": "loadUser",
        "kind": "function",
        "return_type_expr": "Promise<ExternalUser>",
        "payload_notes": [
          "return annotation ExternalUser is not declared in the same source file; static payload falls back to { ok: true }",
          "return annotation references imported type ExternalUser from './types'; read candidate source files: types.ts, types.tsx, types.d.ts, types.js, types.jsx, types.mjs, types.cjs, types/index.ts, types/index.tsx, types/index.d.ts, types/index.js, types/index.jsx, types/index.mjs, types/index.cjs"
        ]
      }
    ]
  },
  "static_code": "it('uses static code', () => { expect(true).toBe(true); });\\n"
}
JSON

TESTLOOP_LLM_PROVIDER_PROMPT_FILE="$prompt_file" \
  sh "${repo_root}/examples/llm-provider.sh" < "$request_json" > "$stdout_json"

python3 - "$stdout_json" <<'PY'
import json
import sys

with open(sys.argv[1], "r", encoding="utf-8") as f:
    payload = json.load(f)

code = payload.get("code", "")
if "uses static code" not in code:
    raise SystemExit(f"provider did not return static code: {payload!r}")
PY

assert_contains "$prompt_file" "## Imported Type Context"
assert_contains "$prompt_file" "### types.ts"
assert_contains "$prompt_file" "export interface ExternalUser"
assert_contains "$prompt_file" "userId: number;"
assert_contains "$prompt_file" "return annotation references imported type ExternalUser"

echo "llm provider example tests passed"
