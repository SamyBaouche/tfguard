package scan

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/SamyBaouche/tfguard/internal/cost"
	"github.com/SamyBaouche/tfguard/internal/policy"
	"github.com/SamyBaouche/tfguard/internal/risk"
)

func TestParseFailOn(t *testing.T) {
	t.Parallel()
	_, enabled, err := ParseFailOn("")
	if err != nil || enabled {
		t.Fatal("empty fail-on should be disabled")
	}
	got, enabled, err := ParseFailOn("danger")
	if err != nil || !enabled || got != risk.DANGER {
		t.Fatalf("got %v enabled=%v err=%v", got, enabled, err)
	}
	if _, _, err := ParseFailOn("nope"); err == nil {
		t.Fatal("expected error")
	}
}

func TestShouldFail(t *testing.T) {
	t.Parallel()
	rep := Report{MaxRisk: risk.DANGER}
	if ShouldFail(rep, risk.CRITICAL, true) || !ShouldFail(rep, risk.DANGER, true) {
		t.Fatal("unexpected ShouldFail for MaxRisk")
	}
	rep = Report{
		MaxRisk: risk.SAFE,
		Policy:  policy.Result{Findings: []policy.Finding{{Severity: policy.SeverityCritical}}},
	}
	if !ShouldFail(rep, risk.CRITICAL, true) {
		t.Fatal("CRITICAL finding should fail")
	}
}

func TestParseMaxCostIncrease(t *testing.T) {
	t.Parallel()
	_, enabled, err := ParseMaxCostIncrease("")
	if err != nil || enabled {
		t.Fatal("empty should be disabled")
	}
	got, enabled, err := ParseMaxCostIncrease("50.5")
	if err != nil || !enabled || got != 50.5 {
		t.Fatalf("got %v enabled=%v err=%v", got, enabled, err)
	}
	if _, _, err := ParseMaxCostIncrease("nope"); err == nil {
		t.Fatal("expected error")
	}
}

func TestCostExceeded(t *testing.T) {
	t.Parallel()
	rep := Report{Cost: cost.Estimate{MonthlyDeltaUSD: 12.5}}
	if CostExceeded(rep, 20, true) || !CostExceeded(rep, 10, true) {
		t.Fatal("unexpected CostExceeded")
	}
	if CostExceeded(rep, 0, false) {
		t.Fatal("disabled gate should not trip")
	}
}

func TestRunMixedPlan(t *testing.T) {
	t.Parallel()
	rep, err := Run(context.Background(), Options{
		PlanPath:    filepath.Join("..", "..", "testdata", "plan_mixed.json"),
		SkipCheckov: true,
		SkipTfsec:   true,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if rep.Summary.Creates != 1 || rep.MaxRisk < risk.DANGER {
		t.Fatalf("unexpected report: %+v", rep)
	}
	// plan_mixed: t3.micro → t3.small ≈ +7.59 USD/mo
	if rep.Cost.Priced < 1 || rep.Cost.MonthlyDeltaUSD <= 0 {
		t.Fatalf("expected positive cost delta, got %+v", rep.Cost)
	}
}
