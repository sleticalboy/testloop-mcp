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
  "source_file": "src/api.ts",
  "context": {
    "language": "typescript",
    "framework": "vitest",
    "source_file": "src/api.ts",
    "imports": ["import type { ExternalUser } from './types'"],
    "types": [],
    "targets": [
      {
        "name": "loadUser",
        "kind": "function",
        "params": ["response"],
        "async": true,
        "return_type": "object",
        "return_type_expr": "Promise<ExternalUser>",
        "payload_notes": [
          "return annotation ExternalUser is not declared in the same source file; static payload falls back to { ok: true }",
          "return annotation references imported type ExternalUser from './types'; read candidate source files: types.ts, types.tsx, types.d.ts, types.js, types.jsx, types.mjs, types.cjs, types/index.ts, types/index.tsx, types/index.d.ts, types/index.js, types/index.jsx, types/index.mjs, types/index.cjs"
        ],
        "return_expressions": ["await response.json()"]
      }
    ],
    "coverage_task": {
      "id": "vitest-1",
      "framework": "vitest",
      "file": "src/api.ts",
      "target": "loadUser",
      "line_range": "8-8",
      "gap_type": "return_path",
      "test_file": "src/api.test.ts",
      "test_name": "covers loadUser response payload",
      "suggested_inputs": ["构造带 json() 方法的 Response-like 输入"],
      "assertion_focus": ["断言未覆盖返回路径的具体结果"],
      "priority": 100
    }
  },
  "static_code": "import { describe, it, expect } from 'vitest';\nimport { loadUser } from './api';\n\n..."
}
```

`coverage_task` 只会在 MCP 调用方传入单个覆盖率任务时出现。外部 LLM provider 应优先遵守其中的 `target`、`test_file`、`test_name`、`suggested_inputs` 和 `assertion_focus`，并把 `static_code` 当作可修改草稿，而不是重新生成整文件测试。

JS/TS 目标中的 `return_type_expr` 会保留 TypeScript 返回注解。`payload_notes` 会解释静态 payload 的保守边界，例如跨文件类型、约束泛型、动态 indexed access 或 `keyof`；当返回注解引用 imported type 时，也会给出 import 来源和候选源码文件。provider 可以据此读取更多项目上下文或保留静态草稿的保守 mock。

provider 的 stdout 支持两种返回格式：

1. 直接返回测试代码。
2. 返回 JSON：`{"code":"..."}`。

provider 输出会自动清洗常见 Markdown 代码围栏和前后解释性文本。例如模型返回：

````markdown
下面是测试代码：

```ts
import { describe, it, expect } from 'vitest';

it('loads user', async () => {
  // ...
});
```
````

最终只会写入代码围栏内的内容。stderr 会作为失败信息返回给 MCP 调用方。stdout 为空、JSON 中缺少 `code`、或清洗后没有可识别测试代码都会被视为失败。

清洗后还会按目标语言做一层轻量测试代码校验：

| 目标语言 | 最低识别信号 |
| --- | --- |
| Go | `func Test...(` |
| Python | `def test_...(` 或 `async def test_...(` |
| JavaScript / TypeScript | `describe(...)`、`it(...)`、`test(...)`、`*.test(...)` 或 `expect(...)` |
| Rust | `#[test]`、`#[tokio::test]` 或 `fn test_...(` |
| Java | `@Test` 或 JUnit `Test` import |

这只是防止解释文本、业务实现片段或调试脚本被误写入测试文件的后验保护，不替代 `run_tests` 的真实编译和执行。

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

示例脚本还会消费 `payload_notes` 中的 `read candidate source files: ...` 提示：当候选文件存在于 `source_file` 同目录或子目录时，会读取这些文件并放入 prompt 的 `Imported Type Context` 小节。默认情况下 stdout 仍只返回 `static_code`，不会把 prompt 写入测试文件。

默认 prompt 模板位于 `examples/llm-provider-prompt.md`。模板支持这些占位符：

| 占位符 | 含义 |
| --- | --- |
| `{{SOURCE_FILE}}` | 当前源码文件路径 |
| `{{LANGUAGE}}` | 目标语言 |
| `{{FRAMEWORK}}` | 测试框架 |
| `{{REQUEST_JSON}}` | 完整 provider 请求 JSON |
| `{{STATIC_CODE}}` | 内置 static provider 生成的测试草稿 |
| `{{IMPORTED_TYPE_CONTEXT}}` | 根据 `payload_notes` 读取到的候选类型文件内容 |
| `{{COVERAGE_TASK_JSON}}` | 单个 coverage task JSON；没有任务时为 `{}` |

调试 prompt：

```bash
TESTLOOP_LLM_PROVIDER_PROMPT_FILE=/tmp/testloop-prompt.md \
  TESTLOOP_LLM_PROVIDER_CMD="sh examples/llm-provider.sh" \
  testloop-mcp
```

替换 prompt 模板：

```bash
TESTLOOP_LLM_PROVIDER_PROMPT_TEMPLATE=/path/to/prompt.md \
  TESTLOOP_LLM_PROVIDER_CMD="sh examples/llm-provider.sh" \
  testloop-mcp
```

接入任意真实模型命令：

```bash
TESTLOOP_LLM_PROVIDER_MODEL_CMD="your-model-cli --generate-tests" \
  TESTLOOP_LLM_PROVIDER_CMD="sh examples/llm-provider.sh" \
  testloop-mcp
```

`TESTLOOP_LLM_PROVIDER_MODEL_CMD` 会从 stdin 收到完整 prompt，并应在 stdout 输出最终测试代码。

Ollama 示例：

```bash
TESTLOOP_OLLAMA_MODEL=qwen2.5-coder:7b \
  TESTLOOP_LLM_PROVIDER_MODEL_CMD="sh examples/model-ollama.sh" \
  TESTLOOP_LLM_PROVIDER_CMD="sh examples/llm-provider.sh" \
  testloop-mcp
```

`examples/model-ollama.sh` 会执行 `ollama run "$TESTLOOP_OLLAMA_MODEL"`，并把 prompt 通过 stdin 传给 Ollama。未设置模型时默认使用 `qwen2.5-coder:7b`。

OpenAI CLI 示例：

```bash
TESTLOOP_OPENAI_MODEL=gpt-5.5 \
  TESTLOOP_LLM_PROVIDER_MODEL_CMD="sh examples/model-openai-cli.sh" \
  TESTLOOP_LLM_PROVIDER_CMD="sh examples/llm-provider.sh" \
  testloop-mcp
```

`examples/model-openai-cli.sh` 会调用官方 `openai responses create` 命令，默认通过 `--transform 'output.#(type=="message").content.0.text'` 只取模型文本输出。可以用 `TESTLOOP_OPENAI_MAX_OUTPUT_TOKENS` 调整最大输出长度。

## 设计约束

- MCP 请求不能直接传任意命令，命令只能由服务端环境变量配置，避免把 `generate_tests` 变成远程命令执行入口。
- provider 应优先只输出测试代码；常见 Markdown 代码围栏会被清洗，但不要依赖模型输出长篇解释。
- `static_code` 是可用回退结果，LLM provider 可以基于它做增强，而不是从零生成。
- 当存在 `context.coverage_task` 时，provider 应只补充该任务对应的增量测试，避免覆盖或扩写成整文件测试套件。
- `examples/model-ollama.sh` 和 `examples/model-openai-cli.sh` 是模型命令包装层，不直接处理 MCP provider JSON；它们只接收 prompt 并输出测试代码。
