package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type verificationSummary struct {
	OverallStatus  string                `json:"overall_status"`
	FailedCount    int                   `json:"failed_count"`
	MarkdownReport string                `json:"markdown_report"`
	Sections       []verificationSection `json:"sections"`
}

type verificationSection struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	ExitCode *int   `json:"exit_code"`
}

type responsePlan struct {
	Action     string
	Conclusion string
	NextSteps  []string
	Skip       []string
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "onboarding agent response demo failed: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: go run ./examples/onboarding-agent-response-demo <verification-summary.json>")
	}

	summary, err := loadSummary(args[0])
	if err != nil {
		return err
	}

	failedSection, exitCode := firstFailedSection(summary)
	plan := planResponse(summary, failedSection)

	fmt.Printf("结论：%s\n\n", plan.Conclusion)
	fmt.Println("证据：")
	fmt.Printf("- agent_next_step=%s\n", plan.Action)
	fmt.Printf("- overall_status=%s\n", summary.OverallStatus)
	fmt.Printf("- failed_count=%d\n", summary.FailedCount)
	if failedSection != "" {
		fmt.Printf("- failed_section=%s\n", failedSection)
	}
	if exitCode != "" {
		fmt.Printf("- exit_code=%s\n", exitCode)
	}
	if summary.MarkdownReport != "" {
		fmt.Printf("- markdown_report=%s\n", summary.MarkdownReport)
	}

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

func loadSummary(path string) (verificationSummary, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return verificationSummary{}, err
	}
	var summary verificationSummary
	if err := json.Unmarshal(data, &summary); err != nil {
		return verificationSummary{}, fmt.Errorf("%s invalid JSON: %w", path, err)
	}
	if summary.OverallStatus == "" {
		return verificationSummary{}, fmt.Errorf("%s missing overall_status", path)
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

func planResponse(summary verificationSummary, failedSection string) responsePlan {
	if summary.OverallStatus != "failed" && summary.FailedCount == 0 {
		return responsePlan{
			Action:     "ready",
			Conclusion: "testloop-mcp onboarding 链路通过，可以继续真实生成、修复或覆盖率闭环。",
			NextSteps: []string{
				"继续运行项目自己的测试或构建命令，确认业务基线稳定。",
				"选择一个低风险模块进入 generate_tests / run_tests / parse_coverage 闭环。",
			},
			Skip: []string{
				"不继续排查安装、MCP transport 或 onboarding artifact。",
			},
		}
	}
	return planFailedSection(failedSection)
}

func planFailedSection(section string) responsePlan {
	switch {
	case strings.Contains(section, "基础安装"):
		return responsePlan{
			Action:     "fix-installation",
			Conclusion: "失败发生在 testloop-mcp 安装或版本门禁，还没进入用户项目 smoke。",
			NextSteps: []string{
				"先运行 testloop-mcp --version，确认是否等于文档要求的版本。",
				"检查二进制路径、客户端配置 roundtrip 和 HTTP /healthz 输出。",
			},
			Skip: []string{
				"不修改用户项目测试。",
				"不排查覆盖率或生成质量。",
			},
		}
	case strings.Contains(section, "MCP 协议"):
		return responsePlan{
			Action:     "inspect-mcp-transport",
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
	case strings.Contains(section, "Agent 闭环"):
		return responsePlan{
			Action:     "inspect-agent-demo",
			Conclusion: "失败发生在最小 Agent 闭环 demo。",
			NextSteps: []string{
				"检查 demo runner、Go 运行环境和 MCP 结构化返回。",
				"先复跑 examples 里的最小 demo，再判断是否需要改工具实现。",
			},
			Skip: []string{
				"不先排查用户项目依赖。",
			},
		}
	case strings.Contains(section, "用户项目"):
		return responsePlan{
			Action:     "inspect-user-project",
			Conclusion: "testloop-mcp onboarding 链路本身是通的，失败发生在用户项目 smoke。",
			NextSteps: []string{
				"打开 verification-report.md 中“用户项目 smoke”这一节，先看项目测试/构建命令的 stdout / stderr。",
				"在用户项目目录复跑同一条 smoke 命令，确认依赖、环境变量或测试本身是否失败。",
			},
			Skip: []string{
				"不先修改 testloop-mcp 安装或 MCP transport。",
				"不先生成/修改测试，除非项目 smoke 的失败日志明确指向测试缺失或断言失败。",
			},
		}
	case strings.Contains(section, "showcase"):
		return responsePlan{
			Action:     "inspect-showcase",
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
			Action:     "inspect-verification",
			Conclusion: "验收报告存在未知失败 section，需要先打开 summary 和 Markdown 报告定位。",
			NextSteps: []string{
				"先读取 verification-summary.json 的 failed section。",
				"再打开 verification-report.md 中对应 section 的 stdout / stderr。",
			},
			Skip: []string{
				"不根据 GitHub Actions 最后一行错误直接修改代码。",
			},
		}
	}
}
