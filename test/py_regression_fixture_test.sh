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
        if row.get("framework") != "pytest":
            raise SystemExit(f"{path}: {task_id} framework={row.get('framework')}, want pytest")
        for key in ("target", "line_range"):
            if row.get(key) != want[key]:
                raise SystemExit(f"{path}: {task_id} {key}={row.get(key)}, want {want[key]}")
        if want["file"] not in row.get("file", ""):
            raise SystemExit(f"{path}: {task_id} file does not point to {want['file']}")
        if want["test_file"] not in row.get("test_file", ""):
            raise SystemExit(f"{path}: {task_id} test_file does not point to {want['test_file']}")


assert_rows(
    "testdata/py-click/ready-hit-tasks.jsonl",
    {
        "pytest-19": {
            "target": "get_binary_stream",
            "line_range": "331-334",
            "file": "src/click/utils.py",
            "test_file": "tests/click/test_utils.py",
        },
        "pytest-20": {
            "target": "get_text_stream",
            "line_range": "352-355",
            "file": "src/click/utils.py",
            "test_file": "tests/click/test_utils.py",
        },
        "pytest-21": {
            "target": "get_app_dir",
            "line_range": "480-489",
            "file": "src/click/utils.py",
            "test_file": "tests/click/test_utils.py",
        },
        "pytest-22": {
            "target": "make_str",
            "line_range": "51-56",
            "file": "src/click/utils.py",
            "test_file": "tests/click/test_utils.py",
        },
        "pytest-23": {
            "target": "PacifyFlushWrapper.flush",
            "line_range": "516-517",
            "file": "src/click/utils.py",
            "test_file": "tests/click/test_utils.py",
        },
        "pytest-32": {
            "target": "safecall",
            "line_range": "39-44",
            "file": "src/click/utils.py",
            "test_file": "tests/click/test_utils.py",
        },
        "pytest-33": {
            "target": "_expand_args",
            "line_range": "619-620",
            "file": "src/click/utils.py",
            "test_file": "tests/click/test_utils.py",
        },
    },
)

assert_rows(
    "testdata/py-internal/internal-tasks.jsonl",
    {
        "pytest-internal-1": {
            "target": "PrivateService.__normalize",
            "line_range": "5-7",
            "file": "src/private_service.py",
            "test_file": "tests/test_private_service.py",
        },
    },
)

print("py regression fixture test passed")
PY
