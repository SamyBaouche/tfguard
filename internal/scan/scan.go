// Package scan orchestrates a tfplan review and builds the final Report for the CLI.
//
// It is the "pipeline coordinator" layer:
// 1. parse + summarize plan mutations
// 2. classify change risk (SAFE..CRITICAL)
// 3. run policy scanners (OPA, optional Checkov/tfsec)
// 4. estimate a static AWS monthly cost delta
// 5. compute the embedded ML high-risk probability
package scan

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/SamyBaouche/tfguard/internal/cost"
	"github.com/SamyBaouche/tfguard/internal/explain"
	"github.com/SamyBaouche/tfguard/internal/mlscore"
	"github.com/SamyBaouche/tfguard/internal/policy"
	"github.com/SamyBaouche/tfguard/internal/risk"
	"github.com/SamyBaouche/tfguard/internal/tfplan"
)

// Progress receives scan phase updates for animated CLI feedback.
// When nil, the pipeline uses a no-op implementation.
type Progress interface {
	Start(message string)
	Done(detail string)
	Fail(detail string)
}

type nopProgress struct{}

func (nopProgress) Start(string) {}
func (nopProgress) Done(string)  {}
func (nopProgress) Fail(string)  {}

// ChangeRisk is one mutating change with its classified risk level.
type ChangeRisk struct {
	Address string
	Type    string
	Action  tfplan.Action
	Level   risk.Level
}

// Report is the full scan result rendered by the CLI.
type Report struct {
	PlanPath string
	Summary  tfplan.Summary
	Changes  []ChangeRisk
	MaxRisk  risk.Level // highest Level among Changes
	Policy   policy.Result
	Cost     cost.Estimate
	Explain  *ExplainResult
	ML       mlscore.Score
}

// ExplainResult holds the optional LLM explanation attached after Run.
type ExplainResult struct {
	Explanation *explain.Explanation
	Cached      bool
	Skipped     bool
	Warning     string
}

// Options controls Run (plan path and which scanners to skip).
type Options struct {
	PlanPath     string
	TerraformDir string
	SkipCheckov  bool
	SkipTfsec    bool
	SkipOPA      bool
	SkipCost     bool
	Progress     Progress // optional; animated CLI steps when set
}

// Run parses the terraform plan, classifies each change, runs policy scanners,
// estimates cost delta, and computes the ML score.
func Run(ctx context.Context, opts Options) (Report, error) {
	prog := opts.Progress
	if prog == nil {
		prog = nopProgress{}
	}

	prog.Start("Parsing terraform plan")
	plan, err := tfplan.ParseFile(opts.PlanPath)
	if err != nil {
		prog.Fail("")
		return Report{}, err
	}
	prog.Done(filepath.Base(opts.PlanPath))

	prog.Start("Classifying change risk")
	summary := tfplan.Summarize(plan)
	changes := make([]ChangeRisk, 0, len(summary.Changes))
	maxRisk := risk.SAFE

	for _, rc := range summary.Changes {
		action := rc.Change.Action()
		level := risk.Classify(action, rc.Type)
		changes = append(changes, ChangeRisk{
			Address: rc.Address,
			Type:    rc.Type,
			Action:  action,
			Level:   level,
		})
		if level > maxRisk {
			maxRisk = level
		}
	}
	prog.Done(fmt.Sprintf("%d changes · max %s", len(changes), maxRisk.String()))

	prog.Start("Running policy scanners")
	pol, err := policy.Scan(ctx, plan, policy.ScanOptions{
		TerraformDir: opts.TerraformDir,
		SkipCheckov:  opts.SkipCheckov,
		SkipTfsec:    opts.SkipTfsec,
		SkipOPA:      opts.SkipOPA,
	})
	if err != nil {
		prog.Fail("")
		return Report{}, err
	}
	prog.Done(fmt.Sprintf("%d findings", len(pol.Findings)))

	var costEst cost.Estimate
	if !opts.SkipCost {
		prog.Start("Estimating cost delta")
		costEst = cost.EstimateChanges(summary.Changes)
		prog.Done(fmt.Sprintf("%+.2f USD/mo · %d priced", costEst.MonthlyDeltaUSD, costEst.Priced))
	}

	rep := Report{
		PlanPath: opts.PlanPath,
		Summary:  summary,
		Changes:  changes,
		MaxRisk:  maxRisk,
		Policy:   pol,
		Cost:     costEst,
	}
	rep.ML = mlscore.Predict(MLFeatures(rep))
	return rep, nil
}

// ParseFailOn parses SAFE|CAUTION|DANGER|CRITICAL for the CLI flag.
// An empty string disables fail-on (enabled=false).
func ParseFailOn(s string) (level risk.Level, enabled bool, err error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return risk.SAFE, false, nil
	}
	switch strings.ToUpper(s) {
	case "SAFE":
		return risk.SAFE, true, nil
	case "CAUTION":
		return risk.CAUTION, true, nil
	case "DANGER":
		return risk.DANGER, true, nil
	case "CRITICAL":
		return risk.CRITICAL, true, nil
	default:
		return 0, false, fmt.Errorf("invalid --fail-on %q (want SAFE, CAUTION, DANGER, or CRITICAL)", s)
	}
}

// FindingLevel maps policy Severity onto risk.Level so --fail-on can
// consider both change risk and scanner findings on one scale.
func FindingLevel(sev policy.Severity) risk.Level {
	switch sev {
	case policy.SeverityCritical:
		return risk.CRITICAL
	case policy.SeverityHigh:
		return risk.DANGER
	case policy.SeverityMedium:
		return risk.CAUTION
	default:
		return risk.SAFE
	}
}

// ShouldFail reports whether CI should exit non-zero.
// Compares MaxRisk and each finding against threshold when enabled.
func ShouldFail(rep Report, threshold risk.Level, enabled bool) bool {
	if !enabled {
		return false
	}
	if rep.MaxRisk >= threshold {
		return true
	}
	for _, f := range rep.Policy.Findings {
		if FindingLevel(f.Severity) >= threshold {
			return true
		}
	}
	return false
}

// ParseMaxCostIncrease parses a USD/month ceiling for --max-cost-increase.
// An empty string disables the gate (enabled=false).
func ParseMaxCostIncrease(s string) (limit float64, enabled bool, err error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false, nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false, fmt.Errorf("invalid --max-cost-increase %q (want a number, e.g. 50)", s)
	}
	return v, true, nil
}

// CostExceeded reports whether the monthly cost delta is above the configured ceiling.
func CostExceeded(rep Report, limit float64, enabled bool) bool {
	if !enabled {
		return false
	}
	return rep.Cost.MonthlyDeltaUSD > limit
}
