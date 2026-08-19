# tfguard

Terraform plan reviewer for AWS. Parses `terraform plan -json`, classifies change risk (`SAFE` → `CRITICAL`), evaluates security policies, and fails CI when `-fail-on` is reached.

[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Terraform](https://img.shields.io/badge/Terraform-plan%20JSON-844FBA?logo=terraform&logoColor=white)](https://www.terraform.io/)
[![OPA](https://img.shields.io/badge/OPA-Rego-000000?logo=openpolicyagent&logoColor=white)](https://www.openpolicyagent.org/)

**Language:** English | [Français](README.fr.md)

![tfguard scan demo](docs/demo.gif)

---

## Problem

A successful `terraform plan` is not a review. Large diffs hide destructive actions—especially on stateful resources (RDS, S3, EBS). tfguard turns the plan into a structured report and an optional non-zero exit for CI gates.

## How it works

```mermaid
flowchart LR
  A[plan.json] --> B[tfplan]
  B --> C[risk]
  B --> D[policy]
  B --> H[cost]
  C --> E[app.Report]
  D --> E
  H --> E
  E --> F[render + CLI]
  F --> G["exit 0 / 1 / 2"]
```

| Layer | Package | Responsibility |
|-------|---------|----------------|
| Parse | `internal/tfplan` | Decode plan JSON; collapse actions; summarize mutations |
| Risk | `internal/risk` | Score each change; escalate stateful AWS types |
| Policy | `internal/policy` | OPA Rego (embedded) + optional Checkov/tfsec → `Finding` |
| Cost | `internal/cost` | Static AWS monthly delta from before/after attributes |
| ML score | `internal/mlscore` | Embedded logistic regression high-risk probability |
| Explain | `internal/explain` | Optional Ollama LLM summary (`--no-ai` to skip) |
| Orchestrate | `internal/app` | Build `Report`; evaluate `-fail-on` / `--max-cost-increase` |
| Present | `internal/render` | Terminal tables |
| CLI | `cmd/tfguard` | Cobra commands: `scan`, `version` |

Optional scanners (Checkov/tfsec) emit **warnings** when missing; they do not crash the run by themselves.

## Risk model

| Action | Base level | If stateful (+1) |
|--------|------------|------------------|
| create / no-op / read | `SAFE` | `CAUTION` |
| update | `CAUTION` | `DANGER` |
| replace / delete | `DANGER` | `CRITICAL` |

Stateful types include RDS, S3, EBS, DynamoDB, EFS, ElastiCache, Redshift, OpenSearch, DocumentDB, Neptune.

## Policies

Embedded Rego under `policies/` (compiled into the binary via `embed`):

| ID | Catches |
|----|---------|
| `TFGUARD-S3-001` | Public S3 ACL |
| `TFGUARD-SG-001` | Security group open to `0.0.0.0/0` |
| `TFGUARD-RDS-001` | RDS without `storage_encrypted` |
| `TFGUARD-IAM-001` | IAM policy with `Action: "*"` |
| `TFGUARD-EBS-001` | Unencrypted EBS volume |

`-fail-on` also maps finding severity onto the same scale: `CRITICAL`→`CRITICAL`, `HIGH`→`DANGER`, `MEDIUM`→`CAUTION`, else `SAFE`.

## CLI

```bash
make test
make build
./bin/tfguard version
./bin/tfguard scan --help

# Basic review
./bin/tfguard scan --plan plan.json

# With HCL scanners + CI gate
./bin/tfguard scan --plan plan.json --dir ./infra --fail-on DANGER

# Fail when monthly cost delta exceeds $50
./bin/tfguard scan --plan plan.json --max-cost-increase 50

# Skip local Ollama explainer (recommended in CI)
./bin/tfguard scan --plan plan.json --no-ai
```

| Flag | Description |
|------|-------------|
| `--plan` | Path to plan JSON (**required**) |
| `--dir` | Terraform root for Checkov/tfsec |
| `--fail-on` | `SAFE` \| `CAUTION` \| `DANGER` \| `CRITICAL` |
| `--max-cost-increase` | Exit 1 if monthly cost delta USD exceeds this value |
| `--no-ai` | Skip the Ollama LLM explainer |
| `--ollama-url` / `--ollama-model` | Ollama endpoint and model (default `llama3.2`) |
| `--skip-checkov` / `--skip-tfsec` / `--skip-opa` / `--skip-cost` | Disable a stage |

**Exit codes:** `0` ok · `1` threshold hit or runtime error · `2` usage error

Example report (abridged):

```text
tfguard scan report
Plan: testdata/plan_mixed.json
Max risk: CRITICAL

Summary
----------------------------------------
  create : 1
  update : 1
  replace: 1
  delete : 1

Changes
----------------------------------------
RISK      ACTION   TYPE             ADDRESS
CRITICAL  delete   aws_db_instance  aws_db_instance.main
...
```

## Repository layout

```text
cmd/tfguard/       CLI entrypoint
internal/tfplan/   plan parse + summary
internal/risk/     risk levels
internal/policy/   Checkov / tfsec / OPA
internal/cost/     static AWS cost delta
internal/mlscore/  embedded ML risk probability
internal/explain/  optional Ollama LLM summary
internal/app/      orchestration + fail-on
internal/render/   terminal report
policies/          embedded Rego rules
scripts/           dataset + model training
testdata/          fixtures for unit tests
```

Module path: `github.com/SamyBaouche/tfguard`

## Cost estimate

Static us-east-1 on-demand prices (~20 SKUs: EC2, RDS, ElastiCache, NAT Gateway, …).
Extracts billable attributes from each change’s before/after (`instance_type`, `instance_class`, `allocated_storage`, …), sums a monthly delta, and shows the top 3 drivers.

Use `--max-cost-increase <USD>` as a CI gate on positive deltas. Figures are for plan review, not billing accuracy.

## ML risk score

Trained logistic regression on ~250 synthetic labeled plans (`scripts/generate_dataset.py` + `scripts/train_model.py`).
Features: change counts, stateful mutations, finding severities, cost delta.
Coefficients are embedded in the binary (`internal/mlscore/model.json`).

Hold-out metrics (seed=42): **precision 1.00 · recall 0.98 · F1 0.99**.

## AI explainer (optional)

When Ollama is running locally, tfguard sends a compact JSON context (changes + findings + cost) to `/api/generate` and prints a structured summary.
Responses are cached by SHA-256 under `$XDG_CACHE_HOME/tfguard/explain/`.
Use `--no-ai` in CI or when Ollama is unavailable.

## GitHub Action

Use the repo root `action.yml` in a workflow:

```yaml
- uses: ./ 
  with:
    plan-path: plan.json
    fail-on: DANGER
    no-ai: "true"
```

## Roadmap

1. **Done** — parse, risk, policies, CLI, cost delta, ML score, Ollama explainer, GitHub Action scaffold
2. **Planned** — demo GIF, GoReleaser tag `v0.1.0`, richer ML dataset

## Development

```bash
make test && make vet && make fmt
```

- Prefer table-driven tests and fixtures over live Terraform
- Wrap errors with `%w`; optional tools soft-fail with warnings
- Keep packages under `internal/` until a public API is intentional

## License

Apache 2.0 — see [LICENSE](LICENSE).
