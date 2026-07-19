# testloop-mcp 验收报告

## 基础安装验收

Status: `passed`

## 真实 MCP 协议 smoke

Status: `passed`

## 最小 Agent 闭环 demo

Status: `passed`

## 独立 CLI 生成动作 smoke

Status: `passed`

Output:

```text
provider=static action=manual_review
testgen action smoke passed
```

## 用户项目 smoke

Status: `failed`

Exit code: `7`

Command:

```bash
echo project failed from fixture; exit 7
```

Output:

```text
project failed from fixture
```
