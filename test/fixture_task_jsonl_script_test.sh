#!/usr/bin/env sh
set -eu

help_output="$(scripts/fixture-task-jsonl.py --help)"

printf '%s\n' "$help_output" | grep -F "Regenerate static regression fixture coverage task JSONL." >/dev/null
printf '%s\n' "$help_output" | grep -F "Default smoke scripts read checked-in testdata/*.jsonl files." >/dev/null
printf '%s\n' "$help_output" | grep -F "Use this helper only when rebuilding or adding fixture inputs." >/dev/null
printf '%s\n' "$help_output" | grep -F "output JSONL path, normally under testdata/" >/dev/null

echo "fixture task jsonl script test passed"
