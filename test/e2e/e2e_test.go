package integration

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sleticalboy/testloop-mcp/tools"
)

// startServer 启动 MCP server 并返回一个已连接的 ClientSession
func startServer(t *testing.T) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()

	server := mcp.NewServer(
		&mcp.Implementation{Name: "testloop-mcp-test", Version: "0.0.0"},
		nil,
	)
	tools.Register(server)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	// server 必须先连接
	go func() {
		if err := server.Run(ctx, serverTransport); err != nil {
			t.Logf("server ended: %v", err)
		}
	}()

	client := mcp.NewClient(
		&mcp.Implementation{Name: "test-client", Version: "1.0.0"},
		nil,
	)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client connect failed: %v", err)
	}
	return session
}

func startStdioCommandServer(t *testing.T) *mcp.ClientSession {
	t.Helper()
	binary := buildTestloopBinary(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	t.Cleanup(cancel)
	cmd := exec.CommandContext(ctx, binary, "--transport=stdio")
	transport := &mcp.CommandTransport{
		Command:           cmd,
		TerminateDuration: 2 * time.Second,
	}
	client := mcp.NewClient(
		&mcp.Implementation{Name: "test-stdio-client", Version: "1.0.0"},
		nil,
	)
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("stdio client connect failed: %v", err)
	}
	t.Cleanup(func() {
		_ = session.Close()
	})
	return session
}

func startStreamableHTTPCommandServer(t *testing.T) *mcp.ClientSession {
	t.Helper()
	binary := buildTestloopBinary(t)
	addr := freeTCPAddr(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)

	cmd := exec.CommandContext(ctx, binary, "--transport=http", "--addr="+addr)
	if err := cmd.Start(); err != nil {
		t.Fatalf("start http testloop-mcp: %v", err)
	}
	t.Cleanup(func() {
		cancel()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	})

	baseURL := "http://" + addr
	waitForHealthz(t, ctx, baseURL+"/healthz")

	client := mcp.NewClient(
		&mcp.Implementation{Name: "test-http-client", Version: "1.0.0"},
		nil,
	)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint:             baseURL + "/mcp",
		DisableStandaloneSSE: true,
	}, nil)
	if err != nil {
		t.Fatalf("streamable http client connect failed: %v", err)
	}
	t.Cleanup(func() {
		_ = session.Close()
	})
	return session
}

func buildTestloopBinary(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "testloop-mcp")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-o", binary, projectRoot())
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build testloop-mcp binary: %v\n%s", err, output)
	}
	return binary
}

func freeTCPAddr(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen free tcp addr: %v", err)
	}
	defer listener.Close()
	return listener.Addr().String()
}

func waitForHealthz(t *testing.T, ctx context.Context, url string) {
	t.Helper()
	client := http.Client{Timeout: 250 * time.Millisecond}
	deadline := time.Now().Add(5 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			t.Fatalf("new healthz request: %v", err)
		}
		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
			lastErr = nil
		} else {
			lastErr = err
		}
		time.Sleep(100 * time.Millisecond)
	}
	if lastErr != nil {
		t.Fatalf("healthz did not become ready: %v", lastErr)
	}
	t.Fatalf("healthz did not return 200 before timeout")
}

// callTool 调用工具并返回解析后的 JSON payload
func callTool(t *testing.T, session *mcp.ClientSession, name string, args map[string]any) map[string]any {
	t.Helper()
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	if err != nil {
		t.Fatalf("CallTool(%s) error: %v", name, err)
	}
	if len(result.Content) == 0 {
		t.Fatalf("CallTool(%s) returned no content", name)
	}
	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("CallTool(%s) content[0] is not TextContent: %T", name, result.Content[0])
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(textContent.Text), &payload); err != nil {
		t.Fatalf("CallTool(%s) failed to parse JSON: %v\nraw: %s", name, err, textContent.Text)
	}
	var structured map[string]any
	if err := json.Unmarshal(toolStructuredContentJSON(t, name, result), &structured); err != nil {
		t.Fatalf("CallTool(%s) failed to parse structuredContent JSON: %v\nraw: %v", name, err, result.StructuredContent)
	}
	if !reflect.DeepEqual(structured, payload) {
		t.Fatalf("CallTool(%s) structuredContent mismatch\nstructured: %#v\ntext: %#v", name, structured, payload)
	}
	return payload
}

// callToolRaw 调用工具并返回原始 JSON（用于返回数组而非对象的工具）
func callToolRaw(t *testing.T, session *mcp.ClientSession, name string, args map[string]any) []any {
	t.Helper()
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	if err != nil {
		t.Fatalf("CallTool(%s) error: %v", name, err)
	}
	if len(result.Content) == 0 {
		t.Fatalf("CallTool(%s) returned no content", name)
	}
	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("CallTool(%s) content[0] is not TextContent: %T", name, result.Content[0])
	}
	var arr []any
	if err := json.Unmarshal([]byte(textContent.Text), &arr); err != nil {
		// 可能是对象而非数组
		var obj map[string]any
		if err2 := json.Unmarshal([]byte(textContent.Text), &obj); err2 != nil {
			t.Fatalf("CallTool(%s) failed to parse JSON: %v\nraw: %s", name, err, textContent.Text)
		}
		t.Fatalf("CallTool(%s) returned object, not array: %v", name, obj)
	}
	var structured []any
	if err := json.Unmarshal(toolStructuredContentJSON(t, name, result), &structured); err != nil {
		t.Fatalf("CallTool(%s) failed to parse structuredContent JSON: %v\nraw: %v", name, err, result.StructuredContent)
	}
	if !reflect.DeepEqual(structured, arr) {
		t.Fatalf("CallTool(%s) structuredContent mismatch\nstructured: %#v\ntext: %#v", name, structured, arr)
	}
	return arr
}

func toolStructuredContentJSON(t *testing.T, name string, result *mcp.CallToolResult) []byte {
	t.Helper()
	if result.StructuredContent == nil {
		t.Fatalf("CallTool(%s) returned nil structuredContent", name)
	}
	if raw, ok := result.StructuredContent.(json.RawMessage); ok {
		return raw
	}
	data, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("CallTool(%s) failed to marshal structuredContent %T: %v", name, result.StructuredContent, err)
	}
	return data
}

// projectRoot 返回项目根目录（从 test/e2e/ 向上两级）
func projectRoot() string {
	abs, _ := filepath.Abs(filepath.Join("..", ".."))
	return abs
}

func prependPath(t *testing.T, dir string) {
	t.Helper()
	separator := string(os.PathListSeparator)
	oldPath := os.Getenv("PATH")
	if oldPath == "" {
		t.Setenv("PATH", dir)
		return
	}
	t.Setenv("PATH", dir+separator+oldPath)
}

func writeExecutable(t *testing.T, path string, content string) {
	t.Helper()
	mode := os.FileMode(0o755)
	if runtime.GOOS == "windows" {
		mode = 0o644
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatalf("write executable %s: %v", path, err)
	}
}

// TestE2E_ListTools 验证 tools/list 返回全部 MCP 工具
func TestE2E_ListTools(t *testing.T) {
	session := startServer(t)
	defer session.Close()

	result, err := session.ListTools(context.Background(), &mcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("ListTools error: %v", err)
	}

	expectedTools := map[string]bool{
		"generate_tests":         false,
		"run_tests":              false,
		"parse_results":          false,
		"fix_suggestions":        false,
		"parse_coverage":         false,
		"validate_coverage_task": false,
	}

	for _, tool := range result.Tools {
		if _, ok := expectedTools[tool.Name]; ok {
			expectedTools[tool.Name] = true
		}
	}

	for name, found := range expectedTools {
		if !found {
			t.Errorf("tool %q not found in tools/list", name)
		}
	}

	t.Logf("ListTools: %d tools registered", len(result.Tools))
}

func TestE2E_StdioCommandTransportListsAndCallsTool(t *testing.T) {
	session := startStdioCommandServer(t)

	result, err := session.ListTools(context.Background(), &mcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("stdio ListTools error: %v", err)
	}
	foundParseResults := false
	for _, tool := range result.Tools {
		if tool.Name == "parse_results" {
			foundParseResults = true
			break
		}
	}
	if !foundParseResults {
		t.Fatalf("stdio ListTools missing parse_results: %+v", result.Tools)
	}

	payload := callTool(t, session, "parse_results", map[string]any{
		"framework": "go-test",
		"output": strings.Join([]string{
			`{"Action":"run","Package":"example.com/calc","Test":"TestAdd"}`,
			`{"Action":"pass","Package":"example.com/calc","Test":"TestAdd","Elapsed":0}`,
			`{"Action":"pass","Package":"example.com/calc","Elapsed":0}`,
		}, "\n"),
	})
	if payload["status"] != "pass" || payload["framework"] != "go-test" {
		t.Fatalf("unexpected stdio parse_results payload: %+v", payload)
	}
}

func TestE2E_StreamableHTTPTransportListsAndCallsTool(t *testing.T) {
	session := startStreamableHTTPCommandServer(t)

	result, err := session.ListTools(context.Background(), &mcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("streamable http ListTools error: %v", err)
	}
	foundParseResults := false
	for _, tool := range result.Tools {
		if tool.Name == "parse_results" {
			foundParseResults = true
			break
		}
	}
	if !foundParseResults {
		t.Fatalf("streamable http ListTools missing parse_results: %+v", result.Tools)
	}

	payload := callTool(t, session, "parse_results", map[string]any{
		"framework": "go-test",
		"output": strings.Join([]string{
			`{"Action":"run","Package":"example.com/calc","Test":"TestHTTP"}`,
			`{"Action":"pass","Package":"example.com/calc","Test":"TestHTTP","Elapsed":0}`,
			`{"Action":"pass","Package":"example.com/calc","Elapsed":0}`,
		}, "\n"),
	})
	if payload["status"] != "pass" || payload["framework"] != "go-test" {
		t.Fatalf("unexpected streamable http parse_results payload: %+v", payload)
	}
}

// TestE2E_GenerateTests 验证 generate_tests 端到端
func TestE2E_GenerateTests(t *testing.T) {
	session := startServer(t)
	defer session.Close()

	srcPath := filepath.Join(projectRoot(), "demo", "calc.go")

	payload := callTool(t, session, "generate_tests", map[string]any{
		"file_path": srcPath,
	})

	if payload["status"] != "ok" {
		t.Fatalf("expected status=ok, got: %v", payload["status"])
	}
	if payload["provider"] != "static" {
		t.Fatalf("expected provider=static, got: %v", payload["provider"])
	}

	testFile, _ := payload["test_file"].(string)
	if testFile == "" {
		t.Fatal("test_file is empty")
	}
	defer os.Remove(testFile)

	preview, _ := payload["preview"].(string)
	if !strings.Contains(preview, "func Test") {
		t.Errorf("preview should contain 'func Test'")
	}

	generatedCases, _ := payload["generated_cases"].(float64)
	if generatedCases <= 0 {
		t.Fatalf("expected generated_cases > 0, got: %v", payload["generated_cases"])
	}

	t.Logf("generate_tests: test_file=%s, preview_len=%d", testFile, len(preview))
}

// TestE2E_GenerateTests_Context 验证 generate_tests 返回语义生成上下文
func TestE2E_GenerateTests_Context(t *testing.T) {
	session := startServer(t)
	defer session.Close()

	srcPath := filepath.Join(projectRoot(), "demo", "calc.py")
	payload := callTool(t, session, "generate_tests", map[string]any{
		"file_path": srcPath,
	})

	testFile, _ := payload["test_file"].(string)
	if testFile == "" {
		t.Fatal("test_file is empty")
	}
	defer os.Remove(testFile)

	contextPayload, ok := payload["context"].(map[string]any)
	if !ok {
		t.Fatalf("expected context object, got: %T", payload["context"])
	}
	if contextPayload["language"] != "python" || contextPayload["framework"] != "pytest" {
		t.Fatalf("unexpected context metadata: %+v", contextPayload)
	}
	targets, ok := contextPayload["targets"].([]any)
	if !ok || len(targets) == 0 {
		t.Fatalf("expected non-empty context targets, got: %v", contextPayload["targets"])
	}
}

func TestE2E_GenerateTests_WithCoverageTask(t *testing.T) {
	session := startServer(t)
	defer session.Close()

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "calc.py")
	if err := os.WriteFile(srcPath, []byte("def add(a, b):\n    return a + b\n"), 0644); err != nil {
		t.Fatal(err)
	}
	testPath := filepath.Join(dir, "tests", "test_calc.py")
	payload := callTool(t, session, "generate_tests", map[string]any{
		"file_path": srcPath,
		"coverage_task": map[string]any{
			"id":              "pytest-1",
			"framework":       "pytest",
			"file":            srcPath,
			"target":          "add",
			"line_range":      "2-2",
			"gap_type":        "return_path",
			"goal":            "为 add 补充测试，覆盖未执行行段 2-2",
			"test_file":       testPath,
			"test_name":       "test_add_covers_gap",
			"assertion_focus": []string{"断言未覆盖返回路径的具体结果"},
			"priority":        100,
			"confidence":      0.9,
		},
	})

	if payload["status"] != "ok" {
		t.Fatalf("expected status=ok, got: %v", payload["status"])
	}
	if payload["test_file"] != testPath {
		t.Fatalf("expected task test file %q, got %v", testPath, payload["test_file"])
	}
	if _, err := os.Stat(testPath); err != nil {
		t.Fatalf("expected generated test file at task path: %v", err)
	}
	contextPayload, ok := payload["context"].(map[string]any)
	if !ok {
		t.Fatalf("expected context object, got: %T", payload["context"])
	}
	if _, ok := contextPayload["coverage_task"].(map[string]any); !ok {
		t.Fatalf("expected coverage_task in context, got %+v", contextPayload)
	}
	if _, ok := payload["coverage_task"].(map[string]any); !ok {
		t.Fatalf("expected coverage_task in output, got %+v", payload)
	}
}

// TestE2E_RunTests 验证 run_tests 端到端
func TestE2E_RunTests(t *testing.T) {
	session := startServer(t)
	defer session.Close()

	payload := callTool(t, session, "run_tests", map[string]any{
		"path":      filepath.Join(projectRoot(), "demo", "calc.go"),
		"framework": "go-test",
	})

	// run_tests 返回 TestResult，status 可能是 "pass"/"fail"
	status, _ := payload["status"].(string)
	if status != "pass" && status != "fail" {
		t.Fatalf("expected status=pass or fail, got: %v", status)
	}

	framework, _ := payload["framework"].(string)
	if framework != "go-test" {
		t.Errorf("expected framework=go-test, got: %s", framework)
	}

	t.Logf("run_tests: status=%s, passed=%v, failed=%v",
		status, payload["passed"], payload["failed"])
}

// TestE2E_ParseResults 验证 parse_results 端到端
func TestE2E_ParseResults(t *testing.T) {
	session := startServer(t)
	defer session.Close()

	// 模拟 Go test 失败输出
	fakeOutput := `=== RUN   TestAdd
--- FAIL: TestAdd (0.00s)
    calc_test.go:10: got 4, want 5
=== RUN   TestDivide
--- FAIL: TestDivide (0.00s)
    calc_test.go:20: division by zero
FAIL
exit status 1`

	payload := callTool(t, session, "parse_results", map[string]any{
		"output":    fakeOutput,
		"framework": "go-test",
	})

	framework, _ := payload["framework"].(string)
	if framework != "go-test" {
		t.Errorf("expected framework=go-test, got: %s", framework)
	}

	// 应该有 failures 数组
	failures, ok := payload["failures"].([]any)
	if !ok {
		t.Fatalf("expected failures array, got: %T", payload["failures"])
	}
	if len(failures) != 2 {
		t.Errorf("expected 2 failures, got %d", len(failures))
	}

	t.Logf("parse_results: framework=%s, failures=%d", framework, len(failures))
}

// TestE2E_FixSuggestions 验证 fix_suggestions 端到端
func TestE2E_FixSuggestions(t *testing.T) {
	session := startServer(t)
	defer session.Close()

	failuresJSON := `[{"test":"TestAdd","file":"calc_test.go","line":10,"message":"got 4, want 5","error":"got 4, want 5"}]`
	srcPath := filepath.Join(projectRoot(), "demo", "calc.go")

	// fix_suggestions 返回的是数组而非对象
	suggestions := callToolRaw(t, session, "fix_suggestions", map[string]any{
		"failures":    failuresJSON,
		"source_code": srcPath,
	})

	if len(suggestions) == 0 {
		t.Fatal("expected non-empty suggestions")
	}

	first, ok := suggestions[0].(map[string]any)
	if !ok {
		t.Fatalf("suggestion[0] is not object: %T", suggestions[0])
	}

	if first["suggested_fix"] == nil {
		t.Error("suggested_fix is missing")
	}
	if first["category"] != "expectation_mismatch" {
		t.Errorf("category = %v, want expectation_mismatch", first["category"])
	}

	t.Logf("fix_suggestions: %d suggestions, first confidence=%v", len(suggestions), first["confidence"])
}

// TestE2E_ParseCoverage 验证 parse_coverage 端到端
func TestE2E_ParseCoverage(t *testing.T) {
	session := startServer(t)
	defer session.Close()

	// 模拟 Go coverprofile 数据
	coverData := `mode: set
github.com/sleticalboy/testloop-mcp/demo/calc.go:1.1,8.1 1 1
github.com/sleticalboy/testloop-mcp/demo/calc.go:11.1,13.1 1 0
github.com/sleticalboy/testloop-mcp/demo/calc.go:15.1,21.2 1 1`

	payload := callTool(t, session, "parse_coverage", map[string]any{
		"data":      coverData,
		"framework": "go-test",
	})

	framework, _ := payload["framework"].(string)
	if framework != "go-test" {
		t.Errorf("expected framework=go-test, got: %s", framework)
	}

	// 应该有 total_percent 字段
	if payload["total_percent"] == nil {
		t.Fatal("total_percent is missing")
	}

	// 应该有 files 数组
	files, ok := payload["files"].([]any)
	if !ok {
		t.Fatalf("expected files array, got: %T", payload["files"])
	}
	if len(files) == 0 {
		t.Error("expected non-empty files")
	}

	t.Logf("parse_coverage: framework=%s, total_percent=%v, files=%d",
		framework, payload["total_percent"], len(files))
}

func TestE2E_RunTests_RustCoverageLoop(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake shell commands are unix-style")
	}

	tmp := t.TempDir()
	projectDir := filepath.Join(tmp, "rust-demo")
	if err := os.MkdirAll(filepath.Join(projectDir, "src"), 0o755); err != nil {
		t.Fatalf("mkdir rust project: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "Cargo.toml"), []byte("[package]\nname = \"rust-demo\"\nversion = \"0.1.0\"\nedition = \"2021\"\n"), 0o644); err != nil {
		t.Fatalf("write Cargo.toml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "src", "lib.rs"), []byte("pub fn add(a: i32, b: i32) -> i32 { a + b }\n"), 0o644); err != nil {
		t.Fatalf("write lib.rs: %v", err)
	}

	binDir := filepath.Join(tmp, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin dir: %v", err)
	}
	writeExecutable(t, filepath.Join(binDir, "cargo"), `#!/bin/sh
if [ "$1" = "tarpaulin" ]; then
  outdir=""
  while [ "$#" -gt 0 ]; do
    if [ "$1" = "--output-dir" ]; then
      shift
      outdir="$1"
    fi
    shift
  done
  mkdir -p "$outdir"
  cat > "$outdir/lcov.info" <<'EOF'
TN:
SF:src/lib.rs
DA:1,1
DA:2,0
end_of_record
EOF
  exit 0
fi
cat <<'EOF'
running 1 test
test tests::adds ... ok

test result: ok. 1 passed; 0 failed; 0 ignored; 0 measured; 0 filtered out
EOF
`)
	prependPath(t, binDir)

	session := startServer(t)
	defer session.Close()

	payload := callTool(t, session, "run_tests", map[string]any{
		"path":      projectDir,
		"framework": "cargo-test",
		"coverage":  true,
	})
	if payload["framework"] != "cargo-test" {
		t.Fatalf("framework = %v, want cargo-test", payload["framework"])
	}
	if payload["status"] != "pass" {
		t.Fatalf("status = %v, want pass; raw=%v", payload["status"], payload["raw_output"])
	}
	if payload["coverage_percent"] != float64(50) {
		t.Fatalf("coverage_percent = %v, want 50", payload["coverage_percent"])
	}

	coveragePayload := callTool(t, session, "parse_coverage", map[string]any{
		"data":      filepath.Join(projectDir, "target", "tarpaulin", "lcov.info"),
		"framework": "cargo-test",
	})
	if coveragePayload["total_percent"] != float64(50) {
		t.Fatalf("total_percent = %v, want 50", coveragePayload["total_percent"])
	}
	if coveragePayload["framework"] != "cargo-test" {
		t.Fatalf("coverage framework = %v, want cargo-test", coveragePayload["framework"])
	}
}

func TestE2E_RunTests_JavaCoverageLoop(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake shell commands are unix-style")
	}

	tmp := t.TempDir()
	projectDir := filepath.Join(tmp, "java-demo")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir java project: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "pom.xml"), []byte("<project/>"), 0o644); err != nil {
		t.Fatalf("write pom.xml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "Calculator.java"), []byte("class Calculator { int add(int a, int b) { return a + b; } }\n"), 0o644); err != nil {
		t.Fatalf("write Calculator.java: %v", err)
	}

	binDir := filepath.Join(tmp, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin dir: %v", err)
	}
	writeExecutable(t, filepath.Join(binDir, "mvn"), `#!/bin/sh
mkdir -p target/site/jacoco
cat > target/site/jacoco/jacoco.xml <<'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<report name="demo">
  <package name="com/example">
    <sourcefile name="Calculator.java">
      <line nr="1" mi="0" ci="1"/>
      <line nr="2" mi="1" ci="0"/>
      <counter type="LINE" missed="1" covered="1"/>
    </sourcefile>
  </package>
  <counter type="LINE" missed="1" covered="1"/>
</report>
EOF
cat <<'EOF'
Tests run: 2, Failures: 0, Errors: 0, Skipped: 0
EOF
`)
	prependPath(t, binDir)

	session := startServer(t)
	defer session.Close()

	payload := callTool(t, session, "run_tests", map[string]any{
		"path":      projectDir,
		"framework": "junit",
		"coverage":  true,
	})
	if payload["framework"] != "junit" {
		t.Fatalf("framework = %v, want junit", payload["framework"])
	}
	if payload["status"] != "pass" {
		t.Fatalf("status = %v, want pass; raw=%v", payload["status"], payload["raw_output"])
	}
	if payload["coverage_percent"] != float64(50) {
		t.Fatalf("coverage_percent = %v, want 50", payload["coverage_percent"])
	}

	coveragePayload := callTool(t, session, "parse_coverage", map[string]any{
		"data":      filepath.Join(projectDir, "target", "site", "jacoco", "jacoco.xml"),
		"framework": "junit",
	})
	if coveragePayload["total_percent"] != float64(50) {
		t.Fatalf("total_percent = %v, want 50", coveragePayload["total_percent"])
	}
	if coveragePayload["framework"] != "junit" {
		t.Fatalf("coverage framework = %v, want junit", coveragePayload["framework"])
	}
}

// TestE2E_FullLoop 验证完整闭环：generate → run → parse → fix → coverage
func TestE2E_FullLoop(t *testing.T) {
	session := startServer(t)
	defer session.Close()

	ctx := context.Background()
	srcPath := filepath.Join(projectRoot(), "demo", "calc.go")

	// Step 1: generate_tests
	t.Log("Step 1: generate_tests")
	genResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "generate_tests",
		Arguments: map[string]any{
			"file_path": srcPath,
		},
	})
	if err != nil {
		t.Fatalf("generate_tests failed: %v", err)
	}
	genText := genResult.Content[0].(*mcp.TextContent).Text
	var genPayload map[string]any
	json.Unmarshal([]byte(genText), &genPayload)
	testFile := genPayload["test_file"].(string)
	defer os.Remove(testFile)
	t.Logf("  → generated: %s", testFile)

	// Step 2: run_tests（对源码包跑测试，不用生成的文件）
	t.Log("Step 2: run_tests")
	runResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "run_tests",
		Arguments: map[string]any{
			"path":      filepath.Join(projectRoot(), "demo"),
			"framework": "go-test",
		},
	})
	if err != nil {
		t.Fatalf("run_tests failed: %v", err)
	}
	runText := runResult.Content[0].(*mcp.TextContent).Text
	var runPayload map[string]any
	json.Unmarshal([]byte(runText), &runPayload)
	t.Logf("  → status=%s, total=%v", runPayload["status"], runPayload["total"])

	// Step 3: parse_results（用 run_tests 的原始输出解析）
	t.Log("Step 3: parse_results")
	parseResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "parse_results",
		Arguments: map[string]any{
			"output":    "FAIL\n--- FAIL: TestX (0.00s)\n    x_test.go:5: got 1, want 2\n",
			"framework": "go-test",
		},
	})
	if err != nil {
		t.Fatalf("parse_results failed: %v", err)
	}
	parseText := parseResult.Content[0].(*mcp.TextContent).Text
	var parsePayload map[string]any
	json.Unmarshal([]byte(parseText), &parsePayload)
	t.Logf("  → failures=%v", len(parsePayload["failures"].([]any)))

	// Step 4: fix_suggestions
	t.Log("Step 4: fix_suggestions")
	fixResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "fix_suggestions",
		Arguments: map[string]any{
			"failures":    `[{"test":"TestX","file":"x_test.go","line":5,"error":"got 1, want 2"}]`,
			"source_code": srcPath,
		},
	})
	if err != nil {
		t.Fatalf("fix_suggestions failed: %v", err)
	}
	fixText := fixResult.Content[0].(*mcp.TextContent).Text
	var fixArr []any
	json.Unmarshal([]byte(fixText), &fixArr)
	t.Logf("  → suggestions=%d", len(fixArr))

	// Step 5: parse_coverage
	t.Log("Step 5: parse_coverage")
	coverResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "parse_coverage",
		Arguments: map[string]any{
			"data":      "mode: set\ngithub.com/sleticalboy/testloop-mcp/demo/calc.go:1.1,8.1 1 1\n",
			"framework": "go-test",
		},
	})
	if err != nil {
		t.Fatalf("parse_coverage failed: %v", err)
	}
	coverText := coverResult.Content[0].(*mcp.TextContent).Text
	var coverPayload map[string]any
	json.Unmarshal([]byte(coverText), &coverPayload)
	t.Logf("  → total_percent=%v", coverPayload["total_percent"])

	t.Log("Full loop completed successfully!")
}

// TestE2E_GenerateTests_Rust 验证 Rust 生成测试
func TestE2E_GenerateTests_Rust(t *testing.T) {
	session := startServer(t)
	defer session.Close()

	payload := callTool(t, session, "generate_tests", map[string]any{
		"file_path": filepath.Join(projectRoot(), "demo", "calc.rs"),
	})

	// 检查是否返回了 test_file 和 preview
	testFile, _ := payload["test_file"].(string)
	if testFile == "" {
		t.Fatal("test_file is empty")
	}

	preview, _ := payload["preview"].(string)
	if !strings.Contains(preview, "#[test]") {
		t.Errorf("preview should contain '#[test]', got: %.100s", preview)
	}
	if !strings.Contains(preview, "fn test_") {
		t.Errorf("preview should contain 'fn test_', got: %.100s", preview)
	}

	generatedCases, _ := payload["generated_cases"].(float64)
	if generatedCases <= 0 {
		t.Fatalf("expected generated_cases > 0, got: %v", payload["generated_cases"])
	}

	t.Logf("generate_tests (Rust): test_file=%s, preview_len=%d", testFile, len(preview))
}

// TestE2E_GenerateTests_Java 验证 Java 生成测试
func TestE2E_GenerateTests_Java(t *testing.T) {
	session := startServer(t)
	defer session.Close()

	payload := callTool(t, session, "generate_tests", map[string]any{
		"file_path": filepath.Join(projectRoot(), "demo", "Calculator.java"),
	})

	testFile, _ := payload["test_file"].(string)
	if testFile == "" {
		t.Fatal("test_file is empty")
	}

	preview, _ := payload["preview"].(string)
	if !strings.Contains(preview, "@Test") {
		t.Errorf("preview should contain '@Test', got: %.100s", preview)
	}
	if !strings.Contains(preview, "assert") {
		t.Errorf("preview should contain 'assert', got: %.100s", preview)
	}

	generatedCases, _ := payload["generated_cases"].(float64)
	if generatedCases <= 0 {
		t.Fatalf("expected generated_cases > 0, got: %v", payload["generated_cases"])
	}

	t.Logf("generate_tests (Java): test_file=%s, preview_len=%d", testFile, len(preview))
}
