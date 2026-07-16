#!/usr/bin/env python3

import pathlib
import sys


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

    markers = (
        "manual_review_internal:",
        "manual_review_unreachable:",
        "manual_review_environment:",
        "manual_review_external_service:",
    )
    if "skip(" not in source or not any(marker in source for marker in markers):
        print(f"FAILED {test_path}::manual_review")
        print("Generated fixture test is not a manual-review skip.")
        return 1

    print("============================= test session starts ==============================")
    print(f"{test_path}::test_manual_review SKIPPED")
    print("============================== 1 skipped in 0.01s ==============================")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
