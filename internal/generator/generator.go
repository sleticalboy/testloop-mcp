package generator

import (
	"fmt"
	"path/filepath"
	"strings"
)

// GenerateTests 根据源文件扩展名分发到对应的生成器
func GenerateTests(srcPath string) (string, error) {
	ext := strings.ToLower(filepath.Ext(srcPath))
	switch ext {
	case ".go":
		return GenerateGoTests(srcPath)
	case ".js", ".jsx", ".ts", ".tsx", ".mjs", ".cjs":
		return GenerateJestTests(srcPath)
	case ".py":
		return GeneratePytestTests(srcPath)
	default:
		return "", fmt.Errorf("不支持的文件类型: %s（支持: .go, .js, .ts, .jsx, .tsx, .py）", ext)
	}
}
