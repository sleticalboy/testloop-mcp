#!/usr/bin/env sh
set -eu

python3 - <<'PY'
import json
from pathlib import Path


def load_jsonl(path):
    return [
        json.loads(line)
        for line in Path(path).read_text(encoding="utf-8").splitlines()
        if line.strip()
    ]


rows = load_jsonl("testdata/py-click/ready-hit-tasks.jsonl")
expected = {
    "pytest-19": ("get_binary_stream", "331-334"),
    "pytest-20": ("get_text_stream", "352-355"),
    "pytest-21": ("get_app_dir", "480-489"),
    "pytest-22": ("make_str", "51-56"),
    "pytest-23": ("PacifyFlushWrapper.flush", "516-517"),
    "pytest-32": ("safecall", "39-44"),
    "pytest-33": ("_expand_args", "619-620"),
}

ids = [row.get("id") for row in rows]
expected_ids = list(expected)
if ids != expected_ids:
    raise SystemExit(f"testdata/py-click/ready-hit-tasks.jsonl: ids={ids}, want={expected_ids}")

for row in rows:
    task_id = row["id"]
    target, line_range = expected[task_id]
    if row.get("framework") != "pytest":
        raise SystemExit(f"{task_id}: framework={row.get('framework')}, want pytest")
    if row.get("target") != target:
        raise SystemExit(f"{task_id}: target={row.get('target')}, want {target}")
    if row.get("line_range") != line_range:
        raise SystemExit(f"{task_id}: line_range={row.get('line_range')}, want {line_range}")
    if "src/click/utils.py" not in row.get("file", ""):
        raise SystemExit(f"{task_id}: file does not point to src/click/utils.py")
    if "tests/click/test_utils.py" not in row.get("test_file", ""):
        raise SystemExit(f"{task_id}: test_file does not point to tests/click/test_utils.py")

print("py regression fixture test passed")
PY
