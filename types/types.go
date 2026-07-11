package types

// TestResult 单次测试执行结果
type TestResult struct {
	Status          string          `json:"status"`
	Framework       string          `json:"framework,omitempty"`
	Total           int             `json:"total,omitempty"`
	Passed          int             `json:"passed"`
	Failed          int             `json:"failed"`
	Skipped         int             `json:"skipped"`
	CoveragePercent float64         `json:"coverage_percent"`
	Failures        []TestFailure   `json:"failures"`
	FixSuggestions  []FixSuggestion `json:"fix_suggestions,omitempty"`
	RawOutput       string          `json:"raw_output"`
}

// TestFailure 单个测试失败详情
type TestFailure struct {
	TestName string `json:"test_name"`
	File     string `json:"file,omitempty"`
	Line     int    `json:"line,omitempty"`
	Column   int    `json:"column,omitempty"`
	Error    string `json:"error"`
	Expected string `json:"expected,omitempty"`
	Received string `json:"received,omitempty"`
}

// FixSuggestion 修复建议
type FixSuggestion struct {
	File         string      `json:"file"`
	Line         int         `json:"line"`
	Issue        string      `json:"issue"`
	Category     string      `json:"category,omitempty"`
	ContextFile  string      `json:"context_file,omitempty"`
	ContextLine  int         `json:"context_line,omitempty"`
	SuggestedFix string      `json:"suggested_fix"`
	Confidence   float64     `json:"confidence"`
	RepairTask   *RepairTask `json:"repair_task,omitempty"`
}

// RepairTask 面向 Agent 的可执行修复任务
type RepairTask struct {
	ID                string   `json:"id"`
	TestName          string   `json:"test_name,omitempty"`
	Category          string   `json:"category"`
	Issue             string   `json:"issue"`
	TargetFile        string   `json:"target_file"`
	TargetLine        int      `json:"target_line,omitempty"`
	ContextFile       string   `json:"context_file,omitempty"`
	ContextLine       int      `json:"context_line,omitempty"`
	ContextSnippet    string   `json:"context_snippet,omitempty"`
	EditableFiles     []string `json:"editable_files"`
	SuggestedCommands []string `json:"suggested_commands,omitempty"`
	AssertionFocus    string   `json:"assertion_focus,omitempty"`
}

// GenerateTestsInput generate_tests 工具输入
type GenerateTestsInput struct {
	FilePath       string            `json:"file_path"`
	Framework      string            `json:"framework,omitempty"`
	CoverageTarget []string          `json:"coverage_target,omitempty"`
	CoverageTask   *CoverageTestTask `json:"coverage_task,omitempty"`
}

// GenerateTestsOutput generate_tests 工具输出
type GenerateTestsOutput struct {
	Status         string                 `json:"status"`
	TestFile       string                 `json:"test_file"`
	GeneratedCases int                    `json:"generated_cases"`
	Preview        string                 `json:"preview,omitempty"`
	Context        *TestGenerationContext `json:"context,omitempty"`
	CoverageTask   *CoverageTestTask      `json:"coverage_task,omitempty"`
	Provider       string                 `json:"provider,omitempty"`
	Error          string                 `json:"error,omitempty"`
	ProviderError  *ProviderErrorOutput   `json:"provider_error,omitempty"`
}

// CoverageTaskValidationOutput describes the generate -> run validation loop
// for a single coverage-driven test task.
type CoverageTaskValidationOutput struct {
	Status        string               `json:"status"`
	Action        string               `json:"action"`
	CoverageTask  *CoverageTestTask    `json:"coverage_task,omitempty"`
	Generated     *GenerateTestsOutput `json:"generated,omitempty"`
	RunResult     *TestResult          `json:"run_result,omitempty"`
	Error         string               `json:"error,omitempty"`
	ProviderError *ProviderErrorOutput `json:"provider_error,omitempty"`
	Metadata      map[string]any       `json:"metadata,omitempty"`
}

// ProviderErrorOutput describes an external test provider failure in a stable
// shape that agents can consume without parsing localized error text.
type ProviderErrorOutput struct {
	Kind     string `json:"kind"`
	Action   string `json:"action"`
	Provider string `json:"provider,omitempty"`
	Message  string `json:"message,omitempty"`
}

// TestGenerationContext describes source structure for semantic test generation.
type TestGenerationContext struct {
	Language     string            `json:"language"`
	Framework    string            `json:"framework"`
	SourceFile   string            `json:"source_file"`
	Imports      []string          `json:"imports,omitempty"`
	Types        []string          `json:"types,omitempty"`
	Targets      []TestTarget      `json:"targets"`
	CoverageTask *CoverageTestTask `json:"coverage_task,omitempty"`
}

// TestTarget is a function or method that can be tested.
type TestTarget struct {
	Name              string   `json:"name"`
	Kind              string   `json:"kind"`
	ClassName         string   `json:"class_name,omitempty"`
	Params            []string `json:"params,omitempty"`
	Async             bool     `json:"async,omitempty"`
	ReturnType        string   `json:"return_type,omitempty"`
	ReturnTypeExpr    string   `json:"return_type_expr,omitempty"`
	ReturnExpressions []string `json:"return_expressions,omitempty"`
	PayloadNotes      []string `json:"payload_notes,omitempty"`
	HasErrorPath      bool     `json:"has_error_path,omitempty"`
	BoundaryCases     []string `json:"boundary_cases,omitempty"`
}

// CoverageReport 覆盖率报告
type CoverageReport struct {
	Framework    string               `json:"framework"`
	TotalPercent float64              `json:"total_percent"`
	Files        []CoverageFile       `json:"files"`
	Summary      CoverageSummary      `json:"summary"`
	Suggestions  []CoverageSuggestion `json:"suggestions,omitempty"`
	TestTasks    []CoverageTestTask   `json:"test_tasks,omitempty"`
}

// CoverageFile 单文件覆盖率
type CoverageFile struct {
	Path    string          `json:"path"`
	Percent float64         `json:"percent"`
	Blocks  []CoverageBlock `json:"blocks,omitempty"`
}

// CoverageBlock 覆盖率块
type CoverageBlock struct {
	StartLine int  `json:"start_line"`
	EndLine   int  `json:"end_line"`
	Count     int  `json:"count"`
	Covered   bool `json:"covered"`
}

// CoverageSummary 覆盖率汇总
type CoverageSummary struct {
	TotalStatements   int      `json:"total_statements"`
	CoveredStatements int      `json:"covered_statements"`
	TotalFiles        int      `json:"total_files"`
	CoveredFiles      int      `json:"covered_files"`
	UncoveredFiles    []string `json:"uncovered_files,omitempty"`
}

// CoverageSuggestion 覆盖率改进建议
type CoverageSuggestion struct {
	File            string   `json:"file"`
	LineRange       string   `json:"line_range"`
	Function        string   `json:"function,omitempty"`
	Kind            string   `json:"kind,omitempty"`
	GapType         string   `json:"gap_type,omitempty"`
	MissingBranches []string `json:"missing_branches,omitempty"`
	UncoveredLines  []int    `json:"uncovered_lines,omitempty"`
	SuggestedInputs []string `json:"suggested_inputs,omitempty"`
	Reason          string   `json:"reason"`
	Confidence      float64  `json:"confidence"`
}

// CoverageTestTask 覆盖率驱动测试任务
type CoverageTestTask struct {
	ID              string   `json:"id"`
	Framework       string   `json:"framework"`
	File            string   `json:"file"`
	Target          string   `json:"target"`
	Kind            string   `json:"kind,omitempty"`
	LineRange       string   `json:"line_range"`
	GapType         string   `json:"gap_type,omitempty"`
	MissingBranches []string `json:"missing_branches,omitempty"`
	UncoveredLines  []int    `json:"uncovered_lines,omitempty"`
	SuggestedInputs []string `json:"suggested_inputs,omitempty"`
	Goal            string   `json:"goal"`
	Command         string   `json:"command,omitempty"`
	TestFile        string   `json:"test_file,omitempty"`
	TestName        string   `json:"test_name,omitempty"`
	AssertionFocus  []string `json:"assertion_focus,omitempty"`
	Priority        int      `json:"priority,omitempty"`
	PriorityReason  string   `json:"priority_reason,omitempty"`
	Confidence      float64  `json:"confidence"`
}
