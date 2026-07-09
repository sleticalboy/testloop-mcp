#!/bin/sh
set -eu

model="${TESTLOOP_OLLAMA_MODEL:-qwen2.5-coder:7b}"

tmp_prompt="$(mktemp)"
trap 'rm -f "$tmp_prompt"' EXIT INT TERM
cat > "$tmp_prompt"

if [ "${TESTLOOP_MODEL_DRY_RUN:-}" = "1" ]; then
    cat <<'EOF'
it('dry run prompt received', () => {
  expect(true).toBe(true);
});
EOF
    exit 0
fi

exec ollama run "$model" < "$tmp_prompt"
