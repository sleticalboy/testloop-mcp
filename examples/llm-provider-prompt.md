You are improving unit tests for a project file.

Return only the final test code. Do not include Markdown fences, explanations, or commentary.

## Target

- Source file: `{{SOURCE_FILE}}`
- Language: `{{LANGUAGE}}`
- Test framework: `{{FRAMEWORK}}`

## Rules

- Start from the static draft and keep imports, framework style, and file layout compatible with the project.
- If a coverage task is present, generate only the incremental test for that task.
- Use imported type context only when it is relevant to assertions or mock payloads.
- Prefer deterministic values and focused assertions.
- Do not rewrite production code.

## Coverage Task

```json
{{COVERAGE_TASK_JSON}}
```

## Static Draft

```text
{{STATIC_CODE}}
```

## Imported Type Context

{{IMPORTED_TYPE_CONTEXT}}

## Full Request JSON

```json
{{REQUEST_JSON}}
```
