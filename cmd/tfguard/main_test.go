package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := execute(&out, &errBuf, []string{"version"})
	if code != 0 {
		t.Fatalf("exit = %d stderr=%q", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "tfguard") {
		t.Fatalf("version output = %q", out.String())
	}
}

func TestScanMissingPlan(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := execute(&out, &errBuf, []string{"scan"})
	if code == 0 {
		t.Fatalf("expected non-zero exit, stdout=%q stderr=%q", out.String(), errBuf.String())
	}
}

func TestScanFailOn(t *testing.T) {
	plan := filepath.Join("..", "..", "testdata", "plan_mixed.json")
	var out, errBuf bytes.Buffer
	code := execute(&out, &errBuf, []string{
		"scan",
		"--plan", plan,
		"--fail-on", "DANGER",
		"--skip-checkov",
		"--skip-tfsec",
	})
	if code != 1 {
		t.Fatalf("exit = %d, want 1; stderr=%q stdout=%q", code, errBuf.String(), out.String())
	}
	if !strings.Contains(out.String(), "Scan report") {
		t.Fatalf("expected report on stdout, got %q", out.String())
	}
}

func TestScanMaxCostIncrease(t *testing.T) {
	plan := filepath.Join("..", "..", "testdata", "plan_mixed.json")
	var out, errBuf bytes.Buffer
	code := execute(&out, &errBuf, []string{
		"scan",
		"--plan", plan,
		"--max-cost-increase", "1",
		"--skip-checkov",
		"--skip-tfsec",
		"--no-banner",
	})
	if code != 1 {
		t.Fatalf("exit = %d, want 1; stderr=%q stdout=%q", code, errBuf.String(), out.String())
	}
	if !strings.Contains(errBuf.String(), "max-cost-increase") {
		t.Fatalf("expected cost gate message, stderr=%q", errBuf.String())
	}
	if !strings.Contains(out.String(), "Cost estimate") {
		t.Fatalf("expected cost section, got %q", out.String())
	}
}

func TestScanMaxCostIncreaseInvalid(t *testing.T) {
	plan := filepath.Join("..", "..", "testdata", "plan_mixed.json")
	var out, errBuf bytes.Buffer
	code := execute(&out, &errBuf, []string{
		"scan",
		"--plan", plan,
		"--max-cost-increase", "nope",
	})
	if code != 2 {
		t.Fatalf("exit = %d, want 2; stderr=%q", code, errBuf.String())
	}
}

func TestUnknownCommand(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := execute(&out, &errBuf, []string{"nope"})
	if code == 0 {
		t.Fatalf("expected non-zero exit for unknown command")
	}
}

func TestEmptyArgs(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := execute(&out, &errBuf, nil)
	if code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
}
