package explain

import "encoding/json"

// Input is a compact scan snapshot for the LLM (no app package dependency).
type Input struct {
	MaxRisk        string        `json:"max_risk"`
	ChangeCount    int           `json:"change_count"`
	Creates        int           `json:"creates"`
	Updates        int           `json:"updates"`
	Replaces       int           `json:"replaces"`
	Deletes        int           `json:"deletes"`
	Changes        []ChangeLine  `json:"changes"`
	Findings       []FindingLine `json:"findings"`
	CostDeltaUSD   float64       `json:"cost_delta_usd_month"`
	TopCostDrivers []CostDriver  `json:"top_cost_drivers"`
}

// ChangeLine is one planned change in the LLM context.
type ChangeLine struct {
	Address string `json:"address"`
	Type    string `json:"type"`
	Action  string `json:"action"`
	Risk    string `json:"risk"`
}

// FindingLine is one policy finding in the LLM context.
type FindingLine struct {
	ID       string `json:"id"`
	Severity string `json:"severity"`
	Resource string `json:"resource"`
	Title    string `json:"title"`
}

// CostDriver is a top cost contributor in the LLM context.
type CostDriver struct {
	Address string  `json:"address"`
	Delta   float64 `json:"delta_usd_month"`
}

// InputJSON serializes the input for hashing and prompting.
func InputJSON(in Input) ([]byte, error) {
	return json.Marshal(in)
}
