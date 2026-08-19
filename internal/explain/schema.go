package explain

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Explanation is the structured LLM output shown in the scan report.
type Explanation struct {
	Summary         string   `json:"summary"`
	Risks           []string `json:"risks"`
	Recommendations []string `json:"recommendations"`
	CostNote        string   `json:"cost_note"`
}

// JSONSchema is passed to Ollama as the response format constraint.
const JSONSchema = `{
  "type": "object",
  "required": ["summary", "risks", "recommendations", "cost_note"],
  "properties": {
    "summary": {"type": "string"},
    "risks": {"type": "array", "items": {"type": "string"}},
    "recommendations": {"type": "array", "items": {"type": "string"}},
    "cost_note": {"type": "string"}
  }
}`

func parseExplanation(raw []byte) (Explanation, error) {
	raw = extractJSONObject(raw)
	var out Explanation
	if err := json.Unmarshal(raw, &out); err != nil {
		return Explanation{}, fmt.Errorf("decode explanation: %w", err)
	}
	if err := validateExplanation(out); err != nil {
		return Explanation{}, err
	}
	return out, nil
}

func validateExplanation(e Explanation) error {
	if strings.TrimSpace(e.Summary) == "" {
		return fmt.Errorf("explanation missing summary")
	}
	if e.Risks == nil {
		return fmt.Errorf("explanation missing risks array")
	}
	if e.Recommendations == nil {
		return fmt.Errorf("explanation missing recommendations array")
	}
	return nil
}

// extractJSONObject keeps the first {...} block when models wrap JSON in prose.
func extractJSONObject(b []byte) []byte {
	s := strings.TrimSpace(string(b))
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		return []byte(s[start : end+1])
	}
	return b
}
