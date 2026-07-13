package generator

import (
	"strings"
	"testing"
)

func TestJSExtractTSTypeDeclsKeepsSimpleGenericParams(t *testing.T) {
	decls := jsExtractTSTypeDecls(`export interface Box<T> {
  data: T
}

type Pair<T, U> = {
  first: T
  second: U
}

type User = {
  id: number
}
`)

	for name, want := range map[string]string{
		"Box<T>":    "{\n  data: T\n}",
		"Pair<T,U>": "{\n  first: T\n  second: U\n}",
		"User":      "{\n  id: number\n}",
	} {
		if got := decls[name]; got != want {
			t.Fatalf("decls[%q] = %q, want %q", name, got, want)
		}
	}
}

func TestJSExtractTSTypeDeclsKeepsInterfaceMethodsAsFunctionFields(t *testing.T) {
	decls := jsExtractTSTypeDecls(`export interface ILogger {
  info(message: string): void;
  warn(message: string, meta?: unknown): void;
  error(error: Error): Promise<void>;
}
`)

	got := decls["ILogger"]
	for _, want := range []string{
		"info(message: string): void",
		"warn(message: string, meta?: unknown): void",
		"error(error: Error): Promise<void>",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("ILogger decl missing %q: %q", want, got)
		}
	}
}
