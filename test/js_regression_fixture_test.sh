#!/usr/bin/env sh
set -eu

python3 - <<'PY'
import json
from pathlib import Path


def load_jsonl(path):
    return [
        json.loads(line)
        for line in Path(path).read_text(encoding="utf-8").splitlines()
        if line.strip()
    ]


def assert_rows(path, expected):
    rows = load_jsonl(path)
    ids = [row.get("id") for row in rows]
    expected_ids = list(expected)
    if ids != expected_ids:
        raise SystemExit(f"{path}: ids={ids}, want={expected_ids}")
    for row in rows:
        task_id = row["id"]
        want = expected[task_id]
        for key in ("framework", "target", "line_range"):
            if row.get(key) != want[key]:
                raise SystemExit(f"{path}: {task_id} {key}={row.get(key)}, want {want[key]}")
        if want["file"] not in row.get("file", ""):
            raise SystemExit(f"{path}: {task_id} file does not point to {want['file']}")
        if want["test_file"] not in row.get("test_file", ""):
            raise SystemExit(f"{path}: {task_id} test_file does not point to {want['test_file']}")


assert_rows(
    "testdata/js-ip2region/ready-hit-tasks.jsonl",
    {
        "jest-1": {
            "framework": "jest",
            "target": "versionFromHeader",
            "line_range": "317-319",
            "file": "util.js",
            "test_file": "util.test.js",
        },
        "jest-2": {
            "framework": "jest",
            "target": "versionFromHeader",
            "line_range": "322-324",
            "file": "util.js",
            "test_file": "util.test.js",
        },
    },
)

assert_rows(
    "testdata/js-no-runtime/no-runtime-tasks.jsonl",
    {
        "jest-no-runtime-1": {
            "framework": "jest",
            "target": "events.ts",
            "line_range": "entire file",
            "file": "src/events.ts",
            "test_file": "tests/events.test.ts",
        },
    },
)

assert_rows(
    "testdata/js-internal/internal-tasks.jsonl",
    {
        "jest-internal-1": {
            "framework": "jest",
            "target": "hidden",
            "line_range": "5-7",
            "file": "src/helper.ts",
            "test_file": "tests/helper.test.ts",
        },
    },
)

assert_rows(
    "testdata/js-mcp-hub/repair-tasks.jsonl",
    {
        "vitest-mcp-hub-repair-1": {
            "framework": "vitest",
            "target": "ConfigManager.loadConfig",
            "line_range": "136-136",
            "file": "src/utils/config.js",
            "test_file": "tests/utils/config.test.js",
        },
        "vitest-mcp-hub-repair-2": {
            "framework": "vitest",
            "target": "ConfigManager.loadConfig",
            "line_range": "199-204",
            "file": "src/utils/config.js",
            "test_file": "tests/utils/config.test.js",
        },
        "vitest-mcp-hub-repair-3": {
            "framework": "vitest",
            "target": "ConfigManager.loadConfig",
            "line_range": "253-260",
            "file": "src/utils/config.js",
            "test_file": "tests/utils/config.test.js",
        },
    },
)

assert_rows(
    "testdata/js-mcp-hub/env-tasks.jsonl",
    {
        "vitest-mcp-hub-env-1": {
            "framework": "vitest",
            "target": "EnvResolver._resolveStringWithPlaceholders",
            "line_range": "219-223",
            "file": "src/utils/env-resolver.js",
            "test_file": "tests/utils/env-resolver.test.js",
        },
        "vitest-mcp-hub-env-2": {
            "framework": "vitest",
            "target": "EnvResolver._resolveStringWithPlaceholders",
            "line_range": "228-232",
            "file": "src/utils/env-resolver.js",
            "test_file": "tests/utils/env-resolver.test.js",
        },
    },
)

assert_rows(
    "testdata/js-mcp-hub/devwatcher-tasks.jsonl",
    {
        "vitest-mcp-hub-devwatcher-1": {
            "framework": "vitest",
            "target": "DevWatcher.stop",
            "line_range": "122-143",
            "file": "src/utils/dev-watcher.js",
            "test_file": "tests/utils/dev-watcher.test.js",
        },
        "vitest-mcp-hub-devwatcher-2": {
            "framework": "vitest",
            "target": "DevWatcher.start",
            "line_range": "72-75",
            "file": "src/utils/dev-watcher.js",
            "test_file": "tests/utils/dev-watcher.test.js",
        },
    },
)

assert_rows(
    "testdata/js-mcp-hub/sse-tasks.jsonl",
    {
        "vitest-mcp-hub-sse-1": {
            "framework": "vitest",
            "target": "SSEManager.setupAutoShutdown",
            "line_range": "80-103",
            "file": "src/utils/sse-manager.js",
            "test_file": "tests/utils/sse-manager.test.js",
        },
        "vitest-mcp-hub-sse-2": {
            "framework": "vitest",
            "target": "SSEManager.addConnection",
            "line_range": "170-185",
            "file": "src/utils/sse-manager.js",
            "test_file": "tests/utils/sse-manager.test.js",
        },
        "vitest-mcp-hub-sse-3": {
            "framework": "vitest",
            "target": "SSEManager.addConnection",
            "line_range": "150-160",
            "file": "src/utils/sse-manager.js",
            "test_file": "tests/utils/sse-manager.test.js",
        },
        "vitest-mcp-hub-sse-4": {
            "framework": "vitest",
            "target": "SSEManager.sendToClient",
            "line_range": "215-220",
            "file": "src/utils/sse-manager.js",
            "test_file": "tests/utils/sse-manager.test.js",
        },
    },
)

assert_rows(
    "testdata/js-mcp-hub/workspace-tasks.jsonl",
    {
        "vitest-mcp-hub-workspace-1": {
            "framework": "vitest",
            "target": "WorkspaceCacheManager.updateWorkspaceState",
            "line_range": "255-260",
            "file": "src/utils/workspace-cache.js",
            "test_file": "tests/utils/workspace-cache.test.js",
        },
        "vitest-mcp-hub-workspace-2": {
            "framework": "vitest",
            "target": "WorkspaceCacheManager.cleanupStaleEntries",
            "line_range": "226-233",
            "file": "src/utils/workspace-cache.js",
            "test_file": "tests/utils/workspace-cache.test.js",
        },
        "vitest-mcp-hub-workspace-3": {
            "framework": "vitest",
            "target": "WorkspaceCacheManager._withLock",
            "line_range": "399-421",
            "file": "src/utils/workspace-cache.js",
            "test_file": "tests/utils/workspace-cache.test.js",
        },
    },
)

print("js regression fixture test passed")
PY
