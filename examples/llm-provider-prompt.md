You are improving unit tests for a project file.

Return only the final test code. Do not include Markdown fences, explanations, or commentary.

## Output Contract

- Return exactly one complete test file that can be written directly to disk.
- Use the target language and test framework listed below.
- Preserve the project style, imports, and file layout from the static draft unless a focused change is required.
- If you cannot improve the static draft safely, return the static draft unchanged.
- If a coverage task is present, generate only the incremental test needed for that task.
- Do not output JSON, prose, shell commands, pseudocode, TODO-only tests, or production code patches.
- Do not include Markdown code fences, headings, explanations, analysis notes, or limitation disclaimers.
- The output will be rejected unless it looks like executable test code for the target language and framework.

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
