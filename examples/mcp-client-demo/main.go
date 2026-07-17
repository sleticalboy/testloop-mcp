package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sleticalboy/testloop-mcp/tools"
	"github.com/sleticalboy/testloop-mcp/types"
)

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "demo failed: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	session, closeSession, err := startSession(ctx)
	if err != nil {
		return err
	}
	defer closeSession()

	projectDir, err := writeDemoProject()
	if err != nil {
		return err
	}
	defer os.RemoveAll(projectDir)

	sourceFile := filepath.Join(projectDir, "calc.go")
	testFile := filepath.Join(projectDir, "calc_test.go")

	failed, err := callTool[types.TestResult](ctx, session, "run_tests", map[string]any{
		"path":                    projectDir,
		"framework":               "go-test",
		"include_fix_suggestions": true,
		"source_code":             sourceFile,
		"test_code":               testFile,
	})
	if err != nil {
		return err
	}
	fmt.Printf("1. run_tests: status=%s failed=%d suggestions=%d\n", failed.Status, failed.Failed, len(failed.FixSuggestions))

	repair := firstRepairTask(failed.FixSuggestions)
	if repair == nil {
		return fmt.Errorf("run_tests did not return a repair_task")
	}
	fmt.Printf("2. repair_task: category=%s target=%s command=%s\n",
		repair.Category,
		filepath.Base(repair.TargetFile),
		strings.Join(repair.SuggestedCommands, " && "),
	)

	if err := repairDemoAssertion(testFile); err != nil {
		return err
	}

	passed, err := callTool[types.TestResult](ctx, session, "run_tests", map[string]any{
		"path":        projectDir,
		"framework":   "go-test",
		"coverage":    true,
		"source_code": sourceFile,
		"test_code":   testFile,
	})
	if err != nil {
		return err
	}
	fmt.Printf("3. rerun: status=%s passed=%d coverage=%.1f\n", passed.Status, passed.Passed, passed.CoveragePercent)

	coverageFile := filepath.Join(projectDir, "coverage.out")
	if err := runGoCoverage(ctx, projectDir, coverageFile); err != nil {
		return err
	}

	report, err := callTool[types.CoverageReport](ctx, session, "parse_coverage", map[string]any{
		"framework": "go-test",
		"data":      coverageFile,
	})
	if err != nil {
		return err
	}
	fmt.Printf("4. parse_coverage: total=%.1f tasks=%d\n", report.TotalPercent, len(report.TestTasks))

	fmt.Println("agent_next_step=use structuredContent first; fall back to text JSON only for older clients")
	return nil
}

func startSession(ctx context.Context) (*mcp.ClientSession, func(), error) {
	server := mcp.NewServer(
		&mcp.Implementation{Name: "testloop-mcp-demo", Version: "0.0.0"},
		nil,
	)
	tools.Register(server)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- server.Run(ctx, serverTransport)
	}()

	client := mcp.NewClient(
		&mcp.Implementation{Name: "testloop-mcp-demo-client", Version: "0.0.0"},
		nil,
	)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		return nil, nil, err
	}
	closeSession := func() {
		_ = session.Close()
		<-serverDone
	}
	return session, closeSession, nil
}

func callTool[T any](ctx context.Context, session *mcp.ClientSession, name string, args map[string]any) (T, error) {
	var out T
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	if err != nil {
		return out, fmt.Errorf("call %s: %w", name, err)
	}
	if result.StructuredContent == nil {
		return decodeTextContent[T](result)
	}
	data, err := structuredContentJSON(result.StructuredContent)
	if err != nil {
		return out, fmt.Errorf("marshal %s structuredContent: %w", name, err)
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return out, fmt.Errorf("decode %s structuredContent: %w", name, err)
	}
	if err := assertTextContentMatches(result, data); err != nil {
		return out, fmt.Errorf("%s content mismatch: %w", name, err)
	}
	return out, nil
}

func structuredContentJSON(value any) ([]byte, error) {
	if raw, ok := value.(json.RawMessage); ok {
		return raw, nil
	}
	return json.Marshal(value)
}

func decodeTextContent[T any](result *mcp.CallToolResult) (T, error) {
	var out T
	if len(result.Content) == 0 {
		return out, fmt.Errorf("empty tool content")
	}
	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		return out, fmt.Errorf("content[0] is %T, want TextContent", result.Content[0])
	}
	if err := json.Unmarshal([]byte(text.Text), &out); err != nil {
		return out, err
	}
	return out, nil
}

func assertTextContentMatches(result *mcp.CallToolResult, structured []byte) error {
	if len(result.Content) == 0 {
		return fmt.Errorf("empty tool content")
	}
	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		return fmt.Errorf("content[0] is %T, want TextContent", result.Content[0])
	}
	var textValue any
	var structuredValue any
	if err := json.Unmarshal([]byte(text.Text), &textValue); err != nil {
		return err
	}
	if err := json.Unmarshal(structured, &structuredValue); err != nil {
		return err
	}
	if !reflect.DeepEqual(textValue, structuredValue) {
		return fmt.Errorf("structuredContent and text JSON differ")
	}
	return nil
}

func writeDemoProject() (string, error) {
	dir, err := os.MkdirTemp("", "testloop-mcp-client-demo-*")
	if err != nil {
		return "", err
	}
	files := map[string]string{
		"go.mod": "module example.com/testloopdemo\n\ngo 1.22\n",
		"calc.go": `package calc

func Add(a, b int) int {
	return a + b
}
`,
		"calc_test.go": `package calc

import "testing"

func TestAdd(t *testing.T) {
	if got := Add(2, 2); got != 5 {
		t.Fatalf("Add(2, 2) = %d, want 5", got)
	}
}
`,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			_ = os.RemoveAll(dir)
			return "", err
		}
	}
	return dir, nil
}

func firstRepairTask(suggestions []types.FixSuggestion) *types.RepairTask {
	for i := range suggestions {
		if suggestions[i].RepairTask != nil {
			return suggestions[i].RepairTask
		}
	}
	return nil
}

func repairDemoAssertion(testFile string) error {
	data, err := os.ReadFile(testFile)
	if err != nil {
		return err
	}
	fixed := strings.Replace(string(data), "got != 5", "got != 4", 1)
	fixed = strings.Replace(fixed, "want 5", "want 4", 1)
	if fixed == string(data) {
		return fmt.Errorf("demo assertion was not repaired")
	}
	return os.WriteFile(testFile, []byte(fixed), 0o644)
}

func runGoCoverage(ctx context.Context, projectDir, coverageFile string) error {
	cmd := exec.CommandContext(ctx, "go", "test", ".", "-coverprofile="+coverageFile)
	cmd.Dir = projectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go coverage failed: %w\n%s", err, output)
	}
	return nil
}
