// Package cost estimates a static monthly AWS cost delta from plan changes.
package cost

// monthlyUSD maps resource type → SKU → approximate USD/month (us-east-1, on-demand).
// Prices are rough static estimates for plan review, not billing accuracy.
// EC2/RDS figures assume ~730 hours/month.
var monthlyUSD = map[string]map[string]float64{
	"aws_instance": {
		"t3.nano":   3.80,
		"t3.micro":  7.59,
		"t3.small":  15.18,
		"t3.medium": 30.37,
		"t3.large":  60.74,
		"t3.xlarge": 121.47,
		"m5.large":  70.08,
		"m5.xlarge": 140.16,
		"c5.large":  62.05,
		"c5.xlarge": 124.10,
		"r5.large":  91.98,
		"r5.xlarge": 183.96,
	},
	"aws_db_instance": {
		"db.t3.micro":  12.41,
		"db.t3.small":  24.82,
		"db.t3.medium": 49.64,
		"db.t3.large":  99.28,
		"db.m5.large":  138.70,
		"db.m5.xlarge": 277.40,
		"db.r5.large":  175.20,
	},
	"aws_elasticache_cluster": {
		"cache.t3.micro": 11.68,
		"cache.t3.small": 23.36,
		"cache.m5.large": 113.88,
	},
	"aws_elasticsearch_domain": {
		"t3.small.elasticsearch": 25.55,
		"m5.large.elasticsearch": 131.40,
	},
	"aws_opensearch_domain": {
		"t3.small.search": 25.55,
		"m5.large.search": 131.40,
	},
	"aws_redshift_cluster": {
		"dc2.large":  180.00,
		"ra3.xlplus": 730.00,
	},
}

// fixedMonthlyUSD is a flat monthly price when the resource exists (create/delete).
var fixedMonthlyUSD = map[string]float64{
	"aws_nat_gateway":          32.40, // ~$0.045/h
	"aws_lb":                   16.20, // ALB base ~$0.0225/h
	"aws_alb":                  16.20,
	"aws_lb_listener":          0,    // priced via parent LB
	"aws_eip":                  3.65, // idle Elastic IP ~$0.005/h
	"aws_efs_file_system":      0,    // storage-driven
	"aws_s3_bucket":            0,    // usage-driven; ignore base
	"aws_dynamodb_table":       0,    // on-demand/usage-driven
	"aws_security_group":       0,
	"aws_subnet":               0,
	"aws_vpc":                  0,
	"aws_internet_gateway":     0,
	"aws_route_table":          0,
	"aws_iam_role":             0,
	"aws_iam_policy":           0,
	"aws_cloudwatch_log_group": 0,
}

// storageGBMonthlyUSD is USD per GB-month for storage attributes.
var storageGBMonthlyUSD = map[string]float64{
	"aws_db_instance":     0.115, // gp2/gp3-ish RDS storage
	"aws_ebs_volume":      0.08,  // gp3
	"aws_efs_file_system": 0.30,
}

// lookupInstancePrice returns the monthly USD for a typed instance SKU, or 0/false.
func lookupInstancePrice(resourceType, sku string) (float64, bool) {
	bySKU, ok := monthlyUSD[resourceType]
	if !ok || sku == "" {
		return 0, false
	}
	p, ok := bySKU[sku]
	return p, ok
}
