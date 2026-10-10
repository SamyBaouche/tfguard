package fix

import (
	"context"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var _ Runner = (*Terraform)(nil)

var integration = flag.Bool("integration", false, "run tests that call the real terraform CLI (slow)")

// newTerraformOrSkip returns the real runner. Integration tests are opt-in
// (`make test-integration`) because the first `terraform init` takes ~30s.
func newTerraformOrSkip(t *testing.T) *Terraform {
	t.Helper()
	if !*integration {
		t.Skip("terraform integration test; run with -integration")
	}
	tf, err := NewTerraform()
	if errors.Is(err, ErrTerraformNotFound) {
		t.Skip("terraform not on PATH")
	}
	if err != nil {
		t.Fatal(err)
	}
	return tf
}

// copyCase copies the .tf files of testdata/fix/<name> into a temp dir so
// terraform never writes .terraform/ or lock files into testdata.
func copyCase(t *testing.T, name string) string {
	t.Helper()
	src := filepath.Join("..", "..", "testdata", "fix", name)
	files, err := filepath.Glob(filepath.Join(src, "*.tf"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no .tf files in %s (err=%v)", src, err)
	}
	dst := t.TempDir()
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dst, filepath.Base(f)), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dst
}

func testContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	t.Cleanup(cancel)
	return ctx
}

// Integration tests run sequentially: concurrent `terraform init` calls can
// race on a shared provider plugin cache.

func TestTerraformValidConfig(t *testing.T) {
	tf := newTerraformOrSkip(t)
	ctx := testContext(t)
	dir := copyCase(t, "already_valid")

	if err := tf.Init(ctx, dir); err != nil {
		t.Fatalf("Init: %v", err)
	}
	res, err := tf.Validate(ctx, dir)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if !res.Valid || len(res.Errors()) != 0 {
		t.Fatalf("expected valid config, got %+v", res)
	}
}

func TestTerraformSyntaxErrorFailsInitButValidateReports(t *testing.T) {
	tf := newTerraformOrSkip(t)
	ctx := testContext(t)
	dir := copyCase(t, "syntax_error")

	err := tf.Init(ctx, dir)
	if err == nil || !strings.Contains(err.Error(), "Unterminated template string") {
		t.Fatalf("Init should fail with the syntax error, got %v", err)
	}

	res, err := tf.Validate(ctx, dir)
	if err != nil {
		t.Fatalf("Validate must still return diagnostics: %v", err)
	}
	want := loadValidateFixture(t, "syntax_error")
	if res.Valid || res.ErrorCount != want.ErrorCount {
		t.Fatalf("Validate = valid:%v errors:%d, fixture has %d errors", res.Valid, res.ErrorCount, want.ErrorCount)
	}
}

func TestTerraformSchemaErrorNeedsInit(t *testing.T) {
	tf := newTerraformOrSkip(t)
	ctx := testContext(t)
	dir := copyCase(t, "unknown_attribute")

	if err := tf.Init(ctx, dir); err != nil {
		t.Fatalf("Init: %v", err)
	}
	res, err := tf.Validate(ctx, dir)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	errs := res.Errors()
	if res.Valid || len(errs) != 1 || errs[0].Summary != "Unsupported argument" {
		t.Fatalf("expected one Unsupported argument error, got %+v", errs)
	}
}

func TestTerraformMissingBinaryReturnsError(t *testing.T) {
	t.Parallel()
	tf := &Terraform{bin: filepath.Join(t.TempDir(), "no-such-terraform")}
	ctx := context.Background()

	if err := tf.Init(ctx, t.TempDir()); err == nil || !strings.Contains(err.Error(), "terraform init") {
		t.Fatalf("Init err = %v", err)
	}
	if _, err := tf.Validate(ctx, t.TempDir()); err == nil || !strings.Contains(err.Error(), "terraform validate") {
		t.Fatalf("Validate err = %v", err)
	}
}

func TestTerraformCanceledContext(t *testing.T) {
	t.Parallel()
	tf := &Terraform{bin: filepath.Join(t.TempDir(), "no-such-terraform")}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := tf.Init(ctx, t.TempDir()); !errors.Is(err, context.Canceled) {
		t.Fatalf("Init err = %v, want context.Canceled", err)
	}
	if _, err := tf.Validate(ctx, t.TempDir()); !errors.Is(err, context.Canceled) {
		t.Fatalf("Validate err = %v, want context.Canceled", err)
	}
}

func TestFirstNonEmpty(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in   []string
		want string
	}{
		{[]string{"", "  ", "b", "c"}, "b"},
		{[]string{"  a  "}, "a"},
		{[]string{"", ""}, ""},
		{nil, ""},
	}
	for _, tc := range tests {
		if got := firstNonEmpty(tc.in...); got != tc.want {
			t.Errorf("firstNonEmpty(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
