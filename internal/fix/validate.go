// Package fix repairs invalid Terraform configurations with an LLM, using
// `terraform validate` as the source of truth.
package fix

import (
	"encoding/json"
	"fmt"
	"sort"
)

// Severity values reported by `terraform validate -json`.
const (
	SeverityError   = "error"
	SeverityWarning = "warning"
)

// ValidateResult is the output of `terraform validate -json`.
type ValidateResult struct {
	FormatVersion string       `json:"format_version"`
	Valid         bool         `json:"valid"`
	ErrorCount    int          `json:"error_count"`
	WarningCount  int          `json:"warning_count"`
	Diagnostics   []Diagnostic `json:"diagnostics"`
}

// Diagnostic is one error or warning reported by Terraform.
type Diagnostic struct {
	Severity string   `json:"severity"`
	Summary  string   `json:"summary"`
	Detail   string   `json:"detail"`
	Range    *Range   `json:"range,omitempty"`   // nil when the problem is not tied to a file
	Snippet  *Snippet `json:"snippet,omitempty"` // nil when Range is nil
}

// Range locates a diagnostic in a source file.
type Range struct {
	Filename string `json:"filename"`
	Start    Pos    `json:"start"`
	End      Pos    `json:"end"`
}

// Pos is a 1-based line and column.
type Pos struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// Snippet is the source code around a diagnostic.
type Snippet struct {
	Context   string `json:"context"` // enclosing block, e.g. resource "aws_s3_bucket" "logs"
	Code      string `json:"code"`
	StartLine int    `json:"start_line"`
}

// ParseValidate decodes the JSON printed by `terraform validate -json`.
func ParseValidate(data []byte) (ValidateResult, error) {
	var res ValidateResult
	if err := json.Unmarshal(data, &res); err != nil {
		return ValidateResult{}, fmt.Errorf("parse terraform validate output: %w", err)
	}
	if res.FormatVersion == "" {
		return ValidateResult{}, fmt.Errorf("parse terraform validate output: missing format_version")
	}
	return res, nil
}

// Errors returns only the error diagnostics.
func (r ValidateResult) Errors() []Diagnostic {
	return r.withSeverity(SeverityError)
}

// Warnings returns only the warning diagnostics.
func (r ValidateResult) Warnings() []Diagnostic {
	return r.withSeverity(SeverityWarning)
}

func (r ValidateResult) withSeverity(severity string) []Diagnostic {
	var out []Diagnostic
	for _, d := range r.Diagnostics {
		if d.Severity == severity {
			out = append(out, d)
		}
	}
	return out
}

// FilesWithErrors returns the sorted, de-duplicated files that contain errors.
func (r ValidateResult) FilesWithErrors() []string {
	seen := map[string]bool{}
	var files []string
	for _, d := range r.Errors() {
		if d.Range == nil || d.Range.Filename == "" || seen[d.Range.Filename] {
			continue
		}
		seen[d.Range.Filename] = true
		files = append(files, d.Range.Filename)
	}
	sort.Strings(files)
	return files
}

// Location returns "file:line", or "" when the diagnostic has no position.
func (d Diagnostic) Location() string {
	if d.Range == nil {
		return ""
	}
	return fmt.Sprintf("%s:%d", d.Range.Filename, d.Range.Start.Line)
}
