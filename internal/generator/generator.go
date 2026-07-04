package generator

import (
	"fmt"
	"os"
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
	case ".rs":
		source, err := os.ReadFile(srcPath)
		if err != nil {
			return "", fmt.Errorf("读取 Rust 源文件失败: %w", err)
		}
		_, content, err := GenerateRustTests(source, srcPath)
		return content, err
	case ".java":
		source, err := os.ReadFile(srcPath)
		if err != nil {
			return "", fmt.Errorf("读取 Java 源文件失败: %w", err)
		}
		_, content, err := GenerateJavaTests(source, srcPath)
		return content, err
	default:
		return "", fmt.Errorf("不支持的文件类型: %s（支持: .go, .js, .ts, .jsx, .tsx, .py, .rs, .java）", ext)
	}
}
