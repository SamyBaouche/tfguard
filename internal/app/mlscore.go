package app

import (
	"github.com/SamyBaouche/tfguard/internal/mlscore"
	"github.com/SamyBaouche/tfguard/internal/policy"
	"github.com/SamyBaouche/tfguard/internal/risk"
	"github.com/SamyBaouche/tfguard/internal/tfplan"
)

// MLFeatures maps a scan report to model inputs.
func MLFeatures(rep Report) mlscore.Features {
	f := mlscore.Features{
		Creates:       rep.Summary.Creates,
		Updates:       rep.Summary.Updates,
		Replaces:      rep.Summary.Replaces,
		Deletes:       rep.Summary.Deletes,
		CostDeltaUSD:  rep.Cost.MonthlyDeltaUSD,
		ChangeCount:   len(rep.Changes),
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

func isMutation(a tfplan.Action) bool {
	switch a {
	case tfplan.ActionUpdate, tfplan.ActionReplace, tfplan.ActionDelete:
		return true
	default:
		return false
	}
}
