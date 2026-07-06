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

func TestJavaTestCommandPrefersGradleWrapper(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("wrapper command paths are unix-style")
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "build.gradle"), []byte("plugins {}\n"), 0o644); err != nil {
		t.Fatalf("write build.gradle: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "gradlew"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write gradlew: %v", err)
	}
	testFile := filepath.Join(dir, "src", "test", "CalculatorTest.java")
	if err := os.MkdirAll(filepath.Dir(testFile), 0o755); err != nil {
		t.Fatalf("mkdir test dir: %v", err)
	}
	if err := os.WriteFile(testFile, []byte("class CalculatorTest {}\n"), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	cmd := javaTestCommand(context.Background(), testFile, true)
	want := []string{"./gradlew", "test", "jacocoTestReport"}
	if !equalStrings(cmd.Args, want) {
		t.Fatalf("args = %v, want %v", cmd.Args, want)
	}
	if cmd.Dir != dir {
		t.Fatalf("dir = %s, want %s", cmd.Dir, dir)
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

	if percent := collectCoveragePercent(context.Background(), "junit", dir, 12.5); percent != 50 {
		t.Fatalf("collectCoveragePercent junit = %.1f, want 50.0", percent)
	}
	if percent := collectCoveragePercent(context.Background(), "go-test", dir, 12.5); percent != 12.5 {
		t.Fatalf("collectCoveragePercent passthrough = %.1f, want 12.5", percent)
	}
}

func TestNormalizeGoTestPathUsesContainingDirForGoFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "calc_test.go")
	if err := os.WriteFile(file, []byte("package calc\n"), 0o644); err != nil {
		t.Fatalf("write go file: %v", err)
	}

	if got := normalizeGoTestPath(file); got != dir {
		t.Fatalf("normalizeGoTestPath(%q) = %q, want %q", file, got, dir)
	}
	if got := normalizeGoTestPath(dir); got != dir {
		t.Fatalf("normalizeGoTestPath(%q) = %q, want %q", dir, got, dir)
	}
}

func TestFindProjectRootWalksParents(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "pkg", "calc")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte("[package]\n"), 0o644); err != nil {
		t.Fatalf("write Cargo.toml: %v", err)
	}
	source := filepath.Join(nested, "calc.rs")
	if err := os.WriteFile(source, []byte("fn add() {}\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	if got := findProjectRoot(source, "Cargo.toml"); got != dir {
		t.Fatalf("findProjectRoot = %q, want %q", got, dir)
	}
}

func TestHandleRunTestsValidatesInput(t *testing.T) {
	if _, _, err := HandleRunTests(context.Background(), nil, runTestsInput{}); err == nil {
		t.Fatal("expected missing path error")
	}
	if _, _, err := HandleRunTests(context.Background(), nil, runTestsInput{Path: ".", Framework: "unknown"}); err == nil {
		t.Fatal("expected unsupported framework error")
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
