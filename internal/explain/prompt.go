package explain

import "fmt"

const systemPrompt = `You are a senior DevOps engineer reviewing a Terraform plan for AWS.
Explain the scan in plain language for a teammate. Be concise and actionable.
Respond with JSON only, matching the required schema.
Focus on the highest-risk changes, security findings, and cost impact.
Do not invent resources or findings not present in the input.`

func buildPrompt(ctxJSON []byte) string {
	return fmt.Sprintf(`%s

Scan data (JSON):
%s

Return JSON with:
- summary: 2-3 sentences overview
- risks: bullet list of main risks (max 5)
- recommendations: bullet list of next steps (max 5)
- cost_note: one sentence about monthly cost delta (or "no priced changes" if zero)`, systemPrompt, string(ctxJSON))
}
