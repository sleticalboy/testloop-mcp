#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"

cd "$repo_root"

tracked_ignored="$(git ls-files -ci --exclude-standard)"
if [ -n "$tracked_ignored" ]; then
  echo "tracked files should not be ignored by .gitignore:" >&2
  printf '%s\n' "$tracked_ignored" >&2
  exit 1
fi

tracked_cache="$(git ls-files | rg '(^|/)__pycache__/|\\.pyc$' || true)"
if [ -n "$tracked_cache" ]; then
  echo "Python bytecode/cache files should not be tracked:" >&2
  printf '%s\n' "$tracked_cache" >&2
  exit 1
fi

echo "repository hygiene test passed"
