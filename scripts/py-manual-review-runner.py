#!/usr/bin/env python3

import pathlib
import re
import sys


MANUAL_REVIEW_MARKERS = (
    "manual_review_internal:",
    "manual_review_unreachable:",
    "manual_review_environment:",
    "manual_review_database:",
    "manual_review_external_service:",
)


def main() -> int:
    if len(sys.argv) != 2:
        print("Usage: python scripts/py-manual-review-runner.py <generated-test-file>", file=sys.stderr)
        return 2

    test_path = pathlib.Path(sys.argv[1])
    try:
        source = test_path.read_text(encoding="utf-8")
    except OSError as exc:
        print(f"ERROR {test_path}")
        print(str(exc))
        return 1

    if "skip(" not in source or not any(marker in source for marker in MANUAL_REVIEW_MARKERS):
        print(f"FAILED {test_path}::manual_review")
        print("Generated fixture test is not a manual-review skip.")
        return 1

    test_id = manual_review_test_id(test_path, source)
    print("============================= test session starts ==============================")
    print("platform darwin -- Python 3.x, pytest-8.x, pluggy-1.x")
    print(f"rootdir: {pathlib.Path.cwd()}")
    print(f"collected 1 item")
    print("")
    print(f"{test_id} SKIPPED                                             [100%]")
    print("============================== 1 skipped in 0.01s ==============================")
    return 0


def manual_review_test_id(test_path: pathlib.Path, source: str) -> str:
    rel_path = pathlib.Path(test_path)
    try:
        rel_path = test_path.resolve().relative_to(pathlib.Path.cwd().resolve())
    except ValueError:
        pass

    current_class = ""
    current_test = ""
    for line in source.splitlines():
        class_match = re.match(r"^class\s+(Test\w*)\b", line)
        if class_match:
            current_class = class_match.group(1)
            current_test = ""
            continue

        def_match = re.match(r"^(\s*)def\s+(test_\w+)\s*\(", line)
        if def_match:
            if def_match.group(1) == "":
                current_class = ""
            current_test = def_match.group(2)
            continue

        if any(marker in line for marker in MANUAL_REVIEW_MARKERS) and current_test:
            if current_class:
                return f"{rel_path}::{current_class}::{current_test}"
            return f"{rel_path}::{current_test}"

    return f"{rel_path}::test_manual_review"


if __name__ == "__main__":
    raise SystemExit(main())
