#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

mkdir -p "$tmp_dir/utils"

cat > "$tmp_dir/go.mod" <<'GO'
module example.com/laoxia-smoke

go 1.23
GO

cat > "$tmp_dir/utils/alias.go" <<'GO'
package utils

import "strings"

func SliceMapper[T any, U any](src []T, mapper func(T) U) []U {
	dst := make([]U, 0, len(src))
	for _, v := range src {
		dst = append(dst, mapper(v))
	}
	return dst
}

func SplitSlice[T any](s, sep string, convert func(string) (T, bool)) []T {
	var result []T
	for _, part := range strings.Split(s, sep) {
		if val, ok := convert(part); ok {
			result = append(result, val)
		}
	}
	return result
}
GO

cat > "$tmp_dir/utils/alias_test.go" <<'GO'
package utils

import "testing"

func TestSliceMapper(t *testing.T) {
	got := SliceMapper([]int{1}, func(v int) int { return v + 1 })
	if len(got) != 1 || got[0] != 2 {
		t.Fatalf("SliceMapper() = %+v", got)
	}
}
GO

output="$tmp_dir/utils/alias_testloop_test.go"
(
  cd "$repo_root"
  go run ./cmd/testgen "$tmp_dir/utils/alias.go" "$output" >/tmp/testloop-go-cli-duplicate-name-smoke.out
)

grep -F "Generated: $output (provider=static action=manual_review)" /tmp/testloop-go-cli-duplicate-name-smoke.out >/dev/null
grep -F "func TestSliceMapperTestLoop(t *testing.T)" "$output" >/dev/null
grep -F "func TestSplitSlice(t *testing.T)" "$output" >/dev/null

(
  cd "$tmp_dir"
  go test ./utils -run 'Test(SliceMapper|SliceMapperTestLoop|SplitSlice)' -count=1
)

echo "go cli duplicate name smoke test passed"
