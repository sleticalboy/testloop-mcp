#!/bin/sh
set -eu

model="${TESTLOOP_OPENAI_MODEL:-gpt-5.5}"
max_output_tokens="${TESTLOOP_OPENAI_MAX_OUTPUT_TOKENS:-4096}"

tmp_prompt="$(mktemp)"
tmp_body="$(mktemp)"
trap 'rm -f "$tmp_prompt" "$tmp_body"' EXIT INT TERM
cat > "$tmp_prompt"

if [ "${TESTLOOP_MODEL_DRY_RUN:-}" = "1" ]; then
    cat <<'EOF'
it('dry run prompt received', () => {
  expect(true).toBe(true);
});
EOF
    exit 0
fi

{
    printf 'model: %s\n' "$model"
    printf 'max_output_tokens: %s\n' "$max_output_tokens"
    printf 'instructions: |\n'
    printf '  Return only the final test code. Do not include Markdown fences, explanations, or commentary.\n'
    printf 'input: |\n'
    sed 's/^/  /' "$tmp_prompt"
} > "$tmp_body"

exec openai responses create \
    --format raw \
    --transform 'output.#(type=="message").content.0.text' \
    < "$tmp_body"
