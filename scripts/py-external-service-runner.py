#!/usr/bin/env python3

import pathlib
import re
import sys


def main() -> int:
    if len(sys.argv) != 2:
        print("Usage: python scripts/py-external-service-runner.py <generated-test-file>", file=sys.stderr)
        return 2

    test_path = pathlib.Path(sys.argv[1])
    try:
        source = test_path.read_text(encoding="utf-8")
    except OSError as exc:
        print(f"ERROR {test_path}")
        print(str(exc))
        return 1

    test_id, test_name = pytest_test_id(test_path, source)
    print("============================= test session starts ==============================")
    print("platform darwin -- Python 3.x, pytest-8.x, pluggy-1.x")
    print(f"rootdir: {pathlib.Path.cwd()}")
    print("collected 1 item")
    print("")
    print(f"{test_id} FAILED                                              [100%]")
    print("=================================== FAILURES ===================================")
    print(f"___________________________ {test_name} ___________________________")
    print("E   TimeoutError: Timeout of 60000ms exceeded while waiting for storage endpoint download response")
    print("============================== 1 failed in 60.00s ==============================")
    return 1


def pytest_test_id(test_path: pathlib.Path, source: str) -> tuple[str, str]:
    rel_path = pathlib.Path(test_path)
    try:
        rel_path = test_path.resolve().relative_to(pathlib.Path.cwd().resolve())
    except ValueError:
        pass

    current_class = ""
    for line in source.splitlines():
        class_match = re.match(r"^class\s+(Test\w*)\b", line)
        if class_match:
            current_class = class_match.group(1)
            continue

        def_match = re.match(r"^(\s*)def\s+(test_\w+)\s*\(", line)
        if not def_match:
            continue

        if def_match.group(1) == "":
            current_class = ""
        test_name = def_match.group(2)
        if current_class:
            return f"{rel_path}::{current_class}::{test_name}", test_name
        return f"{rel_path}::{test_name}", test_name

    return f"{rel_path}::test_external_service_timeout", "test_external_service_timeout"


if __name__ == "__main__":
    raise SystemExit(main())
