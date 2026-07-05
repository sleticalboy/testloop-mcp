# LLM Provider 接入说明

`generate_tests` 默认使用 `static` provider，不依赖任何外部 LLM。需要接入自定义 LLM 时，可以在服务端配置 `TESTLOOP_LLM_PROVIDER_CMD`，再调用 `generate_tests` 时传入 `provider: "llm"` 或 `provider: "auto"`。

## Provider 模式

| provider | 行为 |
| --- | --- |
| `static` | 默认值。只使用内置静态生成器；Go 普通生成优先走 `gotests`，传入 `coverage_task` 时各语言会优先按任务目标生成增量测试草稿。 |
| `llm` | 必须配置 `TESTLOOP_LLM_PROVIDER_CMD`，由外部命令返回最终测试代码。 |
| `auto` | 配置了 `TESTLOOP_LLM_PROVIDER_CMD` 时走 LLM provider，否则自动回退 `static`。 |

## 命令协议

服务端会启动 `TESTLOOP_LLM_PROVIDER_CMD` 指定的命令，并向 stdin 写入 JSON：

```json
{
  "source_file": "src/calc.py",
  "context": {
    "language": "python",
    "framework": "pytest",
    "source_file": "src/calc.py",
    "imports": [],
    "types": [],
    "targets": [],
    "coverage_task": {
      "id": "pytest-1",
      "framework": "pytest",
      "file": "src/calc.py",
      "target": "add",
      "line_range": "2-2",
      "gap_type": "return_path",
      "test_file": "tests/test_calc.py",
      "test_name": "test_add_covers_gap",
      "suggested_inputs": ["构造满足条件 `a == 0` 的输入"],
      "assertion_focus": ["断言未覆盖返回路径的具体结果"],
      "priority": 100
    }
  },
  "static_code": "from calc import add\n\n\ndef test_add():\n    ..."
}
```

`coverage_task` 只会在 MCP 调用方传入单个覆盖率任务时出现。外部 LLM provider 应优先遵守其中的 `target`、`test_file`、`test_name`、`suggested_inputs` 和 `assertion_focus`，并把 `static_code` 当作可修改草稿，而不是重新生成整文件测试。

provider 的 stdout 支持两种返回格式：

1. 直接返回测试代码。
2. 返回 JSON：`{"code":"..."}`。

stderr 会作为失败信息返回给 MCP 调用方。stdout 为空会被视为失败。

## 使用示例

```bash
export TESTLOOP_LLM_PROVIDER_CMD="sh examples/llm-provider.sh"
```

调用 `generate_tests`：

```json
{
  "file_path": "demo/calc.py",
  "provider": "auto"
}
```

`examples/llm-provider.sh` 是一个最小示例：它会读取 stdin JSON，并直接返回 `static_code`。真实接入 OpenAI、Ollama、Claude 或内部模型时，可以在这个脚本里把 `context` 和 `static_code` 组装成 prompt，再把模型返回的测试代码写到 stdout。

## 设计约束

- MCP 请求不能直接传任意命令，命令只能由服务端环境变量配置，避免把 `generate_tests` 变成远程命令执行入口。
- provider 应只输出测试代码，不要输出解释性文本，否则会被写入测试文件。
- `static_code` 是可用回退结果，LLM provider 可以基于它做增强，而不是从零生成。
- 当存在 `context.coverage_task` 时，provider 应只补充该任务对应的增量测试，避免覆盖或扩写成整文件测试套件。
