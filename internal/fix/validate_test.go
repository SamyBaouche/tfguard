package fix

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func loadValidateFixture(t *testing.T, name string) ValidateResult {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "fix", name, "validate.json"))
	if err != nil {
		t.Fatal(err)
	}
	res, err := ParseValidate(data)
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return res
}

func TestParseValidateFixtures(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		valid        bool
		errors       int
		warnings     int
		files        []string
		firstSummary string
		firstLoc     string
	}{
		{"already_valid", true, 0, 0, nil, "", ""},
		{"deprecated_attribute", true, 0, 1, nil, "Argument is deprecated", "main.tf:14"},
		{"syntax_error", false, 6, 0, []string{"main.tf"}, "Invalid multi-line string", "main.tf:15"},
		{"unknown_attribute", false, 1, 0, []string{"main.tf"}, "Unsupported argument", "main.tf:16"},
		{"missing_required", false, 1, 0, []string{"main.tf"}, "Missing required argument", "main.tf:18"},
		{"wrong_type", false, 1, 0, []string{"main.tf"}, "Incorrect attribute value type", "main.tf:22"},
		{"undeclared_ref", false, 2, 0, []string{"main.tf"}, "Reference to undeclared input variable", "main.tf:15"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			res := loadValidateFixture(t, tc.name)

			if res.Valid != tc.valid {
				t.Errorf("Valid = %v, want %v", res.Valid, tc.valid)
			}
			if got := len(res.Errors()); got != tc.errors || res.ErrorCount != tc.errors {
				t.Errorf("errors: len=%d count=%d, want %d", got, res.ErrorCount, tc.errors)
			}
			if got := len(res.Warnings()); got != tc.warnings || res.WarningCount != tc.warnings {
				t.Errorf("warnings: len=%d count=%d, want %d", got, res.WarningCount, tc.warnings)
			}
			if got := res.FilesWithErrors(); !slices.Equal(got, tc.files) {
				t.Errorf("FilesWithErrors = %v, want %v", got, tc.files)
			}

			if tc.firstSummary == "" {
				if len(res.Diagnostics) != 0 {
					t.Errorf("expected no diagnostics, got %d", len(res.Diagnostics))
				}
				return
			}
			first := res.Diagnostics[0]
			if first.Summary != tc.firstSummary || first.Location() != tc.firstLoc {
				t.Errorf("first = %q @ %q, want %q @ %q", first.Summary, first.Location(), tc.firstSummary, tc.firstLoc)
			}
			if first.Snippet == nil || first.Snippet.Context == "" {
				t.Errorf("first diagnostic should carry a snippet with context")
			}
		})
	}
}

func TestParseValidateRejectsInvalidInput(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{"not json", `Error: something went wrong`, "parse terraform validate output"},
		{"empty object", `{}`, "missing format_version"},
		{"empty input", ``, "parse terraform validate output"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := ParseValidate([]byte(tc.input))
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("err = %v, want containing %q", err, tc.wantErr)
			}
		})
	}
}

func TestFilesWithErrorsSkipsWarningsAndUnlocatedErrors(t *testing.T) {
	t.Parallel()
	res := ValidateResult{
		FormatVersion: "1.0",
		Diagnostics: []Diagnostic{
			{Severity: SeverityError, Range: &Range{Filename: "variables.tf"}},
			{Severity: SeverityError, Range: &Range{Filename: "main.tf"}},
			{Severity: SeverityError, Range: &Range{Filename: "main.tf"}},
			{Severity: SeverityWarning, Range: &Range{Filename: "outputs.tf"}},
			{Severity: SeverityError, Summary: "provider not found"},
		},
	}
	want := []string{"main.tf", "variables.tf"}
	if got := res.FilesWithErrors(); !slices.Equal(got, want) {
		t.Fatalf("FilesWithErrors = %v, want %v", got, want)
	}
}

func TestLocationWithoutRange(t *testing.T) {
	t.Parallel()
	if got := (Diagnostic{Summary: "provider not found"}).Location(); got != "" {
		t.Fatalf("Location = %q, want empty", got)
	}
}
