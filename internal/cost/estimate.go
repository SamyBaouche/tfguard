package cost

import (
	"encoding/json"
	"math"
	"sort"

	"github.com/SamyBaouche/tfguard/internal/tfplan"
)

// Driver is one resource's contribution to the monthly cost delta.
type Driver struct {
	Address   string
	Type      string
	BeforeUSD float64
	AfterUSD  float64
	DeltaUSD  float64
}

// Estimate is the static monthly cost delta for a set of plan changes.
type Estimate struct {
	MonthlyDeltaUSD float64
	Drivers         []Driver // top contributors by |delta|, max 3
	Priced          int      // resources with a non-zero before or after price
	Skipped         int      // mutating changes with no known price
}

// EstimateChanges computes a static us-east-1 monthly delta from mutating changes.
func EstimateChanges(changes []tfplan.ResourceChange) Estimate {
	var (
		drivers []Driver
		priced  int
		skipped int
		total   float64
	)

	for _, rc := range changes {
		beforeUSD := monthlyCost(rc.Type, rc.Change.Before)
		afterUSD := monthlyCost(rc.Type, rc.Change.After)
		delta := afterUSD - beforeUSD

		if beforeUSD == 0 && afterUSD == 0 {
			skipped++
			continue
		}
		priced++
		total += delta
		drivers = append(drivers, Driver{
			Address:   rc.Address,
			Type:      rc.Type,
			BeforeUSD: beforeUSD,
			AfterUSD:  afterUSD,
			DeltaUSD:  delta,
		})
	}

	sort.Slice(drivers, func(i, j int) bool {
		ai, aj := math.Abs(drivers[i].DeltaUSD), math.Abs(drivers[j].DeltaUSD)
		if ai != aj {
			return ai > aj
		}
		return drivers[i].Address < drivers[j].Address
	})

	const topN = 3
	if len(drivers) > topN {
		drivers = drivers[:topN]
	}

	return Estimate{
		MonthlyDeltaUSD: total,
		Drivers:         drivers,
		Priced:          priced,
		Skipped:         skipped,
	}
}

// monthlyCost computes the approximate USD/month for one resource type and
// one side of the plan change (`before` or `after`).
//
// The estimator is intentionally static:
// - it only reads a small subset of attributes we know how to price
// - S3/DynamoDB are mostly treated as unpriced (usage-driven)
// - the goal is to highlight large deltas in reviews, not match AWS billing.
func monthlyCost(resourceType string, raw json.RawMessage) float64 {
	attrs := decodeAttrs(raw)
	if attrs == nil {
		return 0
	}

	var total float64
	priced := false

	priceType := pricingType(resourceType)
	if sku := instanceSKU(resourceType, attrs); sku != "" {
		if p, ok := lookupInstancePrice(priceType, sku); ok {
			nodes := cacheNodes(resourceType, attrs)
			total += p * float64(nodes)
			priced = true
		}
	}

	if gb := storageGB(resourceType, attrs); gb > 0 {
		if rate, ok := storageGBMonthlyUSD[resourceType]; ok {
			total += gb * rate
			priced = true
		}
	}

	if !priced {
		if p, ok := fixedMonthlyUSD[resourceType]; ok {
			total += p
		}
	}

	return total
}

// decodeAttrs decodes a Terraform before/after blob into a generic map.
// If the blob is null/unparseable, it returns nil.
func decodeAttrs(raw json.RawMessage) map[string]any {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var attrs map[string]any
	if err := json.Unmarshal(raw, &attrs); err != nil {
		return nil
	}
	return attrs
}

// instanceSKU extracts the instance size identifier (e.g. `t3.micro`) for
// resources where we have SKU-based pricing.
func instanceSKU(resourceType string, attrs map[string]any) string {
	switch resourceType {
	case "aws_instance":
		return stringAttr(attrs, "instance_type")
	case "aws_db_instance":
		return stringAttr(attrs, "instance_class")
	case "aws_elasticache_cluster", "aws_elasticache_replication_group":
		return stringAttr(attrs, "node_type")
	case "aws_elasticsearch_domain", "aws_opensearch_domain":
		if v := stringAttr(attrs, "instance_type"); v != "" {
			return v
		}
		// nested cluster_config.instance_type
		if cfg, ok := attrs["cluster_config"].(map[string]any); ok {
			return stringAttr(cfg, "instance_type")
		}
		return ""
	case "aws_redshift_cluster":
		return stringAttr(attrs, "node_type")
	default:
		return ""
	}
}

// storageGB extracts the provisioned storage size (in GB) when the resource
// has storage-based pricing in our static table.
func storageGB(resourceType string, attrs map[string]any) float64 {
	switch resourceType {
	case "aws_db_instance":
		return floatAttr(attrs, "allocated_storage")
	case "aws_ebs_volume":
		return floatAttr(attrs, "size")
	case "aws_efs_file_system":
		// size_in_bytes is after-only; prefer provisioned throughput sizing later
		return 0
	default:
		return 0
	}
}

// pricingType normalizes resource types so they share the same price table.
func pricingType(resourceType string) string {
	switch resourceType {
	case "aws_elasticache_replication_group":
		return "aws_elasticache_cluster"
	default:
		return resourceType
	}
}

// cacheNodes returns the number of billable nodes for services that price
// per node (e.g. ElastiCache).
func cacheNodes(resourceType string, attrs map[string]any) int {
	switch resourceType {
	case "aws_elasticache_cluster":
		n := int(floatAttr(attrs, "num_cache_nodes"))
		if n < 1 {
			return 1
		}
		return n
	case "aws_elasticache_replication_group", "aws_redshift_cluster":
		n := int(floatAttr(attrs, "number_of_nodes"))
		if n < 1 {
			return 1
		}
		return n
	default:
		return 1
	}
}

// stringAttr safely extracts a string attribute.
func stringAttr(attrs map[string]any, key string) string {
	v, ok := attrs[key]
	if !ok || v == nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

// floatAttr safely extracts a numeric attribute.
func floatAttr(attrs map[string]any, key string) float64 {
	v, ok := attrs[key]
	if !ok || v == nil {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case json.Number:
		f, _ := n.Float64()
		return f
	default:
		return 0
	}
}
