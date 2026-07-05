package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/sleticalboy/testloop-mcp/internal/generator"
)

func main() {
	providerMode := flag.String("provider", "static", "test provider: static, llm, or auto")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: testgen [flags] <source_file> [output_file]\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	srcFile := flag.Arg(0)
	outputFile := generator.TestFileName(srcFile)
	if flag.NArg() > 1 {
		outputFile = flag.Arg(1)
	}

	provider, err := generator.NewTestProvider(*providerMode)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Provider error: %v\n", err)
		os.Exit(1)
	}

	code, err := generator.GenerateTestsWithProvider(context.Background(), srcFile, provider)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(outputFile, []byte(code), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Write error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated: %s (provider=%s)\n", outputFile, provider.Name())
}
