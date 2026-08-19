package scan

import (
	"github.com/SamyBaouche/tfguard/internal/mlscore"
	"github.com/SamyBaouche/tfguard/internal/policy"
	"github.com/SamyBaouche/tfguard/internal/risk"
	"github.com/SamyBaouche/tfguard/internal/tfplan"
)

// MLFeatures converts the scan Report into the numeric feature vector
// expected by the embedded logistic regression model.
func MLFeatures(rep Report) mlscore.Features {
	f := mlscore.Features{
		Creates:        rep.Summary.Creates,
		Updates:        rep.Summary.Updates,
		Replaces:       rep.Summary.Replaces,
		Deletes:        rep.Summary.Deletes,
		CostDeltaUSD:   rep.Cost.MonthlyDeltaUSD,
		ChangeCount:    len(rep.Changes),
		MaxActionLevel: maxActionLevel(rep.Changes),
	}
	for _, c := range rep.Changes {
		if risk.IsStateful(c.Type) && isMutation(c.Action) {
			f.StatefulMutations++
		}
	}
	for _, finding := range rep.Policy.Findings {
		switch finding.Severity {
		case policy.SeverityCritical:
			f.CriticalFindings++
		case policy.SeverityHigh:
			f.HighFindings++
		}
	}
	return f
}

// maxActionLevel summarizes the most "invasive" Terraform action across
// changes. It is intentionally coarse to keep the feature space small.
func maxActionLevel(changes []ChangeRisk) int {
	level := 0
	for _, c := range changes {
		switch c.Action {
		case tfplan.ActionUpdate:
			if level < 1 {
				level = 1
			}
		case tfplan.ActionReplace, tfplan.ActionDelete:
			if level < 2 {
				level = 2
			}
		}
	}
	return level
}

// isMutation returns true for changes that alter infrastructure state
// (create/update/replace/delete), not for reads/no-ops.
func isMutation(a tfplan.Action) bool {
	switch a {
	case tfplan.ActionUpdate, tfplan.ActionReplace, tfplan.ActionDelete:
		return true
	default:
		return false
	}
}
