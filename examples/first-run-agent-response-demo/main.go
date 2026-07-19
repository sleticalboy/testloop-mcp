package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type firstRunContext map[string]string

type verificationSummary struct {
	OverallStatus string                `json:"overall_status"`
	FailedCount   int                   `json:"failed_count"`
	Sections      []verificationSection `json:"sections"`
}

type verificationSection struct {
	Name     string            `json:"name"`
	Status   string            `json:"status"`
	ExitCode *int              `json:"exit_code"`
	Signals  map[string]string `json:"signals"`
}

type responsePlan struct {
	Conclusion string
	NextSteps  []string
	Skip       []string
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "first-run agent response demo failed: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) < 1 || len(args) > 2 {
		return fmt.Errorf("usage: go run ./examples/first-run-agent-response-demo <first-run-context.txt> [verification-summary.json]")
	}

	context, err := loadContext(args[0])
	if err != nil {
		return err
	}

	var summary verificationSummary
	if len(args) == 2 {
		summary, err = loadSummary(args[1])
		if err != nil {
			return err
		}
	}

	action := context["first_run_agent_next_step"]
	if action == "" {
		return fmt.Errorf("%s missing first_run_agent_next_step", args[0])
	}

	failedSection, exitCode := firstFailedSection(summary)
	plan := planResponse(action)

	fmt.Printf("结论：%s\n\n", plan.Conclusion)
	fmt.Println("证据：")
	fmt.Printf("- first_run_agent_next_step=%s\n", action)
	if failedSection != "" {
		fmt.Printf("- failed_section=%s\n", failedSection)
	}
	if exitCode != "" {
		fmt.Printf("- exit_code=%s\n", exitCode)
	}
	if report := context["first_run_report"]; report != "" {
		fmt.Printf("- first_run_report=%s\n", report)
	}
	printSectionSignals(summary.Sections)

	fmt.Println("\n下一步：")
	for _, step := range plan.NextSteps {
		fmt.Printf("- %s\n", step)
	}

	fmt.Println("\n暂不做：")
	for _, item := range plan.Skip {
		fmt.Printf("- %s\n", item)
	}
	return nil
}

func printSectionSignals(sections []verificationSection) {
	for _, section := range sections {
		action := strings.TrimSpace(section.Signals["action"])
		if action == "" {
			continue
		}
		fmt.Printf("- section_signal=%s action=%s\n", section.Name, action)
	}
}

func loadContext(path string) (firstRunContext, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	context := make(firstRunContext)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "testloop-mcp ") || strings.HasSuffix(line, ":") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		context[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return context, nil
}

func loadSummary(path string) (verificationSummary, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return verificationSummary{}, err
	}
	var summary verificationSummary
	if err := json.Unmarshal(data, &summary); err != nil {
		return verificationSummary{}, fmt.Errorf("%s invalid JSON: %w", path, err)
	}
	return summary, nil
}

func firstFailedSection(summary verificationSummary) (string, string) {
	for _, section := range summary.Sections {
		if section.Status != "failed" {
			continue
		}
		if section.ExitCode == nil {
			return section.Name, ""
		}
		return section.Name, fmt.Sprintf("%d", *section.ExitCode)
	}
	return "", ""
}

func planResponse(action string) responsePlan {
	switch action {
	case "ready":
		return responsePlan{
			Conclusion: "testloop-mcp 接入链路通过，可以进入真实测试生成、覆盖率补测或 MCP 客户端接入。",
			NextSteps: []string{
				"继续运行项目自己的测试或构建命令，确认业务基线稳定。",
				"选择一个低风险模块开始生成测试，并用 run_tests / parse_coverage 验证反馈闭环。",
			},
			Skip: []string{
				"不继续排查安装、MCP transport 或 first-run artifact。",
			},
		}
	case "fix-installation":
		return responsePlan{
			Conclusion: "失败发生在 testloop-mcp 安装或版本门禁，还没进入用户项目测试。",
			NextSteps: []string{
				"先运行 testloop-mcp --version，确认是否等于文档要求的版本。",
				"如果是 Homebrew 安装，执行 brew update && brew upgrade sleticalboy/tap/testloop-mcp；仍旧版本时执行 brew reinstall。",
				"重新运行 first-run 诊断，直到基础安装验收通过。",
			},
			Skip: []string{
				"不修改用户项目测试。",
				"不排查覆盖率或生成质量。",
			},
		}
	case "inspect-mcp-transport":
		return responsePlan{
			Conclusion: "失败发生在 MCP transport 或真实协议 smoke。",
			NextSteps: []string{
				"检查 stdio / Streamable HTTP 启动参数、端口占用和客户端配置。",
				"打开 verification-report.md 中 MCP 协议 smoke 的 stdout / stderr。",
			},
			Skip: []string{
				"不先改用户项目测试命令。",
				"不把失败归因到生成质量。",
			},
		}
	case "inspect-agent-demo":
		return responsePlan{
			Conclusion: "失败发生在最小 Agent 闭环 demo。",
			NextSteps: []string{
				"检查 demo runner、Go 运行环境和 MCP 结构化返回。",
				"先复跑 examples 里的最小 demo，再判断是否需要改工具实现。",
			},
			Skip: []string{
				"不先排查用户项目依赖。",
			},
		}
	case "inspect-user-project":
		return responsePlan{
			Conclusion: "testloop-mcp 接入链路本身是通的，失败发生在用户项目 smoke。",
			NextSteps: []string{
				"打开 verification-report.md 中“用户项目 smoke”这一节，先看项目测试/构建命令的 stdout / stderr。",
				"在用户项目目录复跑同一条 smoke 命令，确认依赖、环境变量或测试本身是否失败。",
			},
			Skip: []string{
				"不先修改 testloop-mcp 安装或 MCP transport。",
				"不先生成/修改测试，除非项目 smoke 的失败日志明确指向测试缺失或断言失败。",
			},
		}
	case "inspect-showcase":
		return responsePlan{
			Conclusion: "失败发生在公开 showcase 验证。",
			NextSteps: []string{
				"先区分 GitHub/npm 网络、外部项目 checkout、依赖安装和 action 期望漂移。",
				"查看 showcase 失败 section 的 stdout / stderr，再决定是否需要更新示例或期望。",
			},
			Skip: []string{
				"不把 showcase 网络失败当作用户项目回归。",
			},
		}
	default:
		return responsePlan{
			Conclusion: "first-run action 不在已知分流表中，需要先补齐 artifact 再判断。",
			NextSteps: []string{
				"先读取 agent-decision.txt 和 first-run-context.txt。",
				"如果仍不够，再补 verification-summary.json 和 verification-report.md 的失败 section。",
			},
			Skip: []string{
				"不根据 GitHub Actions 最后一行错误直接修改代码。",
			},
		}
	}
}
