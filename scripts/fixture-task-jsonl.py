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
        "goal": "确认真实 mcp-hub 项目中 ConfigManager.loadConfig 的错误路径会生成 async reject 断言并进入 ready",
        "command": "npx vitest run tests/utils/config.test.js",
        "test_file": os.path.join(project_dir, "tests", "utils", "config.test.js"),
        "test_name": "covers ConfigManager.loadConfig empty config paths branch",
        "assertion_focus": [
            "该分支应断言 ConfigError，生成测试需要使用 await expect(instance.loadConfig()).rejects.toThrow()"
        ],
        "priority": 121,
        "priority_reason": "real mcp-hub regression sample for an async throwing branch that used to be repair_generated_test",
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


def py_apk_station_environment(project_dir: str) -> dict:
    source = os.path.join(project_dir, "app", "main.py")
    return {
        "id": "pytest-apk-frontend-env-1",
        "framework": "pytest",
        "file": source,
        "target": "serve_frontend",
        "kind": "function",
        "line_range": "84-89",
        "gap_type": "branch",
        "missing_branches": [
            "未覆盖 frontend/dist 存在时动态定义的 SPA fallback 分支"
        ],
        "suggested_inputs": [
            "在导入 app.main 前准备 frontend/dist/index.html，再通过 FastAPI TestClient 验证 fallback"
        ],
        "goal": "确认真实 haoy-apk-station FastAPI 项目中动态前端入口会稳定进入 manual_review_environment",
        "command": "python3 -m pytest {path}",
        "test_file": os.path.join(project_dir, "tests", "test_main_frontend_testloop.py"),
        "test_name": "test_serve_frontend_requires_frontend_dist_import_environment",
        "assertion_focus": [
            "serve_frontend 只有 frontend/dist 存在时才在模块导入阶段定义，静态生成测试应提示集成环境而不是直接调用 lifespan 或 app.main"
        ],
        "priority": 125,
        "priority_reason": "real haoy-apk-station FastAPI dynamic frontend route environment sample",
        "confidence": 0.9,
    }


def py_apk_station_external_service(project_dir: str) -> dict:
    source = os.path.join(project_dir, "app", "api", "apps.py")
    return {
        "id": "pytest-apk-download-external-1",
        "framework": "pytest",
        "file": source,
        "target": "download_apk",
        "kind": "function",
        "line_range": "550-570",
        "gap_type": "error_path",
        "missing_branches": [
            "未覆盖 urllib.request.urlopen(download_url, timeout=60) 外部下载 endpoint timeout 后回退重定向的路径"
        ],
        "suggested_inputs": [
            "构造 AppVersion 指向外部对象存储 endpoint，代理下载 timeout 后应回退 RedirectResponse"
        ],
        "goal": "确认真实 haoy-apk-station FastAPI 下载代理依赖外部对象存储 endpoint 的失败会进入 manual_review_external_service",
        "command": "python3 -m pytest {path}",
        "test_file": os.path.join(project_dir, "tests", "test_apps_download_external_testloop.py"),
        "test_name": "test_download_apk_external_endpoint_timeout_requires_integration_review",
        "assertion_focus": [
            "download_apk 代理下载依赖对象存储 endpoint 和 urllib timeout，应该通过 fake storage client 或集成环境验证，而不是把真实 timeout 当普通生成测试修复"
        ],
        "priority": 126,
        "priority_reason": "real haoy-apk-station FastAPI download proxy external storage endpoint sample",
        "confidence": 0.9,
    }


PRESETS = {
    "js-mcp-hub-repair": js_mcp_hub_repair,
    "js-no-runtime": js_no_runtime,
    "js-internal": js_internal,
    "py-apk-station-external-service": py_apk_station_external_service,
    "py-apk-station-environment": py_apk_station_environment,
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
