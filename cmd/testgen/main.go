package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/sleticalboy/testloop-mcp/internal/generator"
)

func main() {
	os.Exit(runTestgen(os.Args[1:], os.Stdout, os.Stderr))
}

func runTestgen(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("testgen", flag.ContinueOnError)
	flags.SetOutput(stderr)
	providerMode := flags.String("provider", "static", "test provider: static, llm, or auto")
	flags.Usage = func() {
		fmt.Fprintf(stderr, "Usage: testgen [flags] <source_file> [output_file]\n\n")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		return 2
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

	code, err := generator.GenerateTestsWithProvider(context.Background(), srcFile, provider)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}

	if err := os.WriteFile(outputFile, []byte(code), 0644); err != nil {
		fmt.Fprintf(stderr, "Write error: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "Generated: %s (provider=%s)\n", outputFile, provider.Name())
	return 0
}
