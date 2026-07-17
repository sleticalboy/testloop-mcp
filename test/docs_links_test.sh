#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"

cd "$repo_root"

python3 - <<'PY'
from pathlib import Path
from urllib.parse import unquote
import re
import sys

repo_root = Path(".").resolve()
docs = [Path("README.md"), *sorted(Path("docs").glob("*.md"))]

link_re = re.compile(r"(?<!!)\[[^\]\n]+\]\(([^)\s]+)(?:\s+['\"][^)]*['\"])?\)")
scheme_re = re.compile(r"^[a-zA-Z][a-zA-Z0-9+.-]*:")


def strip_fenced_blocks(text: str) -> str:
    lines = []
    in_fence = False
    fence_marker = ""
    for line in text.splitlines():
        stripped = line.lstrip()
        if stripped.startswith("```") or stripped.startswith("~~~"):
            marker = stripped[:3]
            if not in_fence:
                in_fence = True
                fence_marker = marker
            elif marker == fence_marker:
                in_fence = False
                fence_marker = ""
            lines.append("")
            continue
        lines.append("" if in_fence else line)
    return "\n".join(lines)


def strip_inline_code(text: str) -> str:
    return re.sub(r"`[^`\n]*`", "", text)


def is_external_or_unsupported(target: str) -> bool:
    return (
        not target
        or target.startswith("#")
        or target.startswith("//")
        or target.startswith("/")
        or scheme_re.match(target) is not None
    )


failures = []
for doc in docs:
    text = strip_inline_code(strip_fenced_blocks(doc.read_text(encoding="utf-8")))
    for match in link_re.finditer(text):
        raw_target = match.group(1).strip("<>")
        if is_external_or_unsupported(raw_target):
            continue

        target_path = unquote(raw_target.split("#", 1)[0])
        if not target_path:
            continue

        resolved = (doc.parent / target_path).resolve()
        try:
            resolved.relative_to(repo_root)
        except ValueError:
            failures.append(f"{doc}: link escapes repo: {raw_target}")
            continue

        if not resolved.exists():
            failures.append(f"{doc}: missing link target: {raw_target}")

if failures:
    print("broken documentation links:", file=sys.stderr)
    for failure in failures:
        print(f"- {failure}", file=sys.stderr)
    sys.exit(1)

print(f"documentation link test passed ({len(docs)} markdown files)")
PY
