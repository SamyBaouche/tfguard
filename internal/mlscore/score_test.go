package mlscore

import (
	"math"
	"testing"
)

func TestPredictDeterministic(t *testing.T) {
	t.Parallel()
	got := Predict(Features{
		Deletes:           2,
		Replaces:          1,
		StatefulMutations: 2,
		CriticalFindings:  1,
		CostDeltaUSD:      50,
		MaxActionLevel:    2,
		ChangeCount:       5,
	})
	if got.Probability <= 0 || got.Probability > 1 {
		t.Fatalf("probability out of range: %v", got.Probability)
	}
	if !got.HighRisk {
		t.Fatalf("expected high risk, got %+v", got)
	}
}

func TestPredictLowRisk(t *testing.T) {
	t.Parallel()
	got := Predict(Features{Creates: 1, ChangeCount: 1})
	if got.HighRisk {
		t.Fatalf("expected low risk, got %+v", got)
	}
	if math.IsNaN(got.Probability) {
		t.Fatal("nan probability")
	}
}

func TestEmbeddedModel(t *testing.T) {
	t.Parallel()
	m := ModelInfo()
	if len(m.Features) != len(m.Weights) || m.Features[0] != "creates" {
		t.Fatalf("model = %+v", m)
	}
}
