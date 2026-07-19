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


def assert_rows(path, expected):
    rows = load_jsonl(path)
    ids = [row.get("id") for row in rows]
    expected_ids = list(expected)
    if ids != expected_ids:
        raise SystemExit(f"{path}: ids={ids}, want={expected_ids}")
    for row in rows:
        task_id = row["id"]
        want = expected[task_id]
        for key in ("framework", "target", "line_range"):
            if row.get(key) != want[key]:
                raise SystemExit(f"{path}: {task_id} {key}={row.get(key)}, want {want[key]}")
        if want["file"] not in row.get("file", ""):
            raise SystemExit(f"{path}: {task_id} file does not point to {want['file']}")
        if want["test_file"] not in row.get("test_file", ""):
            raise SystemExit(f"{path}: {task_id} test_file does not point to {want['test_file']}")


assert_rows(
    "testdata/js-ip2region/ready-hit-tasks.jsonl",
    {
        "jest-1": {
            "framework": "jest",
            "target": "versionFromHeader",
            "line_range": "317-319",
            "file": "util.js",
            "test_file": "util.test.js",
        },
        "jest-2": {
            "framework": "jest",
            "target": "versionFromHeader",
            "line_range": "322-324",
            "file": "util.js",
            "test_file": "util.test.js",
        },
    },
)

print("js regression fixture test passed")
PY
