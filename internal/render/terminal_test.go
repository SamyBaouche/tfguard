package render

import (
	"bytes"
	"strings"
	"testing"

	"github.com/SamyBaouche/tfguard/internal/cost"
	"github.com/SamyBaouche/tfguard/internal/policy"
	"github.com/SamyBaouche/tfguard/internal/risk"
	"github.com/SamyBaouche/tfguard/internal/scan"
	"github.com/SamyBaouche/tfguard/internal/tfplan"
)

func TestTerminal(t *testing.T) {
	t.Parallel()
	rep := scan.Report{
		PlanPath: "plan.json",
		MaxRisk:  risk.CRITICAL,
		Summary:  tfplan.Summary{Deletes: 1},
		Changes: []scan.ChangeRisk{{
			Address: "aws_db_instance.main",
			Type:    "aws_db_instance",
			Action:  tfplan.ActionDelete,
			Level:   risk.CRITICAL,
		}},
		Policy: policy.Result{
			Findings: []policy.Finding{{
				Severity: policy.SeverityHigh,
				Source:   policy.SourceOPA,
				ID:       "TFGUARD-RDS-001",
				Resource: "aws_db_instance.main",
				Title:    "unencrypted",
			}},
		},
		Cost: cost.Estimate{
			MonthlyDeltaUSD: 7.59,
			Priced:          1,
			Skipped:         0,
			Drivers: []cost.Driver{{
				Address:   "aws_instance.web",
				Type:      "aws_instance",
				BeforeUSD: 7.59,
				AfterUSD:  15.18,
				DeltaUSD:  7.59,
			}},
		},
	}
	var buf bytes.Buffer
	if err := Terminal(&buf, rep); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{
		"Scan report", "CRITICAL", "TFGUARD-RDS-001", "highest",
		"Cost estimate", "Top cost drivers", "aws_instance.web", "+7.59",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q\n%s", want, out)
		}
	}
}
