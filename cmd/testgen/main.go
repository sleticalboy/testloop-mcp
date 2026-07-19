package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sleticalboy/testloop-mcp/internal/generator"
)

func main() {
	os.Exit(runTestgen(os.Args[1:], os.Stdout, os.Stderr))
}

func runTestgen(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("testgen", flag.ContinueOnError)
	flags.SetOutput(stderr)
	providerMode := flags.String("provider", "static", "test provider: static, llm, or auto")
	providerCheck := flags.Bool("provider-check", false, "diagnose provider configuration and exit")
	flags.Usage = func() {
		fmt.Fprintf(stderr, "Usage: testgen [flags] <source_file> [output_file]\n\n")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		return 2
	}

	if *providerCheck {
		return runProviderCheck(*providerMode, stdout, stderr)
	}

	if flags.NArg() < 1 {
		flags.Usage()
		return 1
	}

	srcFile := flags.Arg(0)
	outputFile := generator.TestFileName(srcFile)
	if flags.NArg() > 1 {
		outputFile = flags.Arg(1)
	}

	provider, err := generator.NewTestProvider(*providerMode)
	if err != nil {
		fmt.Fprintf(stderr, "Provider error: %v\n", err)
		return 1
	}

	code, err := generator.GenerateTestsWithProviderOptions(context.Background(), srcFile, provider, generator.GenerateTestsOptions{
		TestFile: outputFile,
	})
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		if provider.Name() == "llm-command" {
			fmt.Fprintf(stderr, "Provider diagnostic: run `testgen -provider %s -provider-check` to verify %s and command availability.\n", *providerMode, generator.EnvLLMProviderCommand)
		}
		return 1
	}
	code, err = generator.AvoidDuplicateGoTestNames(srcFile, outputFile, code)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}

	if err := os.WriteFile(outputFile, []byte(code), 0644); err != nil {
		fmt.Fprintf(stderr, "Write error: %v\n", err)
		return 1
	}

	action := generator.GeneratedTestsAction(code, srcFile)
	fmt.Fprintf(stdout, "Generated: %s (provider=%s action=%s)\n", outputFile, provider.Name(), action)
	return 0
}

func runProviderCheck(mode string, stdout, stderr io.Writer) int {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" {
		mode = "static"
	}

	switch mode {
	case "static":
		fmt.Fprintln(stdout, "provider=static")
		fmt.Fprintln(stdout, "status=ok")
		fmt.Fprintln(stdout, "detail=static provider does not require external configuration")
		return 0
	case "auto", "llm":
		return checkLLMProvider(mode, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "Provider error: unsupported test provider %q (supported: static, llm, auto)\n", mode)
		return 1
	}
}

func checkLLMProvider(mode string, stdout, stderr io.Writer) int {
	command := strings.TrimSpace(os.Getenv(generator.EnvLLMProviderCommand))
	fmt.Fprintf(stdout, "provider=%s\n", mode)
	fmt.Fprintf(stdout, "%s=%s\n", generator.EnvLLMProviderCommand, printableProviderCommand(command))

	if command == "" {
		if mode == "auto" {
			fmt.Fprintln(stdout, "status=ok")
			fmt.Fprintln(stdout, "detail=auto provider will fall back to static because TESTLOOP_LLM_PROVIDER_CMD is not set")
			return 0
		}
		fmt.Fprintln(stderr, "status=error")
		fmt.Fprintf(stderr, "detail=provider llm requires %s\n", generator.EnvLLMProviderCommand)
		return 1
	}

	parts := strings.Fields(command)
	if len(parts) == 0 {
		fmt.Fprintln(stderr, "status=error")
		fmt.Fprintf(stderr, "detail=%s is empty after parsing\n", generator.EnvLLMProviderCommand)
		return 1
	}

	resolved, err := exec.LookPath(parts[0])
	if err != nil {
		fmt.Fprintln(stderr, "status=error")
		fmt.Fprintf(stderr, "detail=provider command executable not found: %s\n", parts[0])
		fmt.Fprintln(stderr, "hint=check PATH or use an absolute path in TESTLOOP_LLM_PROVIDER_CMD")
		return 1
	}

	fmt.Fprintln(stdout, "status=ok")
	fmt.Fprintf(stdout, "command=%s\n", command)
	fmt.Fprintf(stdout, "executable=%s\n", filepath.ToSlash(resolved))
	if mode == "auto" {
		fmt.Fprintln(stdout, "detail=auto provider will use llm-command")
	} else {
		fmt.Fprintln(stdout, "detail=llm provider command is configured")
	}
	return 0
}

func printableProviderCommand(command string) string {
	if command == "" {
		return "<unset>"
	}
	return command
}
