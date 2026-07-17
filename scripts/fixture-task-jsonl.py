#!/usr/bin/env python3

import argparse
import json
import os
from pathlib import Path


def js_no_runtime(project_dir: str) -> dict:
    source = os.path.join(project_dir, "src", "events.ts")
    return {
        "id": "jest-no-runtime-1",
        "framework": "jest",
        "file": source,
        "target": "events.ts",
        "kind": "file_level",
        "line_range": "entire file",
        "gap_type": "no_runtime",
        "goal": "确认 events.ts 是 TypeScript 纯类型文件，没有可直接执行的运行时代码覆盖任务",
        "command": "node scripts/js-manual-review-runner.js tests/events.test.ts",
        "test_file": os.path.join(project_dir, "tests", "events.test.ts"),
        "test_name": "marks type-only module as no runtime coverage",
        "assertion_focus": [
            "纯类型声明不会产生有意义的本地 JavaScript coverage task，应通过消费方运行时测试或类型检查验证"
        ],
        "priority": 90,
        "priority_reason": "repository fixture for stable JS no-runtime manual-review smoke",
        "confidence": 0.9,
    }


def js_internal(project_dir: str) -> dict:
    source = os.path.join(project_dir, "src", "helper.ts")
    return {
        "id": "jest-internal-1",
        "framework": "jest",
        "file": source,
        "target": "hidden",
        "kind": "function",
        "line_range": "5-7",
        "gap_type": "branch",
        "missing_branches": ["未覆盖 if 分支: value === \"\""],
        "suggested_inputs": ["直接调用 hidden(\"\") 会命中分支，但 hidden 没有从 ESM 模块导出"],
        "goal": "确认未导出的 TypeScript helper 会被降级为 internal 手审任务",
        "command": "node scripts/js-manual-review-runner.js tests/helper.test.ts",
        "test_file": os.path.join(project_dir, "tests", "helper.test.ts"),
        "test_name": "marks unexported helper as internal manual review",
        "assertion_focus": [
            "未导出的 ESM helper 不能从外部生成测试直接 named import，应通过公开入口、测试 seam 或手审覆盖"
        ],
        "priority": 88,
        "priority_reason": "repository fixture for stable JS internal manual-review smoke",
        "confidence": 0.9,
    }


def js_mcp_hub_repair(project_dir: str) -> dict:
    source = os.path.join(project_dir, "src", "utils", "config.js")
    return {
        "id": "vitest-mcp-hub-repair-1",
        "framework": "vitest",
        "file": source,
        "target": "ConfigManager.loadConfig",
        "kind": "method",
        "line_range": "136-136",
        "gap_type": "branch",
        "missing_branches": [
            "未覆盖 if 分支: !this.configPaths || this.configPaths.length === 0"
        ],
        "uncovered_lines": [136],
        "suggested_inputs": [
            "构造没有 configPaths 或 configPaths 为空数组的 ConfigManager 实例"
        ],
        "goal": "确认真实 mcp-hub 项目中 ConfigManager.loadConfig 的错误路径会进入 repair_generated_test，而不是被当成 ready",
        "command": "npx vitest run tests/utils/config.test.js",
        "test_file": os.path.join(project_dir, "tests", "utils", "config.test.js"),
        "test_name": "covers ConfigManager.loadConfig empty config paths branch",
        "assertion_focus": [
            "当前静态生成器会直接 await loadConfig()，但该分支应断言 ConfigError，因此 run_tests 应返回失败并提示修生成测试"
        ],
        "priority": 121,
        "priority_reason": "real mcp-hub regression sample for failed/repair_generated_test coverage task",
        "confidence": 0.9,
    }


def py_internal(project_dir: str) -> dict:
    source = os.path.join(project_dir, "src", "private_service.py")
    return {
        "id": "pytest-internal-1",
        "framework": "pytest",
        "file": source,
        "target": "PrivateService.__normalize",
        "kind": "method",
        "line_range": "5-7",
        "gap_type": "branch",
        "missing_branches": ["未覆盖 if 分支: value == \"\""],
        "suggested_inputs": ["直接调用 __normalize(\"\") 会命中分支，但该方法会被 Python name mangling 隐藏"],
        "goal": "确认 Python 双下划线私有方法会被降级为 internal 手审任务",
        "command": "python3 scripts/py-manual-review-runner.py tests/test_private_service.py",
        "test_file": os.path.join(project_dir, "tests", "test_private_service.py"),
        "test_name": "test_private_method_requires_internal_review",
        "assertion_focus": [
            "Python name-mangled private method 不应从生成测试直接外部调用，应通过公开方法、测试 seam 或手审覆盖"
        ],
        "priority": 88,
        "priority_reason": "repository fixture for stable Python internal manual-review smoke",
        "confidence": 0.9,
    }


PRESETS = {
    "js-mcp-hub-repair": js_mcp_hub_repair,
    "js-no-runtime": js_no_runtime,
    "js-internal": js_internal,
    "py-internal": py_internal,
}


def main() -> int:
    parser = argparse.ArgumentParser(description="Generate a fixed fixture coverage task JSONL.")
    parser.add_argument("preset", choices=sorted(PRESETS))
    parser.add_argument("project_dir")
    parser.add_argument("output")
    args = parser.parse_args()

    project_dir = str(Path(args.project_dir).resolve())
    task = PRESETS[args.preset](project_dir)

    output = Path(args.output)
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text(json.dumps(task, ensure_ascii=False) + "\n", encoding="utf-8")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
