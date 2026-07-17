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
    test_file = os.path.join(project_dir, "tests", "utils", "config.test.js")
    common = {
        "framework": "vitest",
        "file": source,
        "target": "ConfigManager.loadConfig",
        "kind": "method",
        "gap_type": "branch",
        "command": "npx vitest run tests/utils/config.test.js",
        "test_file": test_file,
        "assertion_focus": [
            "该分支应断言 ConfigError，生成测试需要使用 await expect(instance.loadConfig()).rejects.toThrow()"
        ],
        "confidence": 0.9,
    }
    return [
        {
            **common,
            "id": "vitest-mcp-hub-repair-1",
            "line_range": "136-136",
            "missing_branches": [
                "未覆盖 if 分支: !this.configPaths || this.configPaths.length === 0"
            ],
            "uncovered_lines": [136],
            "suggested_inputs": [
                "构造没有 configPaths 或 configPaths 为空数组的 ConfigManager 实例"
            ],
            "goal": "确认真实 mcp-hub 项目中 ConfigManager.loadConfig 的空路径错误会生成 async reject 断言并进入 ready",
            "test_name": "covers ConfigManager.loadConfig empty config paths branch",
            "priority": 121,
            "priority_reason": "real mcp-hub regression sample for an async throwing empty-path branch that used to be repair_generated_test",
        },
        {
            **common,
            "id": "vitest-mcp-hub-repair-2",
            "line_range": "199-204",
            "missing_branches": [
                "未覆盖 if 分支: hasStdioFields && hasSseFields"
            ],
            "uncovered_lines": [199, 200, 201, 202, 203, 204],
            "suggested_inputs": [
                "构造同时包含 command 和 url 的 mcpServers 配置文件"
            ],
            "goal": "确认真实 mcp-hub 项目中 ConfigManager.loadConfig 的 stdio/sse 混用错误路径会生成配置文件输入并进入 ready",
            "test_name": "covers ConfigManager.loadConfig mixed stdio and sse branch",
            "priority": 122,
            "priority_reason": "real mcp-hub regression sample for a config-file-driven async validation branch that used to be repair_generated_test",
        },
        {
            **common,
            "id": "vitest-mcp-hub-repair-3",
            "line_range": "253-260",
            "missing_branches": [
                "未覆盖 else 分支: server missing both command and url"
            ],
            "uncovered_lines": [253, 254, 255, 256, 257, 258, 259, 260],
            "suggested_inputs": [
                "构造 mcpServers.test 为空对象的配置文件"
            ],
            "goal": "确认真实 mcp-hub 项目中 ConfigManager.loadConfig 的缺少 transport 配置错误路径会生成配置文件输入并进入 ready",
            "test_name": "covers ConfigManager.loadConfig missing transport branch",
            "priority": 123,
            "priority_reason": "real mcp-hub regression sample for a config-file-driven async validation branch that used to be repair_generated_test",
        },
    ]


def js_mcp_hub_env(project_dir: str) -> dict:
    source = os.path.join(project_dir, "src", "utils", "env-resolver.js")
    test_file = os.path.join(project_dir, "tests", "utils", "env-resolver.test.js")
    common = {
        "framework": "vitest",
        "file": source,
        "target": "EnvResolver._resolveStringWithPlaceholders",
        "kind": "method",
        "command": "npx vitest run tests/utils/env-resolver.test.js",
        "test_file": test_file,
        "confidence": 0.9,
    }
    return [
        {
            **common,
            "id": "vitest-mcp-hub-env-1",
            "line_range": "219-223",
            "gap_type": "branch",
            "missing_branches": [
                "未覆盖 if 分支: resolvedValue === undefined"
            ],
            "uncovered_lines": [219, 220, 221, 222, 223],
            "suggested_inputs": [
                "构造字符串 '${MISSING_TOKEN}' 且 context 不包含 MISSING_TOKEN"
            ],
            "goal": "确认真实 mcp-hub 项目中 EnvResolver 严格模式变量缺失分支会生成 async reject 断言并进入 ready",
            "test_name": "covers EnvResolver unresolved strict placeholder branch",
            "assertion_focus": [
                "该分支应断言 Variable 'MISSING_TOKEN' not found"
            ],
            "priority": 124,
            "priority_reason": "real mcp-hub regression sample for strict placeholder resolution branch that used to be repair_generated_test",
        },
        {
            **common,
            "id": "vitest-mcp-hub-env-2",
            "line_range": "228-232",
            "gap_type": "error_path",
            "missing_branches": [
                "未覆盖 if 分支: isCommand"
            ],
            "uncovered_lines": [228, 229, 230, 231, 232],
            "suggested_inputs": [
                "构造字符串 '${cmd: failing-command}' 触发命令执行失败"
            ],
            "goal": "确认真实 mcp-hub 项目中 EnvResolver 命令执行失败分支不会被 maxPasses 错误路径抢占，并进入 ready",
            "test_name": "covers EnvResolver command failure branch",
            "assertion_focus": [
                "该分支应断言 cmd execution failed"
            ],
            "priority": 125,
            "priority_reason": "real mcp-hub regression sample for command-execution error path that previously passed through the wrong maxPasses branch",
        },
    ]


def js_mcp_hub_workspace(project_dir: str) -> dict:
    source = os.path.join(project_dir, "src", "utils", "workspace-cache.js")
    test_file = os.path.join(project_dir, "tests", "utils", "workspace-cache.test.js")
    common = {
        "framework": "vitest",
        "file": source,
        "kind": "method",
        "command": "npx vitest run tests/utils/workspace-cache.test.js",
        "test_file": test_file,
        "confidence": 0.9,
    }
    return [
        {
            **common,
            "id": "vitest-mcp-hub-workspace-1",
            "target": "WorkspaceCacheManager.updateWorkspaceState",
            "line_range": "255-260",
            "gap_type": "branch",
            "missing_branches": [
                "未覆盖 if 分支: cache[workspaceKey]"
            ],
            "uncovered_lines": [255, 256, 257, 258, 259, 260],
            "suggested_inputs": [
                "构造满足条件 `cache[workspaceKey]` 的输入",
                "设置 port 覆盖未执行分支",
                "设置 updates 覆盖未执行分支",
            ],
            "goal": "确认真实 mcp-hub 项目中 WorkspaceCacheManager.updateWorkspaceState 通过 mocked cache/lock 进入 ready",
            "test_name": "covers WorkspaceCacheManager update existing workspace",
            "assertion_focus": [
                "应 mock _withLock/_readCache/_writeCache，避免触碰真实 XDG state 文件"
            ],
            "priority": 126,
            "priority_reason": "real mcp-hub workspace cache update sample requiring filesystem side-effect isolation",
        },
        {
            **common,
            "id": "vitest-mcp-hub-workspace-2",
            "target": "WorkspaceCacheManager.cleanupStaleEntries",
            "line_range": "226-233",
            "gap_type": "branch",
            "missing_branches": [
                "未覆盖 else 分支: process no longer running"
            ],
            "uncovered_lines": [226, 227, 228, 229, 230, 231, 232, 233],
            "suggested_inputs": [
                "构造 cache 包含 pid 不存在的 workspace entry",
                "mock _isProcessRunning 返回 false",
            ],
            "goal": "确认真实 mcp-hub 项目中 WorkspaceCacheManager.cleanupStaleEntries stale pid 分支不会调用真实 process.kill，并进入 ready",
            "test_name": "covers WorkspaceCacheManager cleanup stale entries branch",
            "assertion_focus": [
                "应 mock _withLock/_readCache/_writeCache/_isProcessRunning，避免真实文件锁和真实进程探测"
            ],
            "priority": 127,
            "priority_reason": "real mcp-hub workspace cache stale-process sample requiring process/filesystem side-effect isolation",
        },
        {
            **common,
            "id": "vitest-mcp-hub-workspace-3",
            "target": "WorkspaceCacheManager._withLock",
            "line_range": "399-421",
            "gap_type": "error_path",
            "missing_branches": [
                "未覆盖 stale lock cleanup retry failure path"
            ],
            "uncovered_lines": [399, 400, 401, 407, 413, 416, 421],
            "suggested_inputs": [
                "构造 lockFilePath 指向不可写或持续 EEXIST 的 lock 文件",
                "mock fs.writeFile/unlink",
            ],
            "goal": "确认 _withLock 文件锁重试依赖真实计时和文件系统，应进入环境手审而不是直接写真实 lock 文件",
            "test_name": "marks WorkspaceCacheManager lock retry path as environment manual review",
            "assertion_focus": [
                "文件锁重试路径依赖真实计时、lock 文件和 fs exclusive write，应使用集成 fixture 或手审"
            ],
            "priority": 128,
            "priority_reason": "real mcp-hub workspace cache lock retry path should avoid unsafe generated filesystem mutation",
        },
    ]


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


def py_apk_station_database(project_dir: str) -> dict:
    source = os.path.join(project_dir, "app", "api", "apps.py")
    return {
        "id": "pytest-apk-delete-db-1",
        "framework": "pytest",
        "file": source,
        "target": "delete_app",
        "kind": "function",
        "line_range": "668-672",
        "gap_type": "error_path",
        "missing_branches": [
            "未覆盖 db.query(...).delete() / db.delete(app) / db.commit() 抛出 SQLAlchemyError 后的数据库事务失败路径"
        ],
        "suggested_inputs": [
            "构造 db.commit 抛出 SQLAlchemyError 的 Session，验证删除应用事务失败时的行为"
        ],
        "goal": "确认真实 haoy-apk-station FastAPI 删除应用路径的数据库事务错误会进入 manual_review_database",
        "command": "python3 -m pytest {path}",
        "test_file": os.path.join(project_dir, "tests", "test_apps_delete_database_testloop.py"),
        "test_name": "test_delete_app_database_commit_failure_requires_review",
        "assertion_focus": [
            "delete_app 同时删除版本、下载日志和应用记录，db.commit 失败依赖 SQLAlchemy session/事务行为，应通过测试数据库或注入 session 验证"
        ],
        "priority": 127,
        "priority_reason": "real haoy-apk-station FastAPI delete_app database transaction sample",
        "confidence": 0.9,
    }


PRESETS = {
    "js-mcp-hub-env": js_mcp_hub_env,
    "js-mcp-hub-repair": js_mcp_hub_repair,
    "js-mcp-hub-workspace": js_mcp_hub_workspace,
    "js-no-runtime": js_no_runtime,
    "js-internal": js_internal,
    "py-apk-station-database": py_apk_station_database,
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
    tasks = PRESETS[args.preset](project_dir)
    if isinstance(tasks, dict):
        tasks = [tasks]

    output = Path(args.output)
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text(
        "".join(json.dumps(task, ensure_ascii=False) + "\n" for task in tasks),
        encoding="utf-8",
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
