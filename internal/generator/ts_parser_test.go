package generator

import "testing"

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
