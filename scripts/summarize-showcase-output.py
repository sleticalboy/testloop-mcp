#!/usr/bin/env python3
import json
import sys
from collections import Counter


def usage() -> None:
    print("Usage: scripts/summarize-showcase-output.py <output-jsonl> [expected-actions]", file=sys.stderr)


def load_rows(path: str) -> list[dict]:
    rows = []
    with open(path, "r", encoding="utf-8") as fh:
        for line in fh:
            line = line.strip()
            if line:
                rows.append(json.loads(line))
    return rows


def task_id(row: dict) -> str:
    return (row.get("coverage_task") or {}).get("id", "")


def summarize(path: str, rows: list[dict]) -> dict:
    status_counts = Counter(row.get("status", "") for row in rows)
    action_counts = Counter(row.get("action", "") for row in rows)
    tasks = [
        {
            "id": task_id(row),
            "target": (row.get("coverage_task") or {}).get("target", ""),
            "line_range": (row.get("coverage_task") or {}).get("line_range", ""),
            "status": row.get("status", ""),
            "action": row.get("action", ""),
            "skipped": (row.get("run_result") or {}).get("skipped", 0),
        }
        for row in rows
    ]
    return {
        "output": path,
        "total": len(rows),
        "status_counts": dict(status_counts),
        "action_counts": dict(action_counts),
        "tasks": tasks,
    }


def expectation_failures(rows: list[dict], expected_actions: str) -> list[str]:
    by_id = {task_id(row): row for row in rows}
    failures = []
    for item in [part.strip() for part in expected_actions.split(",") if part.strip()]:
        if "=" not in item:
            failures.append(f"invalid expectation {item!r}, expected task-id=action")
            continue
        expected_id, expected_action = [part.strip() for part in item.split("=", 1)]
        row = by_id.get(expected_id)
        if not row:
            failures.append(f"missing expected task {expected_id!r}")
            continue
        actual_action = row.get("action", "")
        actual_status = row.get("status", "")
        if actual_action != expected_action:
            failures.append(f"{expected_id}: action={actual_action!r}, expected {expected_action!r}")
        if actual_status != "passed":
            failures.append(f"{expected_id}: status={actual_status!r}, expected 'passed'")
    return failures


def main() -> int:
    if len(sys.argv) not in (2, 3):
        usage()
        return 2

    path = sys.argv[1]
    expected_actions = sys.argv[2] if len(sys.argv) == 3 else ""
    rows = load_rows(path)
    print("showcase_summary=" + json.dumps(summarize(path, rows), ensure_ascii=False, sort_keys=True))

    failures = expectation_failures(rows, expected_actions)
    if failures:
        print("showcase_expectations_failed:", file=sys.stderr)
        for failure in failures:
            print(f"- {failure}", file=sys.stderr)
        return 1
    if expected_actions:
        print("showcase_expectations=pass")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
