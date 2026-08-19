// Package mlscore applies an embedded logistic regression model to plan features.
package mlscore

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
)

//go:embed model.json
var modelJSON []byte

// Model is exported logistic regression coefficients.
type Model struct {
	Features  []string  `json:"features"`
	Intercept float64   `json:"intercept"`
	Weights   []float64 `json:"weights"`
}

// Score is the ML prediction for one scan.
type Score struct {
	Probability float64 // 0..1 high-risk probability
	HighRisk    bool    // probability >= 0.5
}

// Features extracted from a scan report for inference.
type Features struct {
	Creates           int
	Updates           int
	Replaces          int
	Deletes           int
	StatefulMutations int
	CriticalFindings  int
	HighFindings      int
	CostDeltaUSD      float64
	MaxActionLevel    int
	ChangeCount       int
}

var loaded Model

func init() {
	if err := json.Unmarshal(modelJSON, &loaded); err != nil {
		panic(fmt.Sprintf("mlscore: bad embedded model: %v", err))
	}
}

// Predict returns the high-risk probability for a feature vector.
func Predict(f Features) Score {
	values := []float64{
		float64(f.Creates),
		float64(f.Updates),
		float64(f.Replaces),
		float64(f.Deletes),
		float64(f.StatefulMutations),
		float64(f.CriticalFindings),
		float64(f.HighFindings),
		f.CostDeltaUSD,
		float64(f.MaxActionLevel),
		float64(f.ChangeCount),
	}
	z := loaded.Intercept
	for i, w := range loaded.Weights {
		if i < len(values) {
			z += w * values[i]
		}
	}
	p := sigmoid(z)
	return Score{Probability: p, HighRisk: p >= 0.5}
}

func sigmoid(x float64) float64 {
	if x >= 0 {
		z := math.Exp(-x)
		return 1.0 / (1.0 + z)
	}
	z := math.Exp(x)
	return z / (1.0 + z)
}

// ModelInfo returns embedded model metadata for docs/tests.
func ModelInfo() Model { return loaded }
