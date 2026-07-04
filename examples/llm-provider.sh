#!/bin/sh
set -eu

tmp_file="$(mktemp)"
trap 'rm -f "$tmp_file"' EXIT
cat > "$tmp_file"

python3 - "$tmp_file" <<'PY'
import json
import sys

with open(sys.argv[1], "r", encoding="utf-8") as f:
    payload = json.load(f)

code = payload.get("static_code", "")
if not code.strip():
    source_file = payload.get("source_file", "source")
    code = f"// No static test code was generated for {source_file}\n"

sys.stdout.write(json.dumps({"code": code}))
PY
