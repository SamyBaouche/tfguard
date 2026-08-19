package app

import (
	"context"

	"github.com/SamyBaouche/tfguard/internal/explain"
)

// ExplainInput builds a compact LLM context from a scan report.
func ExplainInput(rep Report) explain.Input {
	in := explain.Input{
		MaxRisk:        rep.MaxRisk.String(),
		ChangeCount:    len(rep.Changes),
		Creates:        rep.Summary.Creates,
		Updates:        rep.Summary.Updates,
		Replaces:       rep.Summary.Replaces,
		Deletes:        rep.Summary.Deletes,
		CostDeltaUSD:   rep.Cost.MonthlyDeltaUSD,
		Changes:        make([]explain.ChangeLine, 0, len(rep.Changes)),
		Findings:       make([]explain.FindingLine, 0, len(rep.Policy.Findings)),
		TopCostDrivers: make([]explain.CostDriver, 0, len(rep.Cost.Drivers)),
	}
	for _, c := range rep.Changes {
		in.Changes = append(in.Changes, explain.ChangeLine{
			Address: c.Address,
			Type:    c.Type,
			Action:  string(c.Action),
			Risk:    c.Level.String(),
		})
	}
	for _, f := range rep.Policy.Findings {
		in.Findings = append(in.Findings, explain.FindingLine{
			ID:       f.ID,
			Severity: string(f.Severity),
			Resource: f.Resource,
			Title:    f.Title,
		})
	}
	for _, d := range rep.Cost.Drivers {
		in.TopCostDrivers = append(in.TopCostDrivers, explain.CostDriver{
			Address: d.Address,
			Delta:   d.DeltaUSD,
		})
	}
	return in
}

// AttachExplanation runs the LLM explainer and stores the result on the report.
func AttachExplanation(ctx context.Context, rep *Report, opts explain.Options) error {
	res, err := explain.Explain(ctx, ExplainInput(*rep), opts)
	if err != nil {
		return err
	}
	rep.Explain = &ExplainResult{
		Explanation: res.Explanation,
		Cached:      res.Cached,
		Skipped:     res.Skipped,
		Warning:     res.Warning,
	}
	return nil
}
