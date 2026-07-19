#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

mkdir -p "$tmp_dir/src/models" "$tmp_dir/tests"

cat > "$tmp_dir/package.json" <<'JSON'
{
  "scripts": {
    "test": "vitest run"
  },
  "devDependencies": {
    "vitest": "^3.0.0"
  }
}
JSON

cat > "$tmp_dir/src/api.ts" <<'TS'
import type { ExternalUser } from './models';

export async function loadUser(response: Response): Promise<ExternalUser> {
  return await response.json();
}
TS

cat > "$tmp_dir/src/models/index.ts" <<'TS'
export type * from './user';
TS

cat > "$tmp_dir/src/models/user.ts" <<'TS'
export interface ExternalUser {
  userId: number;
  email: string;
}
TS

output="$tmp_dir/tests/api.test.ts"
(
  cd "$repo_root"
  go run ./cmd/testgen "$tmp_dir/src/api.ts" "$output" >/tmp/testloop-js-ts-barrel-cli-smoke.out
)

grep -F "Generated: $output (provider=static)" /tmp/testloop-js-ts-barrel-cli-smoke.out >/dev/null
grep -F "import { describe, it, expect } from 'vitest';" "$output" >/dev/null
grep -F "import { loadUser } from '../src/api';" "$output" >/dev/null
grep -F "json: async () => ({ userId: 1, email: 'user@example.com' })" "$output" >/dev/null
grep -F "expect(result).toEqual({ userId: 1, email: 'user@example.com' });" "$output" >/dev/null

echo "js/ts barrel cli smoke test passed"
