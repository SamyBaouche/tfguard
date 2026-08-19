// Package render formats a scan.Report for the terminal.
package render

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/SamyBaouche/tfguard/internal/explain"
	"github.com/SamyBaouche/tfguard/internal/scan"
	"github.com/SamyBaouche/tfguard/internal/ui"
)

// Terminal writes a styled scan report to w.
func Terminal(w io.Writer, rep scan.Report) error {
	style := ui.NewStyle(w)

	fmt.Fprintln(w)
	ui.BoxTitle(w, style, "Scan report")
	ui.BoxLine(w, style, style.Dim("plan")+"   "+rep.PlanPath)
	ui.BoxLine(w, style, style.Dim("risk")+"   "+style.Risk(rep.MaxRisk.String())+style.Dim("  (highest)"))
	mlPct := fmt.Sprintf("%.0f%%", rep.ML.Probability*100)
	if rep.ML.HighRisk {
		mlPct = style.Red(mlPct)
	} else {
		mlPct = style.Green(mlPct)
	}
	ui.BoxLine(w, style, style.Dim("ml")+"     "+mlPct+style.Dim(" high-risk probability"))
	ui.BoxEnd(w, style)
	fmt.Fprintln(w)

	writeSummary(w, style, rep)
	fmt.Fprintln(w)
	writeCost(w, style, rep)
	fmt.Fprintln(w)
	if err := writeChanges(w, style, rep); err != nil {
		return err
	}
	fmt.Fprintln(w)
	if err := writeFindings(w, style, rep); err != nil {
		return err
	}
	if len(rep.Policy.Warnings) > 0 {
		fmt.Fprintln(w)
		writeWarnings(w, style, rep)
	}
	if rep.Explain != nil && rep.Explain.Explanation != nil {
		fmt.Fprintln(w)
		writeExplanation(w, style, *rep.Explain.Explanation, rep.Explain.Cached)
	}
	if rep.Explain != nil && rep.Explain.Warning != "" {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "  %s %s\n", style.Yellow("!"), rep.Explain.Warning)
	}
	fmt.Fprintln(w)
	writeFooter(w, style, rep)
	return nil
}

func writeSummary(w io.Writer, style ui.Style, rep scan.Report) {
	ui.BoxTitle(w, style, "Summary")
	line := fmt.Sprintf("%s %s   %s %s   %s %s   %s %s",
		style.Green("create"), style.Bold(itoa(rep.Summary.Creates)),
		style.Yellow("update"), style.Bold(itoa(rep.Summary.Updates)),
		style.Red("replace"), style.Bold(itoa(rep.Summary.Replaces)),
		style.Magenta("delete"), style.Bold(itoa(rep.Summary.Deletes)),
	)
	ui.BoxLine(w, style, line)
	ui.BoxEnd(w, style)
}

func writeCost(w io.Writer, style ui.Style, rep scan.Report) {
	ui.BoxTitle(w, style, "Cost estimate")
	delta := rep.Cost.MonthlyDeltaUSD
	deltaStr := fmt.Sprintf("%+.2f USD/mo", delta)
	switch {
	case delta > 0:
		deltaStr = style.Red(deltaStr)
	case delta < 0:
		deltaStr = style.Green(deltaStr)
	default:
		deltaStr = style.Dim(deltaStr)
	}
	ui.BoxLine(w, style, style.Dim("delta")+"  "+style.Bold(deltaStr))
	ui.BoxLine(w, style, style.Dim("priced")+" "+fmt.Sprintf("%d resources · %d unpriced", rep.Cost.Priced, rep.Cost.Skipped))
	ui.BoxEnd(w, style)

	if len(rep.Cost.Drivers) == 0 {
		fmt.Fprintln(w, style.Dim("  (no priced drivers)"))
		return
	}

	ui.Section(w, style, "Top cost drivers")
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\n",
		style.Dim("DELTA"), style.Dim("BEFORE"), style.Dim("AFTER"), style.Dim("ADDRESS"))
	for _, d := range rep.Cost.Drivers {
		deltaCell := fmt.Sprintf("%+.2f", d.DeltaUSD)
		switch {
		case d.DeltaUSD > 0:
			deltaCell = style.Red(deltaCell)
		case d.DeltaUSD < 0:
			deltaCell = style.Green(deltaCell)
		}
		fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\n",
			deltaCell,
			fmt.Sprintf("%.2f", d.BeforeUSD),
			fmt.Sprintf("%.2f", d.AfterUSD),
			d.Address,
		)
	}
	_ = tw.Flush()
}

func writeChanges(w io.Writer, style ui.Style, rep scan.Report) error {
	ui.Section(w, style, "Changes")
	if len(rep.Changes) == 0 {
		fmt.Fprintln(w, style.Dim("  (none)"))
		return nil
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\n",
		style.Dim("RISK"), style.Dim("ACTION"), style.Dim("TYPE"), style.Dim("ADDRESS"))
	for _, c := range rep.Changes {
		fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\n",
			style.Risk(c.Level.String()),
			c.Action,
			style.Dim(c.Type),
			c.Address,
		)
	}
	return tw.Flush()
}

func writeFindings(w io.Writer, style ui.Style, rep scan.Report) error {
	ui.Section(w, style, "Policy findings")
	if len(rep.Policy.Findings) == 0 {
		fmt.Fprintln(w, "  "+style.Green("✓")+" "+style.Dim("no findings"))
		return nil
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\t%s\n",
		style.Dim("SEV"), style.Dim("SOURCE"), style.Dim("ID"),
		style.Dim("RESOURCE"), style.Dim("TITLE"))
	for _, f := range rep.Policy.Findings {
		fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\t%s\n",
			style.Risk(string(f.Severity)),
			style.Cyan(string(f.Source)),
			f.ID,
			style.Dim(f.Resource),
			f.Title,
		)
	}
	return tw.Flush()
}

func writeExplanation(w io.Writer, style ui.Style, exp explain.Explanation, cached bool) {
	title := "AI summary"
	if cached {
		title += " (cached)"
	}
	ui.BoxTitle(w, style, title)
	ui.BoxLine(w, style, exp.Summary)
	ui.BoxEnd(w, style)

	if len(exp.Risks) > 0 {
		ui.Section(w, style, "Key risks")
		for _, r := range exp.Risks {
			fmt.Fprintf(w, "  %s %s\n", style.Red("•"), r)
		}
	}
	if len(exp.Recommendations) > 0 {
		ui.Section(w, style, "Recommendations")
		for _, r := range exp.Recommendations {
			fmt.Fprintf(w, "  %s %s\n", style.Cyan("•"), r)
		}
	}
	if exp.CostNote != "" {
		fmt.Fprintf(w, "  %s %s\n", style.Dim("cost:"), exp.CostNote)
	}
}

func writeWarnings(w io.Writer, style ui.Style, rep scan.Report) {
	ui.Section(w, style, "Warnings")
	for _, warn := range rep.Policy.Warnings {
		fmt.Fprintf(w, "  %s %s\n", style.Yellow("!"), warn)
	}
}

func writeFooter(w io.Writer, style ui.Style, rep scan.Report) {
	total := rep.Summary.Creates + rep.Summary.Updates + rep.Summary.Replaces + rep.Summary.Deletes
	msg := fmt.Sprintf("  %s  %d changes scanned · %d policy findings · max risk %s · cost %+.2f USD/mo",
		style.Cyan("▸"),
		total,
		len(rep.Policy.Findings),
		style.Risk(rep.MaxRisk.String()),
		rep.Cost.MonthlyDeltaUSD,
	)
	fmt.Fprintln(w, msg)
	fmt.Fprintln(w, style.Dim("  "+strings.Repeat("─", 48)))
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}
