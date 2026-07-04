package tools

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestJavaTestCommandUsesCoverageArgs(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pom.xml"), []byte("<project/>"), 0o644); err != nil {
		t.Fatalf("write pom.xml: %v", err)
	}

	cmd := javaTestCommand(context.Background(), dir, true)
	if filepath.Base(cmd.Path) != "mvn" {
		t.Fatalf("command = %s, want mvn", cmd.Path)
	}
	want := []string{"mvn", "test", "jacoco:report"}
	if !equalStrings(cmd.Args, want) {
		t.Fatalf("args = %v, want %v", cmd.Args, want)
	}
	if cmd.Dir != dir {
		t.Fatalf("dir = %s, want %s", cmd.Dir, dir)
	}
}

func TestJavaTestCommandPrefersWrappers(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("wrapper command paths are unix-style")
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pom.xml"), []byte("<project/>"), 0o644); err != nil {
		t.Fatalf("write pom.xml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "mvnw"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write mvnw: %v", err)
	}

	cmd := javaTestCommand(context.Background(), dir, false)
	want := []string{"./mvnw", "test"}
	if !equalStrings(cmd.Args, want) {
		t.Fatalf("args = %v, want %v", cmd.Args, want)
	}
}

func TestCollectJavaCoveragePercentReadsMavenReport(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pom.xml"), []byte("<project/>"), 0o644); err != nil {
		t.Fatalf("write pom.xml: %v", err)
	}
	reportDir := filepath.Join(dir, "target", "site", "jacoco")
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		t.Fatalf("mkdir report dir: %v", err)
	}
	report := `<?xml version="1.0" encoding="UTF-8"?>
<report name="demo">
  <package name="com/example">
    <sourcefile name="Calculator.java">
      <line nr="10" mi="0" ci="1"/>
      <line nr="11" mi="1" ci="0"/>
      <counter type="LINE" missed="1" covered="1"/>
    </sourcefile>
  </package>
  <counter type="LINE" missed="1" covered="1"/>
</report>`
	if err := os.WriteFile(filepath.Join(reportDir, "jacoco.xml"), []byte(report), 0o644); err != nil {
		t.Fatalf("write jacoco.xml: %v", err)
	}

	percent, ok := collectJavaCoveragePercent(dir)
	if !ok {
		t.Fatal("expected Java coverage report to be parsed")
	}
	if percent != 50 {
		t.Fatalf("percent = %.1f, want 50.0", percent)
	}
}

func TestCoverageArgs(t *testing.T) {
	if got := javaMavenArgs(false); !equalStrings(got, []string{"test"}) {
		t.Fatalf("maven no coverage args = %v", got)
	}
	if got := javaMavenArgs(true); !equalStrings(got, []string{"test", "jacoco:report"}) {
		t.Fatalf("maven coverage args = %v", got)
	}
	if got := javaGradleArgs(false); !equalStrings(got, []string{"test"}) {
		t.Fatalf("gradle no coverage args = %v", got)
	}
	if got := javaGradleArgs(true); !equalStrings(got, []string{"test", "jacocoTestReport"}) {
		t.Fatalf("gradle coverage args = %v", got)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
