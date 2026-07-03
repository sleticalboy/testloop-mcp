package types

// TestResult 单次测试执行结果
type TestResult struct {
	Status          string        `json:"status"`
	Framework       string        `json:"framework,omitempty"`
	Total           int           `json:"total,omitempty"`
	Passed          int           `json:"passed"`
	Failed          int           `json:"failed"`
	Skipped        int           `json:"skipped"`
	CoveragePercent float64       `json:"coverage_percent"`
	Failures        []TestFailure `json:"failures"`
	RawOutput       string        `json:"raw_output"`
}

// TestFailure 单个测试失败详情
type TestFailure struct {
	TestName string `json:"test_name"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Error    string `json:"error"`
}

// FixSuggestion 修复建议
type FixSuggestion struct {
	File        string  `json:"file"`
	Line        int     `json:"line"`
	Issue       string  `json:"issue"`
	SuggestedFix string  `json:"suggested_fix"`
	Confidence  float64 `json:"confidence"`
}

// GenerateTestsInput generate_tests 工具输入
type GenerateTestsInput struct {
	FilePath       string   `json:"file_path"`
	Framework      string   `json:"framework,omitempty"`
	CoverageTarget []string `json:"coverage_target,omitempty"`
}

// GenerateTestsOutput generate_tests 工具输出
type GenerateTestsOutput struct {
	Status        string `json:"status"`
	TestFile      string `json:"test_file"`
	GeneratedCases int    `json:"generated_cases"`
	Preview       string `json:"preview,omitempty"`
}
