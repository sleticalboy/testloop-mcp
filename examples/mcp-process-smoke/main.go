package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "client smoke failed: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("mcp-process-smoke", flag.ContinueOnError)
	command := flags.String("command", "", "testloop-mcp binary path or command name")
	transport := flags.String("transport", "all", "transport to verify: all, stdio, or http")
	flags.SetOutput(os.Stderr)
	if err := flags.Parse(args); err != nil {
		return err
	}

	binary, err := resolveCommand(*command)
	if err != nil {
		return err
	}
	switch *transport {
	case "all":
		if err := runStdioSmoke(ctx, binary); err != nil {
			return err
		}
		if err := runHTTPSmoke(ctx, binary); err != nil {
			return err
		}
	case "stdio":
		if err := runStdioSmoke(ctx, binary); err != nil {
			return err
		}
	case "http":
		if err := runHTTPSmoke(ctx, binary); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported transport %q; use all, stdio, or http", *transport)
	}
	fmt.Println("client_smoke=pass")
	return nil
}

func resolveCommand(command string) (string, error) {
	if command == "" {
		command = os.Getenv("TESTLOOP_MCP_COMMAND")
	}
	if command == "" {
		command = "testloop-mcp"
	}
	if strings.Contains(command, string(os.PathSeparator)) {
		info, err := os.Stat(command)
		if err != nil {
			return "", err
		}
		if info.IsDir() {
			return "", fmt.Errorf("%s is a directory", command)
		}
		if info.Mode()&0o111 == 0 {
			return "", fmt.Errorf("%s is not executable", command)
		}
		abs, err := filepath.Abs(command)
		if err != nil {
			return "", err
		}
		return abs, nil
	}
	resolved, err := exec.LookPath(command)
	if err != nil {
		return "", err
	}
	return resolved, nil
}

func runStdioSmoke(ctx context.Context, binary string) error {
	sessionCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(sessionCtx, binary, "--transport=stdio")
	session, err := connectClient(sessionCtx, &mcp.CommandTransport{
		Command:           cmd,
		TerminateDuration: 2 * time.Second,
	}, "testloop-stdio-smoke")
	if err != nil {
		return fmt.Errorf("stdio connect: %w", err)
	}
	defer session.Close()
	if err := verifySession(sessionCtx, session, "stdio"); err != nil {
		return err
	}
	return nil
}

func runHTTPSmoke(ctx context.Context, binary string) error {
	addr, err := freeTCPAddr()
	if err != nil {
		return err
	}
	sessionCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	cmd := exec.CommandContext(sessionCtx, binary, "--transport=http", "--addr="+addr)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start http server: %w", err)
	}
	defer func() {
		cancel()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	}()

	baseURL := "http://" + addr
	if err := waitForHealthz(sessionCtx, baseURL+"/healthz"); err != nil {
		return err
	}
	session, err := connectClient(sessionCtx, &mcp.StreamableClientTransport{
		Endpoint:             baseURL + "/mcp",
		DisableStandaloneSSE: true,
	}, "testloop-http-smoke")
	if err != nil {
		return fmt.Errorf("http connect: %w", err)
	}
	defer session.Close()
	if err := verifySession(sessionCtx, session, "http"); err != nil {
		return err
	}
	return nil
}

func connectClient(ctx context.Context, transport mcp.Transport, name string) (*mcp.ClientSession, error) {
	client := mcp.NewClient(&mcp.Implementation{Name: name, Version: "0.0.0"}, nil)
	return client.Connect(ctx, transport, nil)
}

func verifySession(ctx context.Context, session *mcp.ClientSession, label string) error {
	tools, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		return fmt.Errorf("%s list tools: %w", label, err)
	}
	foundParseResults := false
	for _, tool := range tools.Tools {
		if tool.Name == "parse_results" {
			foundParseResults = true
			break
		}
	}
	if !foundParseResults {
		return fmt.Errorf("%s tools/list missing parse_results", label)
	}

	payload, err := callParseResults(ctx, session)
	if err != nil {
		return fmt.Errorf("%s parse_results: %w", label, err)
	}
	if payload["status"] != "pass" || payload["framework"] != "go-test" {
		return fmt.Errorf("%s unexpected parse_results payload: %+v", label, payload)
	}
	fmt.Printf("%s: tools=%d parse_results=%s structuredContent=ok\n", label, len(tools.Tools), payload["status"])
	return nil
}

func callParseResults(ctx context.Context, session *mcp.ClientSession) (map[string]any, error) {
	output := strings.Join([]string{
		`{"Action":"run","Package":"example.com/calc","Test":"TestSmoke"}`,
		`{"Action":"pass","Package":"example.com/calc","Test":"TestSmoke","Elapsed":0}`,
		`{"Action":"pass","Package":"example.com/calc","Elapsed":0}`,
	}, "\n")
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "parse_results",
		Arguments: map[string]any{
			"framework": "go-test",
			"output":    output,
		},
	})
	if err != nil {
		return nil, err
	}
	return decodeToolPayload(result)
}

func decodeToolPayload(result *mcp.CallToolResult) (map[string]any, error) {
	if len(result.Content) == 0 {
		return nil, errors.New("empty tool content")
	}
	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		return nil, fmt.Errorf("content[0] is %T, want TextContent", result.Content[0])
	}
	var textPayload map[string]any
	if err := json.Unmarshal([]byte(textContent.Text), &textPayload); err != nil {
		return nil, fmt.Errorf("decode text JSON: %w", err)
	}
	if result.StructuredContent == nil {
		return nil, errors.New("missing structuredContent")
	}
	data, err := structuredContentJSON(result.StructuredContent)
	if err != nil {
		return nil, err
	}
	var structuredPayload map[string]any
	if err := json.Unmarshal(data, &structuredPayload); err != nil {
		return nil, fmt.Errorf("decode structuredContent JSON: %w", err)
	}
	if !reflect.DeepEqual(textPayload, structuredPayload) {
		return nil, errors.New("structuredContent and text JSON differ")
	}
	return structuredPayload, nil
}

func structuredContentJSON(value any) ([]byte, error) {
	if raw, ok := value.(json.RawMessage); ok {
		return raw, nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal structuredContent: %w", err)
	}
	return data, nil
}

func freeTCPAddr() (string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	defer listener.Close()
	return listener.Addr().String(), nil
}

func waitForHealthz(ctx context.Context, url string) error {
	client := http.Client{Timeout: 250 * time.Millisecond}
	deadline := time.Now().Add(5 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
			lastErr = fmt.Errorf("healthz status %d", resp.StatusCode)
		} else {
			lastErr = err
		}
		time.Sleep(100 * time.Millisecond)
	}
	if lastErr != nil {
		return fmt.Errorf("healthz did not become ready: %w", lastErr)
	}
	return errors.New("healthz did not become ready")
}
