#!/usr/bin/env python3
import json
import sys
from pathlib import Path


def usage() -> None:
    print("Usage: scripts/render-regression-preflight-report.py <preflight-json|->", file=sys.stderr)


def load_payload(path: str) -> dict:
    if path == "-":
        return json.load(sys.stdin)
    return json.loads(Path(path).read_text(encoding="utf-8"))


def missing_items(payload: dict) -> list[dict]:
    items = payload.get("missing") or []
    if not isinstance(items, list):
        return []
    return [item for item in items if isinstance(item, dict)]


def render_missing_group(title: str, kind: str, items: list[dict]) -> list[str]:
    group = [item for item in items if item.get("kind") == kind]
    if not group:
        return []

    lines = [f"### {title}"]
    for item in group:
        label = item.get("label") or "-"
        value = item.get("value") or "-"
        lines.append(f"- `{label}`: `{value}`")
    lines.append("")
    return lines


def render(payload: dict) -> str:
    ok = payload.get("ok") is True
    missing = missing_items(payload)
    missing_count = payload.get("missing_count", len(missing))

    lines = [
        "## Regression Smoke 前置检查",
        "",
        f"- 状态：{'通过' if ok else '未通过'}",
        f"- 缺失项：{missing_count}",
        "",
    ]

    if ok:
        lines.extend(
            [
                "可以继续运行：",
                "",
                "```bash",
                "scripts/validate-regression-smoke.sh",
                "```",
                "",
            ]
        )
        return "\n".join(lines).rstrip() + "\n"

    lines.extend(
        [
            "请先补齐以下前置条件，或通过对应 `TESTLOOP_*_REGRESSION_*` 环境变量改到本机实际路径。",
            "",
        ]
    )
    lines.extend(render_missing_group("缺失命令", "command", missing))
    lines.extend(render_missing_group("缺失目录", "dir", missing))
    lines.extend(render_missing_group("缺失 JSONL fixture", "file", missing))
    lines.extend(
        [
            "补齐后重新运行：",
            "",
            "```bash",
            "TESTLOOP_REGRESSION_PREFLIGHT_FORMAT=json scripts/validate-regression-preflight.sh | scripts/render-regression-preflight-report.py -",
            "```",
            "",
        ]
    )
    return "\n".join(lines).rstrip() + "\n"


def main() -> int:
    if len(sys.argv) != 2:
        usage()
        return 2
    payload = load_payload(sys.argv[1])
    sys.stdout.write(render(payload))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
