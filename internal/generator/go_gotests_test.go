package generator

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/sleticalboy/testloop-mcp/types"
)

func TestGenerateGoTestsPreferredUsesGotestsWhenAvailable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fake gotests is unix-only")
	}

	srcPath := writeTempGoSource(t)
	fakeBin := writeFakeGotests(t, `#!/bin/sh
if [ "$1" != "-all" ]; then
  echo "missing -all" >&2
  exit 2
fi
if [ "$2" != "calc.go" ]; then
  echo "unexpected source: $2" >&2
  exit 3
fi
cat <<'EOF'
package sample

import "testing"

func TestFromGotests(t *testing.T) {}
EOF
`)

	t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))

	code, err := GenerateGoTestsPreferred(srcPath)
	if err != nil {
		t.Fatalf("GenerateGoTestsPreferred() error = %v", err)
	}
	if !strings.Contains(code, "TestFromGotests") {
		t.Fatalf("expected gotests output, got:\n%s", code)
	}
}

func TestGenerateGoTestsPreferredFallsBackWhenGotestsFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fake gotests is unix-only")
	}

	srcPath := writeTempGoSource(t)
	fakeBin := writeFakeGotests(t, `#!/bin/sh
echo "boom" >&2
exit 42
`)

	t.Setenv("PATH", fakeBin)

	code, err := GenerateGoTestsPreferred(srcPath)
	if err != nil {
		t.Fatalf("GenerateGoTestsPreferred() fallback error = %v", err)
	}
	if strings.Contains(code, "TestFromGotests") {
		t.Fatalf("expected fallback output, got gotests output:\n%s", code)
	}
	if !strings.Contains(code, "func TestAdd") || !strings.Contains(code, "skip: false") {
		t.Fatalf("expected built-in fallback output, got:\n%s", code)
	}
	if !strings.Contains(code, "a:    1,") || !strings.Contains(code, "b:    2,") || !strings.Contains(code, "ret0: 1 + 2,") {
		t.Fatalf("expected seeded exact test case, got:\n%s", code)
	}
}

func TestGenerateGoTestsSeedsSimplePureFunction(t *testing.T) {
	srcPath := writeTempGoSource(t)

	code, err := GenerateGoTests(srcPath)
	if err != nil {
		t.Fatalf("GenerateGoTests() error = %v", err)
	}
	for _, want := range []string{
		"name: \"simple\"",
		"skip: false",
		"a:    1,",
		"b:    2,",
		"ret0: 1 + 2,",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	if strings.Contains(code, "name: \"todo\"") {
		t.Fatalf("simple pure function should not generate TODO case:\n%s", code)
	}
}

func TestGenerateGoTestsCoversCompositeSourceBranches(t *testing.T) {
	src := `package sample

type Store struct {
	Name string
	Count int
}

type Writer interface {
	Write(data []byte) (int, error)
	Close()
}

func Identity[T any](value T) T {
	return value
}

func Join(prefix string, parts ...string) string {
	return prefix
}

func (s *Store) Names(limit int) []string {
	return []string{s.Name}
}
`
	srcPath := filepath.Join(t.TempDir(), "composite.go")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}

	code, err := GenerateGoTests(srcPath)
	if err != nil {
		t.Fatalf("GenerateGoTests() error = %v", err)
	}
	for _, want := range []string{
		"\"reflect\"",
		"func makeTestStore() Store",
		"Name:  \"\"",
		"Count: 0",
		"type WriterMock struct",
		"WriteFn func([]byte) (int, error)",
		"func (m *WriterMock) Close()",
		"func TestIdentity(t *testing.T)",
		"value int",
		"Identity[int](tt.value)",
		"func TestJoin(t *testing.T)",
		"parts  []string",
		"Join(tt.prefix, tt.parts...)",
		"func TestStore_Names(t *testing.T)",
		"s := &Store{}",
		"got := s.Names(tt.limit)",
		"if !reflect.DeepEqual(got, tt.ret0)",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
}

func TestGenerateGoTestsReturnsMessageWhenNoFuncsOrInterfaces(t *testing.T) {
	srcPath := filepath.Join(t.TempDir(), "empty.go")
	if err := os.WriteFile(srcPath, []byte("package sample\n\ntype Store struct{ Name string }\n"), 0644); err != nil {
		t.Fatal(err)
	}

	code, err := GenerateGoTests(srcPath)
	if err != nil {
		t.Fatalf("GenerateGoTests() error = %v", err)
	}
	if code != "// 未发现需要生成测试的 exported 函数或接口" {
		t.Fatalf("GenerateGoTests() = %q", code)
	}
}

func TestGenerateGoTestsForCoverageTaskTargetsFunction(t *testing.T) {
	src := `package sample

func Add(a, b int) int {
	return a + b
}

func Sub(a, b int) int {
	return a - b
}
`
	srcPath := filepath.Join(t.TempDir(), "calc.go")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	task := types.CoverageTestTask{
		ID:             "go-test-1",
		Framework:      "go-test",
		Target:         "Add",
		LineRange:      "3-3",
		GapType:        "branch",
		TestName:       "TestAddCoverageGap",
		AssertionFocus: []string{"断言未覆盖分支的返回值或副作用"},
	}

	code, err := GenerateGoTestsForCoverageTask(srcPath, &task)
	if err != nil {
		t.Fatalf("GenerateGoTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"func TestAddCoverageGap",
		"coverage task: go-test-1 | lines 3-3",
		"name: \"coverage branch gap\"",
		"Add(tt.a, tt.b)",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	if strings.Contains(code, "TestSub") || strings.Contains(code, "Sub(tt.a, tt.b)") {
		t.Fatalf("task-aware generation should only target Add:\n%s", code)
	}
}

func TestGenerateGoTestsForCoverageTaskUsesBranchSuggestedInputs(t *testing.T) {
	src := `package sample

func Add(a, b int) int {
	if a == 0 {
		return b
	}
	return a + b
}
`
	srcPath := filepath.Join(t.TempDir(), "calc.go")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	task := types.CoverageTestTask{
		ID:              "go-test-branch",
		Framework:       "go-test",
		Target:          "Add",
		LineRange:       "4-4",
		GapType:         "branch",
		TestName:        "TestAddZeroBranch",
		MissingBranches: []string{"未覆盖 if 分支: a == 0"},
		SuggestedInputs: []string{"构造满足条件 `a == 0` 的输入"},
		AssertionFocus:  []string{"断言分支返回值"},
	}

	code, err := GenerateGoTestsForCoverageTask(srcPath, &task)
	if err != nil {
		t.Fatalf("GenerateGoTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"func TestAddZeroBranch(t *testing.T)",
		"\"coverage branch gap\"",
		"skip: false",
		"a:    0,",
		"b:    2,",
		"ret0: 2,",
		"got := Add(tt.a, tt.b)",
		"if got != tt.ret0",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	for _, notWant := range []string{
		"skip: true",
		"TODO: 填写有意义的输入",
	} {
		if strings.Contains(code, notWant) {
			t.Fatalf("did not expect %q in generated code:\n%s", notWant, code)
		}
	}
}

func TestGenerateGoTestsForCoverageTaskUsesURLInputForMultiReturnErrorPath(t *testing.T) {
	src := `package sample

import (
	"io"
	"net/http"
)

func GetBytes(api, tag string) ([]byte, error) {
	resp, err := http.Get(api)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
`
	srcPath := filepath.Join(t.TempDir(), "http.go")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	task := types.CoverageTestTask{
		ID:              "go-test-error-path",
		Framework:       "go-test",
		Target:          "GetBytes",
		LineRange:       "7-9",
		GapType:         "branch",
		TestName:        "TestGetBytesErrorPath",
		MissingBranches: []string{"未覆盖 if 分支: err != nil"},
		SuggestedInputs: []string{"构造满足条件 `err != nil` 的输入", "设置 api 覆盖未执行分支"},
		AssertionFocus:  []string{"断言错误路径返回 nil bytes 和非 nil error"},
	}

	code, err := GenerateGoTestsForCoverageTask(srcPath, &task)
	if err != nil {
		t.Fatalf("GenerateGoTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"func TestGetBytesErrorPath(t *testing.T)",
		`"reflect"`,
		"skip: false",
		`api:  "://invalid-url"`,
		`ret0: nil`,
		"got0, got1 := GetBytes(tt.api, tt.tag)",
		"if !reflect.DeepEqual(got0, tt.ret0)",
		"if got1 == nil",
		`expected error, got nil`,
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	for _, notWant := range []string{
		"skip: true",
		"TODO: 填写有意义的输入",
		"unexpected error",
	} {
		if strings.Contains(code, notWant) {
			t.Fatalf("did not expect %q in generated code:\n%s", notWant, code)
		}
	}
}

func TestGenerateGoTestsForCoverageTaskBuildsHTTPWrapperServerCases(t *testing.T) {
	src := `package sample

import (
	"io"
	"net/http"
)

func GetJson(api, tag string, v any) error {
	resp, err := http.Get(api)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if string(body) == "{" {
		return errJSON
	}
	return nil
}

func GetBytes(api, tag string) ([]byte, error) {
	resp, err := http.Get(api)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
`
	srcPath := filepath.Join(t.TempDir(), "http.go")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		target   string
		gapType  string
		testName string
		want     []string
		notWant  []string
	}{
		{
			name:     "get json invalid server body",
			target:   "GetJson",
			gapType:  "error_path",
			testName: "TestGetJsonInvalidBody",
			want: []string{
				`"net/http"`,
				`"net/http/httptest"`,
				"skip: false",
				"v:    &map[string]any{}",
				"srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {",
				`_, _ = w.Write([]byte("{"))`,
				"tt.api = srv.URL",
				"err := GetJson(tt.api, tt.tag, tt.v)",
				"if err == nil",
				`expected error, got nil`,
			},
			notWant: []string{"skip: true", "unexpected error"},
		},
		{
			name:     "get bytes server body",
			target:   "GetBytes",
			gapType:  "return_path",
			testName: "TestGetBytesBody",
			want: []string{
				`"net/http"`,
				`"net/http/httptest"`,
				`"reflect"`,
				"skip: false",
				`ret0: []byte("test-body")`,
				`ret1: nil`,
				`_, _ = w.Write([]byte("test-body"))`,
				"tt.api = srv.URL",
				"got0, got1 := GetBytes(tt.api, tt.tag)",
				"if !reflect.DeepEqual(got0, tt.ret0)",
				"if got1 != nil",
			},
			notWant: []string{"skip: true", "expected error, got nil"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := types.CoverageTestTask{
				ID:             "go-test-http-wrapper",
				Framework:      "go-test",
				Target:         tt.target,
				LineRange:      "1-1",
				GapType:        tt.gapType,
				TestName:       tt.testName,
				AssertionFocus: []string{"覆盖 HTTP wrapper path"},
			}
			code, err := GenerateGoTestsForCoverageTask(srcPath, &task)
			if err != nil {
				t.Fatalf("GenerateGoTestsForCoverageTask() error = %v", err)
			}
			for _, want := range append([]string{"func " + tt.testName + "(t *testing.T)"}, tt.want...) {
				if !strings.Contains(code, want) {
					t.Fatalf("expected %q in generated code:\n%s", want, code)
				}
			}
			for _, notWant := range tt.notWant {
				if strings.Contains(code, notWant) {
					t.Fatalf("did not expect %q in generated code:\n%s", notWant, code)
				}
			}
		})
	}
}

func TestGenerateGoTestsForCoverageTaskBuildsTraceTransportSlowBranch(t *testing.T) {
	src := `package sample

import (
	"net/http"
	"net/http/httptrace"
	"strings"
	"time"
)

type TraceTransport struct {
	Transport     http.RoundTripper
	SlowThreshold time.Duration
}

func (t *TraceTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	requestStart := time.Now()
	defer func() {
		totalCost := time.Since(requestStart)
		if totalCost > t.SlowThreshold {
			_ = strings.Split(req.URL.String(), "?")[0]
		}
	}()
	trace := &httptrace.ClientTrace{}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	return t.Transport.RoundTrip(req)
}
`
	srcPath := filepath.Join(t.TempDir(), "http.go")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	task := types.CoverageTestTask{
		ID:              "go-test-trace-transport",
		Framework:       "go-test",
		Target:          "TraceTransport.RoundTrip",
		LineRange:       "17-19",
		GapType:         "branch",
		TestName:        "TestTraceTransportRoundTrip",
		MissingBranches: []string{"未覆盖 if 分支: totalCost > t.SlowThreshold"},
		SuggestedInputs: []string{"构造满足条件 `totalCost > t.SlowThreshold` 的输入"},
		AssertionFocus:  []string{"断言未覆盖分支的返回值或副作用"},
	}

	code, err := GenerateGoTestsForCoverageTask(srcPath, &task)
	if err != nil {
		t.Fatalf("GenerateGoTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		`"net/http"`,
		`"net/http/httptest"`,
		`"time"`,
		"func TestTraceTransportRoundTrip(t *testing.T)",
		"skip: false",
		"req:  nil",
		"srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {",
		"time.Sleep(time.Millisecond)",
		"w.WriteHeader(http.StatusNoContent)",
		"t.Cleanup(srv.Close)",
		"receiver.Transport = http.DefaultTransport",
		"receiver.SlowThreshold = -time.Nanosecond",
		`tt.req, reqErr = http.NewRequest(http.MethodGet, srv.URL+"?token=secret", nil)`,
		"got0, got1 := receiver.RoundTrip(tt.req)",
		"if got0 == nil",
		"if got1 != nil",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	for _, notWant := range []string{"skip: true", "ret0 *http.Response", "got0 != tt.ret0"} {
		if strings.Contains(code, notWant) {
			t.Fatalf("did not expect %q in generated code:\n%s", notWant, code)
		}
	}
}

func TestGenerateGoTestsForCoverageTaskBuildsHTTPRequestBranches(t *testing.T) {
	src := `package sample

import (
	"net"
	"net/http"
	"strings"
)

var ipLookups = strings.Split("X-Forwarded-For,X-Real-IP,RemoteAddr", ",")

func RemoteIP(r *http.Request, fallback string) string {
	realIP := r.Header.Get("X-Real-IP")
	forwardedFor := r.Header.Get("X-Forwarded-For")
	for _, lookup := range ipLookups {
		if lookup == "RemoteAddr" {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				return r.RemoteAddr
			}
			return ip
		}
		if lookup == "X-Forwarded-For" && forwardedFor != "" {
			parts := strings.Split(forwardedFor, ",")
			for i, p := range parts {
				parts[i] = strings.TrimSpace(p)
			}
			partIndex := len(parts) - 1
			if partIndex < 0 {
				partIndex = 0
			}
			partIndex = 0
			return parts[partIndex]
		}
		if lookup == "X-Real-IP" && realIP != "" {
			return realIP
		}
	}
	return fallback
}
`
	srcPath := filepath.Join(t.TempDir(), "http.go")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		gapType  string
		branch   string
		want     []string
		notWant  []string
		testName string
	}{
		{
			name:     "forwarded for",
			gapType:  "branch",
			branch:   `lookup == "X-Forwarded-For" && forwardedFor != ""`,
			testName: "TestRemoteIPForwardedFor",
			want: []string{
				`r:        &http.Request{Header: http.Header{"X-Forwarded-For": []string{"198.51.100.1, 198.51.100.2"}}, RemoteAddr: "203.0.113.9:1234"}`,
				`ret0:     "198.51.100.1"`,
				"skip:     false",
			},
			notWant: []string{"skip:     true"},
		},
		{
			name:     "real ip",
			gapType:  "branch",
			branch:   `lookup == "X-Real-IP" && realIP != ""`,
			testName: "TestRemoteIPRealIP",
			want: []string{
				`r:        &http.Request{Header: http.Header{"X-Real-Ip": []string{"198.51.100.10"}}, RemoteAddr: "203.0.113.9:1234"}`,
				`ret0:     "198.51.100.10"`,
				"skip:     false",
			},
			notWant: []string{"skip:     true"},
		},
		{
			name:     "remote addr",
			gapType:  "branch",
			branch:   `lookup == "RemoteAddr"`,
			testName: "TestRemoteIPRemoteAddr",
			want: []string{
				`r:        &http.Request{Header: http.Header{}, RemoteAddr: "203.0.113.9:1234"}`,
				`ret0:     "203.0.113.9"`,
				"skip:     false",
			},
			notWant: []string{"skip:     true"},
		},
		{
			name:     "remote addr parse error",
			gapType:  "branch",
			branch:   `err != nil`,
			testName: "TestRemoteIPRemoteAddrError",
			want: []string{
				`r:        &http.Request{Header: http.Header{}, RemoteAddr: "bad-remote-addr"}`,
				`ret0:     "bad-remote-addr"`,
				"skip:     false",
			},
			notWant: []string{"skip:     true"},
		},
		{
			name:     "unreachable part index",
			gapType:  "branch",
			branch:   `partIndex < 0`,
			testName: "TestRemoteIPPartIndex",
			want: []string{
				`Static generator cannot infer exact coverage case: no simple if boundary was detected.`,
				"skip:     true",
			},
		},
		{
			name:     "fallback return path",
			gapType:  "return_path",
			testName: "TestRemoteIPFallback",
			want: []string{
				`r:        &http.Request{Header: http.Header{}, RemoteAddr: "203.0.113.9:1234"}`,
				`fallback: "fallback"`,
				`ret0:     "fallback"`,
				"oldIPLookups := ipLookups",
				`ipLookups = []string{"Unknown"}`,
				"t.Cleanup(func() { ipLookups = oldIPLookups })",
				"skip:     false",
			},
			notWant: []string{"skip:     true"},
		},
		{
			name:     "entry statement",
			gapType:  "statement",
			testName: "TestRemoteIPEntry",
			want: []string{
				`r:        &http.Request{Header: http.Header{}, RemoteAddr: "203.0.113.9:1234"}`,
				`fallback: "fallback"`,
				`ret0:     "203.0.113.9"`,
				"skip:     false",
			},
			notWant: []string{"skip:     true", "oldIPLookups := ipLookups"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := types.CoverageTestTask{
				ID:              "go-test-remote-ip",
				Framework:       "go-test",
				Target:          "RemoteIP",
				LineRange:       "10-10",
				GapType:         tt.gapType,
				TestName:        tt.testName,
				MissingBranches: []string{"未覆盖 if 分支: " + tt.branch},
				SuggestedInputs: []string{"构造满足条件 `" + tt.branch + "` 的输入"},
				AssertionFocus:  []string{"断言分支返回值"},
			}
			code, err := GenerateGoTestsForCoverageTask(srcPath, &task)
			if err != nil {
				t.Fatalf("GenerateGoTestsForCoverageTask() error = %v", err)
			}
			for _, want := range append([]string{`"net/http"`, "func " + tt.testName + "(t *testing.T)"}, tt.want...) {
				if !strings.Contains(code, want) {
					t.Fatalf("expected %q in generated code:\n%s", want, code)
				}
			}
			for _, notWant := range tt.notWant {
				if strings.Contains(code, notWant) {
					t.Fatalf("did not expect %q in generated code:\n%s", notWant, code)
				}
			}
		})
	}
}

func TestGenerateGoTestsForCoverageTaskBuildsParseTokenSuccessBranch(t *testing.T) {
	src := `package sample

import "car-svc/global"

type AuthClaims struct {
	UID  uint
	Name string
}

func GenerateToken(id uint, name string) (string, error) {
	return "", nil
}

func ParseToken(token string) (*AuthClaims, error) {
	if claims, ok := any(&AuthClaims{}).(*AuthClaims); ok && token != "" {
		return claims, nil
	}
	return nil, nil
}

func touchGlobal() {
	_ = global.Config
}
`
	srcPath := filepath.Join(t.TempDir(), "jwt.go")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	task := types.CoverageTestTask{
		ID:              "go-test-jwt",
		Framework:       "go-test",
		Target:          "ParseToken",
		LineRange:       "12-14",
		GapType:         "branch",
		TestName:        "TestParseTokenValid",
		MissingBranches: []string{"未覆盖 if 分支: ok && tc.Valid"},
		SuggestedInputs: []string{"构造满足条件 `ok && tc.Valid` 的输入"},
		AssertionFocus:  []string{"断言未覆盖分支的返回值或副作用"},
	}

	code, err := GenerateGoTestsForCoverageTask(srcPath, &task)
	if err != nil {
		t.Fatalf("GenerateGoTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		`"car-svc/global"`,
		"func TestParseTokenValid(t *testing.T)",
		"skip: false",
		`token: func() string {`,
		`global.Config.Jwt.Key = "test-secret"`,
		`global.Config.Jwt.ExpireTime = 3600`,
		`token, _ := GenerateToken(1, "admin")`,
		"got0, got1 := ParseToken(tt.token)",
		"if got0 == nil",
		`if got1 != nil`,
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	if strings.Contains(code, "skip: true") || strings.Contains(code, "ret0 *AuthClaims") {
		t.Fatalf("expected non-skipped non-nil result test without pointer expected field:\n%s", code)
	}
}

func TestGenerateGoTestsForCoverageTaskBuildsRecoverPanicBranch(t *testing.T) {
	src := `package sample

func Recover(cleanups ...func()) {
	for _, cleanup := range cleanups {
		cleanup()
	}
	if p := recover(); p != nil {
		_ = p
	}
}
`
	srcPath := filepath.Join(t.TempDir(), "recover.go")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	task := types.CoverageTestTask{
		ID:              "go-test-recover",
		Framework:       "go-test",
		Target:          "Recover",
		LineRange:       "6-8",
		GapType:         "branch",
		TestName:        "TestRecoverPanic",
		MissingBranches: []string{"未覆盖 if 分支: p != nil"},
		SuggestedInputs: []string{"构造满足条件 `p != nil` 的输入"},
		AssertionFocus:  []string{"断言 recover 分支不会继续 panic"},
	}

	code, err := GenerateGoTestsForCoverageTask(srcPath, &task)
	if err != nil {
		t.Fatalf("GenerateGoTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"func TestRecoverPanic(t *testing.T)",
		"skip:     false",
		"cleanups: []func(){func() {}}",
		"defer Recover(tt.cleanups...)",
		`panic("test panic")`,
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	if strings.Contains(code, "skip:     true") {
		t.Fatalf("expected non-skipped recover panic test:\n%s", code)
	}
}

func TestGenerateGoTestsForCoverageTaskBuildsJSONErrorBranches(t *testing.T) {
	src := `package sample

import (
	"encoding/json"
	"os"
	"reflect"
)

func AsJson(src any) string {
	if src == nil {
		return ""
	}
	data, err := json.Marshal(&src)
	if err != nil {
		if tp := reflect.TypeOf(src); tp.Kind() == reflect.Array || tp.Kind() == reflect.Slice {
			return "[]"
		}
		return "{}"
	}
	return string(data)
}

func FromJsonFile(path string, dst any) error {
	buf, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return FromJson(buf, dst)
}

func FromJson(data []byte, dst any) error {
	if err := json.Unmarshal(data, dst); err != nil {
		return err
	}
	return nil
}
`
	srcPath := filepath.Join(t.TempDir(), "json.go")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name           string
		target         string
		gapType        string
		branch         string
		testName       string
		assertionFocus []string
		want           []string
		notWant        []string
	}{
		{
			name:     "as json slice marshal error",
			target:   "AsJson",
			branch:   `tp.Kind() == reflect.Array || tp.Kind() == reflect.Slice`,
			testName: "TestAsJsonSliceError",
			want: []string{
				"skip: false",
				"src:  []func(){func() {}}",
				`ret0: "[]"`,
				"got := AsJson(tt.src)",
			},
			notWant: []string{"skip: true"},
		},
		{
			name:     "as json object marshal error",
			target:   "AsJson",
			branch:   `err != nil`,
			testName: "TestAsJsonObjectError",
			want: []string{
				"skip: false",
				"src:  func() {}",
				`ret0: "{}"`,
				"got := AsJson(tt.src)",
			},
			notWant: []string{"skip: true"},
		},
		{
			name:     "from json invalid data",
			target:   "FromJson",
			branch:   `err != nil`,
			testName: "TestFromJsonError",
			want: []string{
				"skip: false",
				`data: []byte("{")`,
				"dst:  &map[string]any{}",
				"err := FromJson(tt.data, tt.dst)",
				"if err == nil",
				`expected error, got nil`,
			},
			notWant: []string{"skip: true", "unexpected error"},
		},
		{
			name:     "from json file missing path",
			target:   "FromJsonFile",
			branch:   `err != nil`,
			testName: "TestFromJsonFileError",
			want: []string{
				"skip: false",
				`path: "testdata/does-not-exist.json"`,
				"dst:  &map[string]any{}",
				"err := FromJsonFile(tt.path, tt.dst)",
				"if err == nil",
				`expected error, got nil`,
			},
			notWant: []string{"skip: true", "unexpected error"},
		},
		{
			name:     "from json file success path",
			target:   "FromJsonFile",
			gapType:  "error_path",
			testName: "TestFromJsonFileSuccess",
			assertionFocus: []string{
				"断言未覆盖返回路径的具体结果",
				"未覆盖返回路径",
				"覆盖未执行行: 39",
			},
			want: []string{
				`"os"`,
				"skip: false",
				`path: ""`,
				"dst:  &map[string]any{}",
				`if tt.path == ""`,
				`tt.path = t.TempDir() + "/input.json"`,
				`os.WriteFile(tt.path, []byte(` + "`" + `{"ok":true}` + "`" + `), 0644)`,
				"err := FromJsonFile(tt.path, tt.dst)",
				"if err != nil",
				`unexpected error`,
			},
			notWant: []string{"skip: true", "expected error, got nil"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gapType := tt.gapType
			if gapType == "" {
				gapType = "branch"
			}
			assertionFocus := tt.assertionFocus
			if len(assertionFocus) == 0 {
				assertionFocus = []string{"断言 JSON error path"}
			}
			task := types.CoverageTestTask{
				ID:              "go-test-json",
				Framework:       "go-test",
				Target:          tt.target,
				LineRange:       "1-1",
				GapType:         gapType,
				TestName:        tt.testName,
				MissingBranches: []string{"未覆盖 if 分支: " + tt.branch},
				SuggestedInputs: []string{"构造满足条件 `" + tt.branch + "` 的输入"},
				AssertionFocus:  assertionFocus,
			}
			code, err := GenerateGoTestsForCoverageTask(srcPath, &task)
			if err != nil {
				t.Fatalf("GenerateGoTestsForCoverageTask() error = %v", err)
			}
			for _, want := range append([]string{"func " + tt.testName + "(t *testing.T)"}, tt.want...) {
				if !strings.Contains(code, want) {
					t.Fatalf("expected %q in generated code:\n%s", want, code)
				}
			}
			for _, notWant := range tt.notWant {
				if strings.Contains(code, notWant) {
					t.Fatalf("did not expect %q in generated code:\n%s", notWant, code)
				}
			}
		})
	}
}

func TestGenerateGoTestsForCoverageTaskBuildsAliasUtilityBranches(t *testing.T) {
	src := `package sample

import (
	"strings"
	"time"
)

func SliceMapper0[T any, U any](src []T, mapper func(T) U) []U {
	dst := make([]U, 0, len(src))
	filter := map[any]bool{}
	for _, v := range src {
		ret := mapper(v)
		if filter[ret] {
			continue
		}
		dst = append(dst, ret)
		filter[ret] = true
	}
	return dst
}

func UserDurationOf(tpy uint8) time.Duration {
	switch tpy {
	case 1:
		return time.Hour
	case 2:
		return time.Hour * 24
	case 3:
		return time.Hour * 24 * 30
	case 4:
		return time.Hour * 24 * 365
	case 5:
		return time.Hour * 24 * 365 * 99
	default:
		return time.Hour
	}
}

func UserTypeOf(unit time.Duration) int {
	if d := (unit.Abs() - time.Hour).Abs().Milliseconds(); d < 1000 {
		return 1
	}
	return 1
}

func Ptr[T any](v T) *T {
	return &v
}

func TrimSpaceSlice(s []string) []string {
	var result []string
	for _, v := range s {
		if v = strings.TrimSpace(v); v != "" {
			result = append(result, v)
		}
	}
	return result
}
`
	srcPath := filepath.Join(t.TempDir(), "alias.go")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		target   string
		gapType  string
		branch   string
		testName string
		want     []string
		notWant  []string
	}{
		{
			name:     "slice mapper duplicate",
			target:   "SliceMapper0",
			gapType:  "branch",
			branch:   "filter[ret]",
			testName: "TestSliceMapper0Duplicate",
			want: []string{
				"skip:   false",
				"src:    []int{1, 1, 2}",
				"mapper: func(i int) int { return i }",
				"ret0:   []int{1, 2}",
				"got := SliceMapper0[int, int](tt.src, tt.mapper)",
				"if !reflect.DeepEqual(got, tt.ret0)",
			},
			notWant: []string{"skip:   true"},
		},
		{
			name:     "user duration switch",
			target:   "UserDurationOf",
			gapType:  "branch",
			branch:   "switch/case",
			testName: "TestUserDurationOfCase",
			want: []string{
				"skip: false",
				"tpy:  5",
				"ret0: time.Hour * 24 * 365 * 99",
				"got := UserDurationOf(tt.tpy)",
			},
			notWant: []string{"skip: true"},
		},
		{
			name:     "ptr return",
			target:   "Ptr",
			gapType:  "return_path",
			testName: "TestPtrReturn",
			want: []string{
				"skip: false",
				"v:    1",
				"got := Ptr[int](tt.v)",
				"if got == nil",
				`got nil pointer`,
				"if *got != tt.v",
				"got %v, want %v",
			},
			notWant: []string{"skip: true", "ret0 *int", "got != tt.ret0"},
		},
		{
			name:     "user type default return",
			target:   "UserTypeOf",
			gapType:  "return_path",
			testName: "TestUserTypeOfDefault",
			want: []string{
				"skip: false",
				"unit: time.Minute",
				"ret0: 1",
				"got := UserTypeOf(tt.unit)",
			},
			notWant: []string{"skip: true"},
		},
		{
			name:     "slice mapper return",
			target:   "SliceMapper0",
			gapType:  "return_path",
			testName: "TestSliceMapper0Return",
			want: []string{
				"skip:   false",
				"src:    []int{1, 1, 2}",
				"mapper: func(i int) int { return i }",
				"ret0:   []int{1, 2}",
				"got := SliceMapper0[int, int](tt.src, tt.mapper)",
			},
			notWant: []string{"skip:   true"},
		},
		{
			name:     "slice mapper statement",
			target:   "SliceMapper0",
			gapType:  "statement",
			testName: "TestSliceMapper0Statement",
			want: []string{
				"skip:   false",
				"src:    []int{1, 1, 2}",
				"mapper: func(i int) int { return i }",
				"ret0:   []int{1, 2}",
				"got := SliceMapper0[int, int](tt.src, tt.mapper)",
			},
			notWant: []string{"skip:   true"},
		},
		{
			name:     "trim space non-empty",
			target:   "TrimSpaceSlice",
			gapType:  "branch",
			branch:   `v != ""`,
			testName: "TestTrimSpaceSliceNonEmpty",
			want: []string{
				"skip: false",
				`s:    []string{" a ", " ", "b"}`,
				`ret0: []string{"a", "b"}`,
				"got := TrimSpaceSlice(tt.s)",
				"if !reflect.DeepEqual(got, tt.ret0)",
			},
			notWant: []string{"skip: true"},
		},
		{
			name:     "trim space return",
			target:   "TrimSpaceSlice",
			gapType:  "return_path",
			testName: "TestTrimSpaceSliceReturn",
			want: []string{
				"skip: false",
				`s:    []string{" a ", " ", "b"}`,
				`ret0: []string{"a", "b"}`,
				"got := TrimSpaceSlice(tt.s)",
				"if !reflect.DeepEqual(got, tt.ret0)",
			},
			notWant: []string{"skip: true"},
		},
		{
			name:     "trim space statement",
			target:   "TrimSpaceSlice",
			gapType:  "statement",
			testName: "TestTrimSpaceSliceStatement",
			want: []string{
				"skip: false",
				`s:    []string{" a ", " ", "b"}`,
				`ret0: []string{"a", "b"}`,
				"got := TrimSpaceSlice(tt.s)",
				"if !reflect.DeepEqual(got, tt.ret0)",
			},
			notWant: []string{"skip: true"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := types.CoverageTestTask{
				ID:              "go-test-alias",
				Framework:       "go-test",
				Target:          tt.target,
				LineRange:       "1-1",
				GapType:         tt.gapType,
				TestName:        tt.testName,
				MissingBranches: []string{"未覆盖 if 分支: " + tt.branch},
				SuggestedInputs: []string{"构造满足条件 `" + tt.branch + "` 的输入"},
				AssertionFocus:  []string{"断言工具函数分支返回值"},
			}
			code, err := GenerateGoTestsForCoverageTask(srcPath, &task)
			if err != nil {
				t.Fatalf("GenerateGoTestsForCoverageTask() error = %v", err)
			}
			for _, want := range append([]string{"func " + tt.testName + "(t *testing.T)"}, tt.want...) {
				if !strings.Contains(code, want) {
					t.Fatalf("expected %q in generated code:\n%s", want, code)
				}
			}
			for _, notWant := range tt.notWant {
				if strings.Contains(code, notWant) {
					t.Fatalf("did not expect %q in generated code:\n%s", notWant, code)
				}
			}
		})
	}
}

func TestGenerateGoTestsForCoverageTaskUsesStringBoolAndNilBranchInputs(t *testing.T) {
	src := `package sample

type User struct {
	Name string
}

func Label(name string) string {
	if name == "" {
		return "anonymous"
	}
	return name
}

func Normalize(name string) string {
	if name != "" {
		return name
	}
	return "anonymous"
}

func Toggle(enabled bool) string {
	if enabled == false {
		return "off"
	}
	return "on"
}

func UserName(user *User) string {
	if user == nil {
		return "missing"
	}
	return user.Name
}

func ActiveUser(user *User) string {
	if user != nil {
		return "present"
	}
	return "missing"
}

func ErrorMessage(err error) string {
	if err != nil {
		return "failed"
	}
	return "ok"
}

func NoError(err error) string {
	if err == nil {
		return "ok"
	}
	return "failed"
}

func SkipLabel(skip bool) string {
	if skip == false {
		return "run"
	}
	return "skip"
}
`
	srcPath := filepath.Join(t.TempDir(), "branches.go")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name      string
		target    string
		testName  string
		condition string
		wants     []string
	}{
		{
			name:      "empty string",
			target:    "Label",
			testName:  "TestLabelEmptyBranch",
			condition: `name == ""`,
			wants:     []string{`nameValue: ""`, `got := Label(tt.nameValue)`, `ret0:      "anonymous"`},
		},
		{
			name:      "non empty string",
			target:    "Normalize",
			testName:  "TestNormalizeNonEmptyBranch",
			condition: `name != ""`,
			wants:     []string{`nameValue: "test"`, `got := Normalize(tt.nameValue)`, `ret0:      "test"`},
		},
		{
			name:      "false bool",
			target:    "Toggle",
			testName:  "TestToggleFalseBranch",
			condition: "enabled == false",
			wants:     []string{`enabled: false`, `ret0:    "off"`},
		},
		{
			name:      "nil pointer",
			target:    "UserName",
			testName:  "TestUserNameNilBranch",
			condition: "user == nil",
			wants:     []string{`user: nil`, `ret0: "missing"`},
		},
		{
			name:      "non nil pointer",
			target:    "ActiveUser",
			testName:  "TestActiveUserNonNilBranch",
			condition: "user != nil",
			wants:     []string{`user: &User{}`, `ret0: "present"`},
		},
		{
			name:      "non nil error",
			target:    "ErrorMessage",
			testName:  "TestErrorMessageNonNilBranch",
			condition: "err != nil",
			wants:     []string{`"errors"`, `err:  errors.New("test")`, `ret0: "failed"`},
		},
		{
			name:      "nil error",
			target:    "NoError",
			testName:  "TestNoErrorNilBranch",
			condition: "err == nil",
			wants:     []string{`err:  nil`, `ret0: "ok"`},
		},
		{
			name:      "skip parameter",
			target:    "SkipLabel",
			testName:  "TestSkipLabelFalseBranch",
			condition: "skip == false",
			wants:     []string{`skipValue: false`, `got := SkipLabel(tt.skipValue)`, `ret0:      "run"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := types.CoverageTestTask{
				ID:              "go-test-branch-" + tt.target,
				Framework:       "go-test",
				Target:          tt.target,
				LineRange:       "1-1",
				GapType:         "branch",
				TestName:        tt.testName,
				SuggestedInputs: []string{"构造满足条件 `" + tt.condition + "` 的输入"},
				AssertionFocus:  []string{"断言分支返回值"},
			}

			code, err := GenerateGoTestsForCoverageTask(srcPath, &task)
			if err != nil {
				t.Fatalf("GenerateGoTestsForCoverageTask() error = %v", err)
			}
			for _, want := range append([]string{
				"func " + tt.testName + "(t *testing.T)",
				"skip:",
				"got := " + tt.target + "(",
				"if got != tt.ret0",
			}, tt.wants...) {
				if !strings.Contains(code, want) {
					t.Fatalf("expected %q in generated code:\n%s", want, code)
				}
			}
			if strings.Contains(code, "skip: true") || strings.Contains(code, "TODO: 填写有意义的输入") {
				t.Fatalf("did not expect skipped TODO in generated code:\n%s", code)
			}
		})
	}
}

func TestGenerateGoTestsForCoverageTaskUsesStringGuardForMultiReturnErrorPath(t *testing.T) {
	src := `package sample

import "fmt"

type Status struct {
	SocketPath string
}

func QueryStatus(socketPath string) (Status, error) {
	if socketPath == "" {
		return Status{}, fmt.Errorf("socket path is required")
	}
	return Status{SocketPath: socketPath}, nil
}
`
	srcPath := filepath.Join(t.TempDir(), "client.go")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	task := types.CoverageTestTask{
		ID:              "go-test-query-status",
		Framework:       "go-test",
		Target:          "QueryStatus",
		LineRange:       "10-12",
		GapType:         "branch",
		TestName:        "TestQueryStatus",
		MissingBranches: []string{`未覆盖 if 分支: socketPath == ""`},
		SuggestedInputs: []string{`构造满足条件 ` + "`socketPath == \"\"`" + ` 的输入`},
		AssertionFocus:  []string{"断言错误分支返回值"},
	}

	code, err := GenerateGoTestsForCoverageTask(srcPath, &task)
	if err != nil {
		t.Fatalf("GenerateGoTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		`"reflect"`,
		"func TestQueryStatus(t *testing.T)",
		"skip:       false",
		`socketPath: ""`,
		"ret0:       *new(Status)",
		"got0, got1 := QueryStatus(tt.socketPath)",
		"if got1 == nil",
		"if !reflect.DeepEqual(got0, tt.ret0)",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	if strings.Contains(code, "skip: true") || strings.Contains(code, "TODO: 填写有意义的输入") {
		t.Fatalf("did not expect skipped TODO in generated code:\n%s", code)
	}
}

func TestGenerateGoTestsForCoverageTaskUsesVariadicStringGuardForMultiReturnErrorPath(t *testing.T) {
	src := `package sample

import "fmt"

type ControlOptions struct {
	Force bool
}

type ControlResponse struct {
	OK bool
}

func SendControl(socketPath, control string, opts ...ControlOptions) (ControlResponse, error) {
	if control == "" {
		return ControlResponse{}, fmt.Errorf("control is required")
	}
	return ControlResponse{OK: true}, nil
}
`
	srcPath := filepath.Join(t.TempDir(), "client.go")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	task := types.CoverageTestTask{
		ID:              "go-test-send-control",
		Framework:       "go-test",
		Target:          "SendControl",
		LineRange:       "13-15",
		GapType:         "branch",
		TestName:        "TestSendControl",
		MissingBranches: []string{`未覆盖 if 分支: control == ""`},
		SuggestedInputs: []string{`构造满足条件 ` + "`control == \"\"`" + ` 的输入`},
		AssertionFocus:  []string{"断言错误分支返回值"},
	}

	code, err := GenerateGoTestsForCoverageTask(srcPath, &task)
	if err != nil {
		t.Fatalf("GenerateGoTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"func TestSendControl(t *testing.T)",
		"opts       []ControlOptions",
		"skip:       false",
		`socketPath: "test"`,
		`control:    ""`,
		"opts:       nil",
		"ret0:       *new(ControlResponse)",
		"got0, got1 := SendControl(tt.socketPath, tt.control, tt.opts...)",
		"if got1 == nil",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	if strings.Contains(code, "skip: true") || strings.Contains(code, "TODO: 填写有意义的输入") {
		t.Fatalf("did not expect skipped TODO in generated code:\n%s", code)
	}
}

func TestGenerateGoTestsForCoverageTaskUsesMissingSocketForNetDialErrorPath(t *testing.T) {
	src := `package sample

import "net"

type Status struct {
	SocketPath string
}

func QueryStatus(socketPath string) (Status, error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return Status{}, err
	}
	defer conn.Close()
	return Status{SocketPath: socketPath}, nil
}
`
	srcPath := filepath.Join(t.TempDir(), "client.go")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	task := types.CoverageTestTask{
		ID:              "go-test-query-status",
		Framework:       "go-test",
		Target:          "QueryStatus",
		LineRange:       "10-12",
		GapType:         "branch",
		TestName:        "TestQueryStatus",
		MissingBranches: []string{`未覆盖 if 分支: err != nil`},
		SuggestedInputs: []string{`构造满足条件 ` + "`err != nil`" + ` 的输入`},
		AssertionFocus:  []string{"断言错误分支返回值"},
	}

	code, err := GenerateGoTestsForCoverageTask(srcPath, &task)
	if err != nil {
		t.Fatalf("GenerateGoTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		`"path/filepath"`,
		"skip:       false",
		`socketPath: filepath.Join(t.TempDir(), "missing.sock")`,
		"got0, got1 := QueryStatus(tt.socketPath)",
		"if got1 == nil",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	if strings.Contains(code, "skip: true") || strings.Contains(code, "TODO: 填写有意义的输入") {
		t.Fatalf("did not expect skipped TODO in generated code:\n%s", code)
	}
}

func TestGenerateGoTestsForCoverageTaskDisambiguatesRepeatedErrBranchesByLineRange(t *testing.T) {
	src := `package sample

import "net"

type Status struct {
	SocketPath string
}

func QueryStatus(socketPath string) (Status, error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return Status{}, err
	}
	_, err = conn.Write([]byte("status\n"))
	if err != nil {
		return Status{}, err
	}
	return Status{SocketPath: socketPath}, nil
}
`
	srcPath := filepath.Join(t.TempDir(), "client.go")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	task := types.CoverageTestTask{
		ID:              "go-test-query-status-write",
		Framework:       "go-test",
		Target:          "QueryStatus",
		LineRange:       "14-16",
		GapType:         "branch",
		TestName:        "TestQueryStatusWriteError",
		MissingBranches: []string{`未覆盖 if 分支: err != nil`},
		SuggestedInputs: []string{`构造满足条件 ` + "`err != nil`" + ` 的输入`},
		AssertionFocus:  []string{"断言写入错误分支返回值"},
	}

	code, err := GenerateGoTestsForCoverageTask(srcPath, &task)
	if err != nil {
		t.Fatalf("GenerateGoTestsForCoverageTask() error = %v", err)
	}
	if !strings.Contains(code, "skip:       true") {
		t.Fatalf("expected unsupported write error branch to stay skipped:\n%s", code)
	}
	if strings.Contains(code, "filepath.Join") || strings.Contains(code, `"path/filepath"`) {
		t.Fatalf("did not expect net.Dial missing socket seed for write branch:\n%s", code)
	}
}

func TestGenerateGoTestsForCoverageTaskUsesUnixSocketCloseForReadBytesErrorPath(t *testing.T) {
	src := `package sample

import (
	"bufio"
	"encoding/json"
	"net"
)

type Status struct{}
type BindRequest struct{ Control string }

func QueryStatus(socketPath string) (Status, error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return Status{}, err
	}
	bind, _ := json.Marshal(BindRequest{Control: "status"})
	if _, err := conn.Write(append(bind, '\n')); err != nil {
		return Status{}, err
	}
	line, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil {
		return Status{}, err
	}
	_ = line
	return Status{}, nil
}
`
	srcPath := filepath.Join(t.TempDir(), "client.go")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	task := types.CoverageTestTask{
		ID:              "go-test-query-status-read",
		Framework:       "go-test",
		Target:          "QueryStatus",
		LineRange:       "22-24",
		GapType:         "branch",
		TestName:        "TestQueryStatusReadError",
		MissingBranches: []string{`未覆盖 if 分支: err != nil`},
		SuggestedInputs: []string{`构造满足条件 ` + "`err != nil`" + ` 的输入`},
		AssertionFocus:  []string{"断言读取错误分支返回值"},
	}

	code, err := GenerateGoTestsForCoverageTask(srcPath, &task)
	if err != nil {
		t.Fatalf("GenerateGoTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		`"bufio"`,
		`"net"`,
		`"path/filepath"`,
		"skip:       false",
		`socketPath: filepath.Join(t.TempDir(), "s")`,
		`listener, listenErr := net.Listen("unix", tt.socketPath)`,
		"_, _ = bufio.NewReader(conn).ReadBytes('\\n')",
		"_ = conn.Close()",
		"got0, got1 := QueryStatus(tt.socketPath)",
		"if got1 == nil",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	if strings.Contains(code, "skip: true") || strings.Contains(code, "TODO: 填写有意义的输入") {
		t.Fatalf("did not expect skipped TODO in generated code:\n%s", code)
	}
}

func TestGenerateGoTestsForCoverageTaskUsesInvalidUnixSocketJSONForUnmarshalErrorPath(t *testing.T) {
	src := `package sample

import (
	"bufio"
	"encoding/json"
	"net"
)

type Status struct{}
type BindRequest struct{ Control string }

func QueryStatus(socketPath string) (Status, error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return Status{}, err
	}
	bind, _ := json.Marshal(BindRequest{Control: "status"})
	if _, err := conn.Write(append(bind, '\n')); err != nil {
		return Status{}, err
	}
	line, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil {
		return Status{}, err
	}
	var status Status
	if err := json.Unmarshal(line, &status); err != nil {
		return Status{}, err
	}
	return status, nil
}
`
	srcPath := filepath.Join(t.TempDir(), "client.go")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	task := types.CoverageTestTask{
		ID:              "go-test-query-status-json",
		Framework:       "go-test",
		Target:          "QueryStatus",
		LineRange:       "26-28",
		GapType:         "branch",
		TestName:        "TestQueryStatusJSONError",
		MissingBranches: []string{`未覆盖 if 分支: err != nil`},
		SuggestedInputs: []string{`构造满足条件 ` + "`err != nil`" + ` 的输入`},
		AssertionFocus:  []string{"断言 JSON 解码错误分支返回值"},
	}

	code, err := GenerateGoTestsForCoverageTask(srcPath, &task)
	if err != nil {
		t.Fatalf("GenerateGoTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		`"bufio"`,
		`"net"`,
		`"path/filepath"`,
		"skip:       false",
		`socketPath: filepath.Join(t.TempDir(), "s")`,
		`listener, listenErr := net.Listen("unix", tt.socketPath)`,
		"_, _ = bufio.NewReader(conn).ReadBytes('\\n')",
		"_, _ = conn.Write([]byte(\"{\\n\"))",
		"got0, got1 := QueryStatus(tt.socketPath)",
		"if got1 == nil",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	if strings.Contains(code, "skip: true") || strings.Contains(code, "TODO: 填写有意义的输入") {
		t.Fatalf("did not expect skipped TODO in generated code:\n%s", code)
	}
}

func TestGenerateGoTestsForCoverageTaskUsesUnixSocketBindDefaultErrorResponse(t *testing.T) {
	src := `package sample

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"net"
)

type BindRequest struct{ Name string }
type BindResponse struct {
	OK    bool   ` + "`json:\"ok\"`" + `
	Error string ` + "`json:\"error\"`" + `
}

func RunClient(socketPath, name string, stdin io.Reader, stdout io.Writer) error {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return err
	}
	bind, _ := json.Marshal(BindRequest{Name: name})
	if _, err := conn.Write(append(bind, '\n')); err != nil {
		return err
	}
	line, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil {
		return err
	}
	var resp BindResponse
	if err := json.Unmarshal(line, &resp); err != nil {
		return err
	}
	if !resp.OK {
		if resp.Error == "" {
			resp.Error = "bind failed"
		}
		return errors.New(resp.Error)
	}
	return nil
}
`
	srcPath := filepath.Join(t.TempDir(), "client.go")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	task := types.CoverageTestTask{
		ID:              "go-test-run-client-bind-default",
		Framework:       "go-test",
		Target:          "RunClient",
		LineRange:       "34-36",
		GapType:         "branch",
		TestName:        "TestRunClientBindDefaultError",
		MissingBranches: []string{`未覆盖 if 分支: resp.Error == ""`},
		SuggestedInputs: []string{`构造满足条件 ` + "`resp.Error == \"\"`" + ` 的输入`},
		AssertionFocus:  []string{"断言绑定默认错误"},
	}

	code, err := GenerateGoTestsForCoverageTask(srcPath, &task)
	if err != nil {
		t.Fatalf("GenerateGoTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		`"bufio"`,
		`"net"`,
		`"path/filepath"`,
		"skip:       false",
		`socketPath: filepath.Join(t.TempDir(), "s")`,
		`_, _ = conn.Write([]byte("{\"ok\":false}" + "\n"))`,
		"err := RunClient(tt.socketPath, tt.nameValue, tt.stdin, tt.stdout)",
		"if err == nil",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	if strings.Contains(code, "skip: true") || strings.Contains(code, "TODO: 填写有意义的输入") {
		t.Fatalf("did not expect skipped TODO in generated code:\n%s", code)
	}
}

func TestGenerateGoTestsForCoverageTaskUsesUnixSocketInvalidStatusResponse(t *testing.T) {
	src := `package sample

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
)

type BindRequest struct{ Control string }
type BindResponse struct {
	OK    bool   ` + "`json:\"ok\"`" + `
	Error string ` + "`json:\"error\"`" + `
}
type Status struct {
	SocketPath string
	Servers    []string
}

func QueryStatus(socketPath string) (Status, error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return Status{}, err
	}
	bind, _ := json.Marshal(BindRequest{Control: "status"})
	if _, err := conn.Write(append(bind, '\n')); err != nil {
		return Status{}, err
	}
	line, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil {
		return Status{}, err
	}
	var bindResp BindResponse
	if err := json.Unmarshal(line, &bindResp); err == nil && !bindResp.OK && bindResp.Error != "" {
		return Status{}, fmt.Errorf(bindResp.Error)
	}
	var status Status
	if err := json.Unmarshal(line, &status); err != nil {
		return Status{}, err
	}
	if status.SocketPath == "" && status.Servers == nil {
		return Status{}, fmt.Errorf("invalid status response")
	}
	return status, nil
}
`
	srcPath := filepath.Join(t.TempDir(), "client.go")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	task := types.CoverageTestTask{
		ID:              "go-test-query-status-invalid",
		Framework:       "go-test",
		Target:          "QueryStatus",
		LineRange:       "40-42",
		GapType:         "branch",
		TestName:        "TestQueryStatusInvalidResponse",
		MissingBranches: []string{`未覆盖 if 分支: status.SocketPath == "" && status.Servers == nil`},
		SuggestedInputs: []string{`构造满足条件 ` + "`status.SocketPath == \"\" && status.Servers == nil`" + ` 的输入`},
		AssertionFocus:  []string{"断言 invalid status response"},
	}

	code, err := GenerateGoTestsForCoverageTask(srcPath, &task)
	if err != nil {
		t.Fatalf("GenerateGoTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		`"bufio"`,
		`"net"`,
		`"path/filepath"`,
		"skip:       false",
		`socketPath: filepath.Join(t.TempDir(), "s")`,
		`_, _ = conn.Write([]byte("{}" + "\n"))`,
		"got0, got1 := QueryStatus(tt.socketPath)",
		"if got1 == nil",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	if strings.Contains(code, "skip: true") || strings.Contains(code, "TODO: 填写有意义的输入") {
		t.Fatalf("did not expect skipped TODO in generated code:\n%s", code)
	}
}

func TestGenerateGoTestsForCoverageTaskUsesUnixSocketControlDefaultErrorResponse(t *testing.T) {
	src := `package sample

import (
	"bufio"
	"encoding/json"
	"net"
)

type BindRequest struct{ Control string }
type ControlOptions struct{}
type ControlResponse struct {
	OK    bool   ` + "`json:\"ok\"`" + `
	Error string ` + "`json:\"error\"`" + `
}

func SendControl(socketPath, control string, opts ...ControlOptions) (ControlResponse, error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return ControlResponse{}, err
	}
	bind, _ := json.Marshal(BindRequest{Control: control})
	if _, err := conn.Write(append(bind, '\n')); err != nil {
		return ControlResponse{}, err
	}
	line, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil {
		return ControlResponse{}, err
	}
	var resp ControlResponse
	if err := json.Unmarshal(line, &resp); err != nil {
		return ControlResponse{}, err
	}
	if !resp.OK && resp.Error == "" {
		resp.Error = "control failed"
	}
	return resp, nil
}
`
	srcPath := filepath.Join(t.TempDir(), "client.go")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	task := types.CoverageTestTask{
		ID:              "go-test-send-control-default",
		Framework:       "go-test",
		Target:          "SendControl",
		LineRange:       "32-34",
		GapType:         "branch",
		TestName:        "TestSendControlDefaultError",
		MissingBranches: []string{`未覆盖 if 分支: !resp.OK && resp.Error == ""`},
		SuggestedInputs: []string{`构造满足条件 ` + "`!resp.OK && resp.Error == \"\"`" + ` 的输入`},
		AssertionFocus:  []string{"断言默认控制错误响应"},
	}

	code, err := GenerateGoTestsForCoverageTask(srcPath, &task)
	if err != nil {
		t.Fatalf("GenerateGoTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		`"bufio"`,
		`"net"`,
		`"path/filepath"`,
		`"reflect"`,
		"skip:       false",
		`socketPath: filepath.Join(t.TempDir(), "s")`,
		`_, _ = conn.Write([]byte("{\"ok\":false}" + "\n"))`,
		`ret0:       ControlResponse{Error: "control failed"}`,
		"ret1:       nil",
		"got0, got1 := SendControl(tt.socketPath, tt.control, tt.opts...)",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	if strings.Contains(code, "skip: true") || strings.Contains(code, "TODO: 填写有意义的输入") {
		t.Fatalf("did not expect skipped TODO in generated code:\n%s", code)
	}
}

func TestGenerateGoTestsForCoverageTaskExplainsUnsafeBranchFallback(t *testing.T) {
	src := `package sample

type User struct {
	Name string
}

func UserName(user *User) string {
	if user != nil {
		return user.Name
	}
	return "missing"
}
`
	srcPath := filepath.Join(t.TempDir(), "user.go")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	task := types.CoverageTestTask{
		ID:              "go-test-branch-UserName",
		Framework:       "go-test",
		Target:          "UserName",
		LineRange:       "8-10",
		GapType:         "branch",
		TestName:        "TestUserNameNonNilBranch",
		SuggestedInputs: []string{"构造满足条件 `user != nil` 的输入"},
		AssertionFocus:  []string{"断言分支返回值"},
	}

	code, err := GenerateGoTestsForCoverageTask(srcPath, &task)
	if err != nil {
		t.Fatalf("GenerateGoTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"func TestUserNameNonNilBranch(t *testing.T)",
		`Static generator cannot infer exact coverage case: branch "user != nil" returns "user.Name", which needs manual expected value review.`,
		"skip: true",
		"user: nil",
		`ret0: ""`,
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	if strings.Contains(code, "user: &User{}") || strings.Contains(code, `ret0: "test"`) {
		t.Fatalf("did not expect unsafe exact seed in generated code:\n%s", code)
	}
}

func TestGenerateGoTestsForCoverageTaskSynthesizesAndCompoundBranch(t *testing.T) {
	src := `package sample

func Score(a, b int) int {
	if a > 0 && b > 0 {
		return 1
	}
	return 0
}
`
	srcPath := filepath.Join(t.TempDir(), "score.go")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	task := types.CoverageTestTask{
		ID:              "go-test-branch-Score",
		Framework:       "go-test",
		Target:          "Score",
		LineRange:       "4-6",
		GapType:         "branch",
		TestName:        "TestScorePositiveBranch",
		SuggestedInputs: []string{"构造满足条件 `a > 0 && b > 0` 的输入"},
		AssertionFocus:  []string{"断言分支返回值"},
	}

	code, err := GenerateGoTestsForCoverageTask(srcPath, &task)
	if err != nil {
		t.Fatalf("GenerateGoTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"func TestScorePositiveBranch(t *testing.T)",
		"skip: false",
		"a:    1",
		"b:    1",
		"ret0: 1",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	for _, notWant := range []string{
		"skip: true",
		"multi-parameter input synthesis is not supported yet",
	} {
		if strings.Contains(code, notWant) {
			t.Fatalf("did not expect %q in generated code:\n%s", notWant, code)
		}
	}
}

func TestGenerateGoTestsForCoverageTaskSynthesizesRepeatedParamIntegerRange(t *testing.T) {
	src := `package sample

func InRange(a int) string {
	if a > 0 && a < 10 {
		return "inside"
	}
	return "outside"
}
`
	srcPath := filepath.Join(t.TempDir(), "range.go")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	task := types.CoverageTestTask{
		ID:              "go-test-branch-InRange",
		Framework:       "go-test",
		Target:          "InRange",
		LineRange:       "4-6",
		GapType:         "branch",
		TestName:        "TestInRangeInsideBranch",
		SuggestedInputs: []string{"构造满足条件 `a > 0 && a < 10` 的输入"},
		AssertionFocus:  []string{"断言分支返回值"},
	}

	code, err := GenerateGoTestsForCoverageTask(srcPath, &task)
	if err != nil {
		t.Fatalf("GenerateGoTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"func TestInRangeInsideBranch(t *testing.T)",
		"skip: false",
		"a:    1",
		`ret0: "inside"`,
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	if strings.Contains(code, "skip: true") || strings.Contains(code, "repeats parameter") {
		t.Fatalf("did not expect fallback for supported integer range:\n%s", code)
	}
}

func TestGenerateGoTestsForCoverageTaskExplainsOrCompoundBranchFallback(t *testing.T) {
	src := `package sample

func Score(a, b int) int {
	if a > 0 || b > 0 {
		return 1
	}
	return 0
}
`
	srcPath := filepath.Join(t.TempDir(), "score.go")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	task := types.CoverageTestTask{
		ID:              "go-test-branch-Score",
		Framework:       "go-test",
		Target:          "Score",
		LineRange:       "4-6",
		GapType:         "branch",
		TestName:        "TestScorePositiveBranch",
		SuggestedInputs: []string{"构造满足条件 `a > 0 || b > 0` 的输入"},
		AssertionFocus:  []string{"断言分支返回值"},
	}

	code, err := GenerateGoTestsForCoverageTask(srcPath, &task)
	if err != nil {
		t.Fatalf("GenerateGoTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"func TestScorePositiveBranch(t *testing.T)",
		`Static generator cannot infer exact coverage case: branch "a > 0 || b > 0" uses ||; only simple && compound input synthesis is supported.`,
		"skip: true",
		"a:    0",
		"b:    0",
		"ret0: 0",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	if strings.Contains(code, "skip: false") || strings.Contains(code, "ret0: 1") {
		t.Fatalf("did not expect exact seed for || branch:\n%s", code)
	}
}

func TestGenerateGoTestsForCoverageTaskUsesSmokeCaseForNoArgReturn(t *testing.T) {
	src := `package sample

import "time"

func GetCurrentDate() time.Time {
	return time.Now()
}
`
	srcPath := filepath.Join(t.TempDir(), "time.go")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	task := types.CoverageTestTask{
		ID:             "go-test-time",
		Framework:      "go-test",
		Target:         "GetCurrentDate",
		LineRange:      "5-7",
		GapType:        "return_path",
		TestName:       "TestGetCurrentDate",
		AssertionFocus: []string{"覆盖未执行返回路径"},
	}

	code, err := GenerateGoTestsForCoverageTask(srcPath, &task)
	if err != nil {
		t.Fatalf("GenerateGoTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"func TestGetCurrentDate(t *testing.T)",
		"name: \"coverage return path\"",
		"skip: false",
		"got := GetCurrentDate()",
		"_ = got",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	for _, notWant := range []string{
		"skip: true",
		"TODO: 填写有意义的输入",
		"ret0 string",
		"\"reflect\"",
		"if got != tt.ret0",
	} {
		if strings.Contains(code, notWant) {
			t.Fatalf("did not expect %q in generated code:\n%s", notWant, code)
		}
	}
}

func TestGenerateGoTestsForCoverageTaskAssertsTimeNowFormatReturn(t *testing.T) {
	src := `package sample

import "time"

func GetNowDate() string {
	return time.Now().Format("2006-01-02")
}
`
	srcPath := filepath.Join(t.TempDir(), "time.go")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	task := types.CoverageTestTask{
		ID:             "go-test-time-format",
		Framework:      "go-test",
		Target:         "GetNowDate",
		LineRange:      "5-7",
		GapType:        "return_path",
		TestName:       "TestGetNowDate",
		AssertionFocus: []string{"断言日期格式"},
	}

	code, err := GenerateGoTestsForCoverageTask(srcPath, &task)
	if err != nil {
		t.Fatalf("GenerateGoTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"\"time\"",
		"func TestGetNowDate(t *testing.T)",
		"\"coverage return path\"",
		"skip:   false",
		"layout: \"2006-01-02\"",
		"got := GetNowDate()",
		"if _, err := time.Parse(tt.layout, got); err != nil",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	for _, notWant := range []string{
		"skip: true",
		"_ = got",
		"ret0 string",
		"if got != tt.ret0",
	} {
		if strings.Contains(code, notWant) {
			t.Fatalf("did not expect %q in generated code:\n%s", notWant, code)
		}
	}
}

func TestGenerateGoTestsForCoverageTaskAssertsTimeDateZeroReturn(t *testing.T) {
	src := `package sample

import "time"

func GetCurrentDate() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}
`
	srcPath := filepath.Join(t.TempDir(), "time.go")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	task := types.CoverageTestTask{
		ID:             "go-test-current-date",
		Framework:      "go-test",
		Target:         "GetCurrentDate",
		LineRange:      "5-7",
		GapType:        "return_path",
		TestName:       "TestGetCurrentDate",
		AssertionFocus: []string{"断言日期归零"},
	}

	code, err := GenerateGoTestsForCoverageTask(srcPath, &task)
	if err != nil {
		t.Fatalf("GenerateGoTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"func TestGetCurrentDate(t *testing.T)",
		"\"coverage return path\"",
		"skip: false",
		"got := GetCurrentDate()",
		"got.Hour() != 0 || got.Minute() != 0 || got.Second() != 0 || got.Nanosecond() != 0",
		"want date boundary",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	for _, notWant := range []string{
		"skip: true",
		"_ = got",
		"ret0 time.Time",
		"\"reflect\"",
		"if !reflect.DeepEqual",
	} {
		if strings.Contains(code, notWant) {
			t.Fatalf("did not expect %q in generated code:\n%s", notWant, code)
		}
	}
}

func TestGenerateGoTestsForCoverageTaskAssertsBeforeSaveDefaultBranch(t *testing.T) {
	src := `package sample

import (
	"strings"

	"gorm.io/gorm"
)

type Role struct {
	Code string
	Name string
	Desc string
	DataScope string
}

func (r *Role) BeforeSave(tx *gorm.DB) error {
	r.Code = strings.TrimSpace(r.Code)
	r.Name = strings.TrimSpace(r.Name)
	r.Desc = strings.TrimSpace(r.Desc)
	r.DataScope = strings.TrimSpace(r.DataScope)
	if r.DataScope == "" {
		r.DataScope = "self"
	}
	return nil
}
`
	srcPath := filepath.Join(t.TempDir(), "role.go")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	task := types.CoverageTestTask{
		ID:              "go-test-role-before-save",
		Framework:       "go-test",
		Target:          "Role.BeforeSave",
		LineRange:       "17-19",
		GapType:         "branch",
		TestName:        "TestRoleBeforeSave",
		MissingBranches: []string{"未覆盖 if 分支: r.DataScope == \"\""},
		SuggestedInputs: []string{"构造满足条件 `r.DataScope == \"\"` 的输入"},
		AssertionFocus:  []string{"断言默认 DataScope"},
	}

	code, err := GenerateGoTestsForCoverageTask(srcPath, &task)
	if err != nil {
		t.Fatalf("GenerateGoTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		`"gorm.io/gorm"`,
		"func TestRoleBeforeSave(t *testing.T)",
		"tx   *gorm.DB",
		"skip: false",
		"r := &Role{}",
		`r.Code = " admin "`,
		`r.DataScope = " "`,
		"err := r.BeforeSave(tt.tx)",
		`if r.DataScope != "self" {`,
		`t.Errorf("DataScope = %q, want %q", r.DataScope, "self")`,
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	for _, notWant := range []string{
		"skip: true",
		"TODO: 填写有意义的输入",
		"ret0 error",
	} {
		if strings.Contains(code, notWant) {
			t.Fatalf("did not expect %q in generated code:\n%s", notWant, code)
		}
	}
}

func TestGenerateGoTestsForCoverageTaskAssertsBeforeSaveTrimOnly(t *testing.T) {
	src := `package sample

import (
	"strings"

	"gorm.io/gorm"
)

type User struct {
	UUID string
	NickName string
	Phone string
	Email string
}

func (u *User) BeforeSave(tx *gorm.DB) error {
	u.UUID = strings.TrimSpace(u.UUID)
	u.NickName = strings.TrimSpace(u.NickName)
	u.Phone = strings.TrimSpace(u.Phone)
	u.Email = strings.TrimSpace(u.Email)
	return nil
}
`
	srcPath := filepath.Join(t.TempDir(), "user.go")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	task := types.CoverageTestTask{
		ID:             "go-test-user-before-save",
		Framework:      "go-test",
		Target:         "User.BeforeSave",
		LineRange:      "15-20",
		GapType:        "error_path",
		TestName:       "TestUserBeforeSave",
		AssertionFocus: []string{"断言返回 nil error 并裁剪字符串字段"},
	}

	code, err := GenerateGoTestsForCoverageTask(srcPath, &task)
	if err != nil {
		t.Fatalf("GenerateGoTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"func TestUserBeforeSave(t *testing.T)",
		`name: "coverage error path"`,
		"skip: false",
		"u := &User{}",
		`u.UUID = " uuid-1 "`,
		`u.NickName = " Admin "`,
		"err := u.BeforeSave(tt.tx)",
		`if u.Email != "admin@example.com" {`,
		`t.Errorf("Email = %q, want %q", u.Email, "admin@example.com")`,
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	if strings.Contains(code, "skip: true") || strings.Contains(code, "ret0 error") {
		t.Fatalf("BeforeSave trim-only task should be executable without return fixture:\n%s", code)
	}
}

func TestGenerateTestsWithProviderOptionsUsesGoCoverageTask(t *testing.T) {
	src := `package sample

func Add(a, b int) int {
	return a + b
}

func Sub(a, b int) int {
	return a - b
}
`
	srcPath := filepath.Join(t.TempDir(), "calc.go")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	task := types.CoverageTestTask{
		ID:        "go-test-1",
		Framework: "go-test",
		Target:    "Add",
		LineRange: "3-3",
		GapType:   "branch",
		TestName:  "TestAddCoverageGap",
	}

	code, err := GenerateTestsWithProviderOptions(context.Background(), srcPath, StaticProvider{}, GenerateTestsOptions{CoverageTask: &task})
	if err != nil {
		t.Fatalf("GenerateTestsWithProviderOptions() error = %v", err)
	}
	if !strings.Contains(code, "func TestAddCoverageGap") || strings.Contains(code, "TestSub") {
		t.Fatalf("expected task-aware static Go output, got:\n%s", code)
	}
}

func TestGenerateGoTestsForCoverageTaskPreservesFunctionParamType(t *testing.T) {
	src := `package sample

func Map(values []int, mapper func(int) int) []int {
	out := make([]int, 0, len(values))
	for _, value := range values {
		out = append(out, mapper(value))
	}
	return out
}
`
	srcPath := filepath.Join(t.TempDir(), "map.go")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	task := types.CoverageTestTask{
		ID:        "go-test-map",
		Framework: "go-test",
		Target:    "Map",
		LineRange: "5-7",
		GapType:   "branch",
		TestName:  "TestMap",
	}

	code, err := GenerateGoTestsForCoverageTask(srcPath, &task)
	if err != nil {
		t.Fatalf("GenerateGoTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"mapper func(int) int",
		"mapper: nil",
		"Map(tt.values, tt.mapper)",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
	if strings.Contains(code, "mapper func()") {
		t.Fatalf("function parameter type lost signature:\n%s", code)
	}
}

func TestGenerateGoTestsForCoverageTaskImportsSelectorParamPackage(t *testing.T) {
	src := `package sample

import "net/http"

func RemoteIP(r *http.Request, fallback string) string {
	if r == nil {
		return fallback
	}
	return r.RemoteAddr
}
`
	srcPath := filepath.Join(t.TempDir(), "http.go")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	task := types.CoverageTestTask{
		ID:        "go-test-http",
		Framework: "go-test",
		Target:    "RemoteIP",
		LineRange: "5-7",
		GapType:   "branch",
		TestName:  "TestRemoteIP",
	}

	code, err := GenerateGoTestsForCoverageTask(srcPath, &task)
	if err != nil {
		t.Fatalf("GenerateGoTestsForCoverageTask() error = %v", err)
	}
	for _, want := range []string{
		"\"net/http\"",
		"r        *http.Request",
		"RemoteIP(tt.r, tt.fallback)",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected %q in generated code:\n%s", want, code)
		}
	}
}

func writeTempGoSource(t *testing.T) string {
	t.Helper()

	src := `package sample

func Add(a, b int) int {
	return a + b
}
`
	path := filepath.Join(t.TempDir(), "calc.go")
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeFakeGotests(t *testing.T, script string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "gotests")
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	return dir
}
