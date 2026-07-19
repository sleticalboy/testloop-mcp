#!/usr/bin/env sh
set -eu

python3 - <<'PY'
import json
from pathlib import Path

path = Path("testdata/java-rocketmq-statuschecker/statuschecker-tasks.jsonl")
rows = [json.loads(line) for line in path.read_text(encoding="utf-8").splitlines() if line.strip()]

ids = [row.get("id") for row in rows]
expected_ids = ["junit-272", "junit-273", "junit-418", "junit-826"]
if ids != expected_ids:
    raise SystemExit(f"{path}: ids={ids}, want={expected_ids}")

expected_ranges = {
    "junit-272": "109-109",
    "junit-273": "84-84",
    "junit-418": "73-73",
    "junit-826": "107-108",
}
for row in rows:
    task_id = row["id"]
    if row.get("framework") != "junit":
        raise SystemExit(f"{path}: {task_id} framework={row.get('framework')}, want junit")
    if row.get("target") != "StatusChecker.check":
        raise SystemExit(f"{path}: {task_id} target={row.get('target')}, want StatusChecker.check")
    if row.get("line_range") != expected_ranges[task_id]:
        raise SystemExit(f"{path}: {task_id} line_range={row.get('line_range')}, want {expected_ranges[task_id]}")
    if "StatusChecker.java" not in row.get("file", ""):
        raise SystemExit(f"{path}: {task_id} file does not point to StatusChecker.java")
    if "StatusCheckerTest.java" not in row.get("test_file", ""):
        raise SystemExit(f"{path}: {task_id} test_file does not point to StatusCheckerTest.java")

print("java regression fixture test passed")
PY
