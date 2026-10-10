package fix

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Runner runs the Terraform commands the fixer needs. Terraform is the real
// implementation; tests can substitute a fake.
type Runner interface {
	Init(ctx context.Context, dir string) error
	Validate(ctx context.Context, dir string) (ValidateResult, error)
}

// ErrTerraformNotFound is returned when the terraform binary is not on PATH.
var ErrTerraformNotFound = errors.New("terraform not found on PATH (install: https://developer.hashicorp.com/terraform/install)")

// Terraform is a Runner backed by the terraform CLI.
type Terraform struct {
	bin string
}

// NewTerraform locates the terraform binary on PATH.
func NewTerraform() (*Terraform, error) {
	bin, err := exec.LookPath("terraform")
	if err != nil {
		return nil, ErrTerraformNotFound
	}
	return &Terraform{bin: bin}, nil
}

// Init runs `terraform init` without a backend, so no state or cloud
// credentials are needed. It fails when the configuration has syntax errors;
// Validate still reports those, so callers should not stop on an Init error.
func (t *Terraform) Init(ctx context.Context, dir string) error {
	stdout, stderr, err := t.run(ctx, dir, "init", "-backend=false", "-input=false", "-no-color")
	if ctxErr := ctx.Err(); ctxErr != nil {
		return fmt.Errorf("terraform init: %w", ctxErr)
	}
	if err != nil {
		return fmt.Errorf("terraform init: %s", firstNonEmpty(stderr, string(stdout), err.Error()))
	}
	return nil
}

// Validate runs `terraform validate -json`. Terraform exits with code 1 when the
// configuration is invalid but still prints a complete JSON report, so a
// non-zero exit is only an error when that report is missing.
func (t *Terraform) Validate(ctx context.Context, dir string) (ValidateResult, error) {
	stdout, stderr, runErr := t.run(ctx, dir, "validate", "-json", "-no-color")
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ValidateResult{}, fmt.Errorf("terraform validate: %w", ctxErr)
	}
	res, err := ParseValidate(stdout)
	if err != nil {
		if runErr != nil {
			return ValidateResult{}, fmt.Errorf("terraform validate: %s", firstNonEmpty(stderr, runErr.Error()))
		}
		return ValidateResult{}, err
	}
	return res, nil
}

func (t *Terraform) run(ctx context.Context, dir string, args ...string) (stdout []byte, stderr string, err error) {
	cmd := exec.CommandContext(ctx, t.bin, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "TF_IN_AUTOMATION=1")

	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	err = cmd.Run()
	return out.Bytes(), strings.TrimSpace(errOut.String()), err
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}
