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
        assert_task(path, row, want)


def assert_task(path, row, want):
    task_id = row["id"]
    if row.get("framework") != "junit":
        raise SystemExit(f"{path}: {task_id} framework={row.get('framework')}, want junit")
    for key in ("target", "line_range"):
        if row.get(key) != want[key]:
            raise SystemExit(f"{path}: {task_id} {key}={row.get(key)}, want {want[key]}")
    if want["file"] not in row.get("file", ""):
        raise SystemExit(f"{path}: {task_id} file does not point to {want['file']}")
    if want["test_file"] not in row.get("test_file", ""):
        raise SystemExit(f"{path}: {task_id} test_file does not point to {want['test_file']}")


assert_rows(
    "testdata/java-rocketmq-statuschecker/statuschecker-tasks.jsonl",
    {
        "junit-272": {
            "target": "StatusChecker.check",
            "line_range": "109-109",
            "file": "StatusChecker.java",
            "test_file": "StatusCheckerTest.java",
        },
        "junit-273": {
            "target": "StatusChecker.check",
            "line_range": "84-84",
            "file": "StatusChecker.java",
            "test_file": "StatusCheckerTest.java",
        },
        "junit-418": {
            "target": "StatusChecker.check",
            "line_range": "73-73",
            "file": "StatusChecker.java",
            "test_file": "StatusCheckerTest.java",
        },
        "junit-826": {
            "target": "StatusChecker.check",
            "line_range": "107-108",
            "file": "StatusChecker.java",
            "test_file": "StatusCheckerTest.java",
        },
    },
)

assert_rows(
    "testdata/java-commons-lang/ready-hit-tasks.jsonl",
    {
        "junit-44": {
            "target": "CharSequenceUtils.toCharArray",
            "line_range": "419-419",
            "file": "CharSequenceUtils.java",
            "test_file": "CharSequenceUtilsTestLoopTest.java",
        },
        "junit-50": {
            "target": "Failable.tryWithResources",
            "line_range": "651-651",
            "file": "Failable.java",
            "test_file": "FailableTestLoopTest.java",
        },
    },
)

assert_rows(
    "testdata/java-commons-lang/manual-internal-tasks.jsonl",
    {
        "junit-52": {
            "target": "TypeUtils.isAssignable",
            "line_range": "1028-1028",
            "file": "TypeUtils.java",
            "test_file": "TypeUtilsTestLoopTest.java",
        },
    },
)

assert_rows(
    "testdata/java-commons-codec/unreachable-tasks.jsonl",
    {
        "junit-130": {
            "target": "Metaphone.metaphone",
            "line_range": "279-279",
            "file": "Metaphone.java",
            "test_file": "MetaphoneTestLoopTest.java",
        },
    },
)

print("java regression fixture test passed")
PY
