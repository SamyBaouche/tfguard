package cost

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/SamyBaouche/tfguard/internal/tfplan"
)

func TestEstimateEC2Resize(t *testing.T) {
	t.Parallel()
	changes := []tfplan.ResourceChange{{
		Address: "aws_instance.web",
		Type:    "aws_instance",
		Change: tfplan.Change{
			Actions: []string{"update"},
			Before:  raw(`{"instance_type":"t3.micro"}`),
			After:   raw(`{"instance_type":"t3.small"}`),
		},
	}}
	got := EstimateChanges(changes)
	want := 15.18 - 7.59
	if math.Abs(got.MonthlyDeltaUSD-want) > 0.001 {
		t.Fatalf("delta = %v, want %v", got.MonthlyDeltaUSD, want)
	}
	if got.Priced != 1 || len(got.Drivers) != 1 {
		t.Fatalf("priced/drivers = %d/%d", got.Priced, len(got.Drivers))
	}
	if got.Drivers[0].Address != "aws_instance.web" {
		t.Fatalf("driver = %+v", got.Drivers[0])
	}
}

func TestEstimateCreateDelete(t *testing.T) {
	t.Parallel()
	changes := []tfplan.ResourceChange{
		{
			Address: "aws_nat_gateway.main",
			Type:    "aws_nat_gateway",
			Change: tfplan.Change{
				Actions: []string{"create"},
				Before:  raw(`null`),
				After:   raw(`{"allocation_id":"eipalloc-1"}`),
			},
		},
		{
			Address: "aws_instance.old",
			Type:    "aws_instance",
			Change: tfplan.Change{
				Actions: []string{"delete"},
				Before:  raw(`{"instance_type":"t3.micro"}`),
				After:   raw(`null`),
			},
		},
	}
	got := EstimateChanges(changes)
	want := 32.40 - 7.59
	if math.Abs(got.MonthlyDeltaUSD-want) > 0.001 {
		t.Fatalf("delta = %v, want %v", got.MonthlyDeltaUSD, want)
	}
	if len(got.Drivers) != 2 {
		t.Fatalf("drivers = %d, want 2", len(got.Drivers))
	}
	// Top driver by |delta| should be NAT gateway.
	if got.Drivers[0].Address != "aws_nat_gateway.main" {
		t.Fatalf("top driver = %s", got.Drivers[0].Address)
	}
}

func TestEstimateTopThreeDrivers(t *testing.T) {
	t.Parallel()
	changes := []tfplan.ResourceChange{
		mkInstance("a", "t3.micro", "t3.small"),  // +7.59
		mkInstance("b", "t3.micro", "t3.medium"), // +22.78
		mkInstance("c", "t3.nano", "t3.large"),   // +56.94
		mkInstance("d", "t3.micro", "t3.micro"),  // 0 — skipped as priced with 0 delta still priced
	}
	// d has same type both sides → delta 0 but still priced; still a driver.
	got := EstimateChanges(changes)
	if len(got.Drivers) != 3 {
		t.Fatalf("drivers = %d, want top 3", len(got.Drivers))
	}
	if got.Drivers[0].Address != "aws_instance.c" {
		t.Fatalf("top = %s", got.Drivers[0].Address)
	}
}

func TestEstimateRDSWithStorage(t *testing.T) {
	t.Parallel()
	changes := []tfplan.ResourceChange{{
		Address: "aws_db_instance.main",
		Type:    "aws_db_instance",
		Change: tfplan.Change{
			Actions: []string{"update"},
			Before:  raw(`{"instance_class":"db.t3.micro","allocated_storage":20}`),
			After:   raw(`{"instance_class":"db.t3.small","allocated_storage":100}`),
		},
	}}
	got := EstimateChanges(changes)
	before := 12.41 + 20*0.115
	after := 24.82 + 100*0.115
	want := after - before
	if math.Abs(got.MonthlyDeltaUSD-want) > 0.001 {
		t.Fatalf("delta = %v, want %v", got.MonthlyDeltaUSD, want)
	}
}

func TestEstimateUnknownSkipped(t *testing.T) {
	t.Parallel()
	got := EstimateChanges([]tfplan.ResourceChange{{
		Address: "aws_s3_bucket.logs",
		Type:    "aws_s3_bucket",
		Change: tfplan.Change{
			Actions: []string{"create"},
			Before:  raw(`null`),
			After:   raw(`{"bucket":"logs"}`),
		},
	}})
	if got.Skipped != 1 || got.Priced != 0 || got.MonthlyDeltaUSD != 0 {
		t.Fatalf("got %+v", got)
	}
}

func mkInstance(name, before, after string) tfplan.ResourceChange {
	return tfplan.ResourceChange{
		Address: "aws_instance." + name,
		Type:    "aws_instance",
		Change: tfplan.Change{
			Actions: []string{"update"},
			Before:  raw(`{"instance_type":"` + before + `"}`),
			After:   raw(`{"instance_type":"` + after + `"}`),
		},
	}
}

func raw(s string) json.RawMessage { return json.RawMessage(s) }
