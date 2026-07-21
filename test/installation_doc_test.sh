#!/usr/bin/env sh
set -eu

python3 - <<'PY'
from pathlib import Path

doc = Path("docs/installation.md")
text = doc.read_text(encoding="utf-8")

required = [
    "brew install testloop-mcp",
    "curl -fsSL https://raw.githubusercontent.com/sleticalboy/testloop-mcp/main/scripts/install.sh | sh",
    "TESTLOOP_MCP_VERSION=v0.5.20",
    "testloop-mcp_v0.5.20_linux_amd64.tar.gz",
    "scripts/generate-homebrew-formula.sh v0.5.20",
    "docker compose up -d",
    "testloop-mcp --print-config=codex",
    "testloop-mcp --check-config ~/.codex/config.toml",
    "scripts/verify-client-setup.sh /absolute/path/to/testloop-mcp",
    "TESTLOOP_MCP_VERIFY_EXPECT_VERSION=0.5.20",
    "go run ./examples/agent-response-manifest-demo",
    "docs/fixtures/agent-response-artifact-manifest.json",
    "agent-response-artifact-manifest.schema.json",
    "./quickstart.md",
    "./mcp-client-contract-tests.md",
]

missing = [item for item in required if item not in text]
if missing:
    print("installation doc test failed:")
    for item in missing:
        print(f"- missing {item}")
    raise SystemExit(1)

for path in [
    Path("docs/quickstart.md"),
    Path("docs/mcp-client-contract-tests.md"),
    Path("docs/fixtures/agent-response-artifact-manifest.json"),
    Path("docs/fixtures/agent-response-artifact-manifest.schema.json"),
    Path("examples/agent-response-manifest-demo/main.go"),
]:
    if not path.exists():
        print(f"missing referenced file: {path}")
        raise SystemExit(1)

print("installation doc test passed")
PY
