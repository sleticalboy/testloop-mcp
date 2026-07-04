# v0.3.0 发布说明草案

## 标题

testloop-mcp v0.3.0

## 摘要

v0.3.0 聚焦提升静态测试生成质量。这个版本让 Go/Python/Jest 生成器在简单可推断场景下生成更具体、更可执行的断言，而不是只给出宽泛类型检查或 TODO/skip 骨架。

## 主要变化

- Python 生成器会对简单 return 表达式生成精确断言。
- Jest 生成器会对简单 return 表达式生成精确断言。
- Python/Jest 边界用例会把边界值带入 return 表达式，生成更具体的断言。
- Go 内置生成器会为简单纯函数生成可执行表驱动 case，不再默认只生成 TODO/skip。
- Python/Jest 生成器会识别简单 if-return 分支，为普通路径和边界路径分别生成期望值。
- Go/Python/Jest 生成器新增 golden tests，固定代表性输出，降低生成质量回退风险。

## 示例

Python:

```python
def add(a, b):
    return a + b
```

生成结果会包含：

```python
result = add(1, 2)
assert result == (1 + 2)
```

Jest:

```js
function formatText(mode, prefix, text) {
  if (mode === 'short') {
    return prefix;
  }
  return prefix + text;
}
```

生成结果会分别覆盖普通路径和 `mode = 'short'` 边界路径。

Go:

```go
func Add(a, b int) int {
    return a + b
}
```

生成结果会包含 `skip: false` 的表驱动 case，并填入 `a: 1`、`b: 2`、`ret0: 1 + 2`。

## 已知限制

- 精确断言只覆盖简单、安全的 return 表达式。
- Python/Jest 分支识别目前只覆盖简单 if-return 结构。
- Go 内置生成器仍会对复杂函数、方法、变参、多返回、错误返回和不安全表达式保守生成 TODO/skip。

## 发布前验证

- [x] `go test ./...`
- [x] GitHub Actions CI passed

## 建议发布命令

```bash
git tag v0.3.0
git push origin v0.3.0
gh release create v0.3.0 --title "testloop-mcp v0.3.0" --notes-file docs/plan-release-notes-v0.3.0.md
```
