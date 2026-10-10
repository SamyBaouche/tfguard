# tfguard

**A pre-apply reviewer for Terraform on AWS.** tfguard reads a `terraform plan` JSON, scores the blast radius of every change, runs security policies, estimates the monthly cost delta, and gates CI with a deterministic exit code — with no cloud credentials, and in under a second for the built-in checks.

[![CI](https://github.com/SamyBaouche/tfguard/actions/workflows/ci.yml/badge.svg)](https://github.com/SamyBaouche/tfguard/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/SamyBaouche/tfguard?label=v0.1.0)](https://github.com/SamyBaouche/tfguard/releases/tag/v0.1.0)
[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Terraform](https://img.shields.io/badge/Terraform-plan%20JSON-844FBA?logo=terraform&logoColor=white)](https://www.terraform.io/)
[![OPA](https://img.shields.io/badge/OPA-Rego-000000?logo=openpolicyagent&logoColor=white)](https://www.openpolicyagent.org/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

> **English** · Read in [Français](README.fr.md)

---

## Table of contents

- [Demo](#demo)
- [Why tfguard](#why-tfguard)
- [Features](#features)
- [How it works](#how-it-works)
- [Installation](#installation)
- [Quick start](#quick-start)
- [Generate a Terraform plan JSON](#generate-a-terraform-plan-json)
- [CLI reference](#cli-reference)
- [Report sections](#report-sections)
- [Risk model](#risk-model)
- [Security policies](#security-policies)
- [Cost estimation](#cost-estimation)
- [ML risk score](#ml-risk-score)
- [AI explainer (optional)](#ai-explainer-optional)
- [CI/CD integration](#cicd-integration)
- [GitHub Action](#github-action)
- [Project structure](#project-structure)
- [Development](#development)
- [Design decisions](#design-decisions)
- [Limitations](#limitations)
- [Roadmap](#roadmap)
- [Contributing](#contributing)
- [License](#license)

---

## Demo

![tfguard scan demo](docs/demo.gif)

> **Add your own screenshots** under `docs/images/` and embed them below:
>
> | Image | Suggested file | What to capture |
> |-------|----------------|-----------------|
> | Full terminal report | `docs/images/report-full.png` | Complete `scan` output |
> | Architecture diagram | `docs/images/architecture.png` | Pipeline / boxes diagram |
> | Cost drivers section | `docs/images/cost-drivers.png` | Top 3 cost drivers table |
> | GitHub Action PR comment | `docs/images/ci-pr-comment.png` | PR comment from Action |
> | Risk matrix | `docs/images/risk-matrix.png` | SAFE → CRITICAL legend |

<!-- Uncomment when you add screenshots:
![Full report](docs/images/report-full.png)
![Architecture](docs/images/architecture.png)
![Cost drivers](docs/images/cost-drivers.png)
![CI PR comment](docs/images/ci-pr-comment.png)
-->

---

## Why tfguard

**A successful `terraform plan` is not a safe change.**

In a 200-line diff it is easy to miss that a database is being **deleted**, a security group is being **replaced**, or an instance resize adds **$100/month**. Those mistakes are cheap to catch before `apply` and expensive after.

tfguard reads the plan JSON **before apply** and turns it into a structured review plus a deterministic exit code, so the same checks run on a laptop and as a required CI gate.

| Question a reviewer asks | How tfguard answers it |
|---|---|
| **What is actually changing?** | Collapses Terraform actions into `create / update / replace / delete` |
| **Could this lose data?** | Risk levels `SAFE` → `CRITICAL`, escalated for stateful AWS resources |
| **Is it secure?** | Embedded OPA/Rego policies, plus Checkov and tfsec when installed |
| **What will it cost?** | Static monthly USD delta with the top cost drivers |
| **How risky is the plan overall?** | ML high-risk probability |
| **Can someone explain it to me?** | Optional plain-language summary from a local LLM (Ollama) |
| **Should CI block it?** | `--fail-on` and `--max-cost-increase` gates |

tfguard **does not deploy** infrastructure. It **reviews** the plan and can **block CI** with a non-zero exit code.

---

## Features

### Core

- **Plan parsing** — decode `terraform show -json` output; collapse `["delete","create"]` into `replace`
- **Mutation summary** — count create / update / replace / delete; skip no-ops, reads and data sources
- **Risk classification** — rule-based levels with **stateful escalation** (RDS, S3, EBS, DynamoDB, …)
- **Policy scanning** — embedded **OPA/Rego** (5 rules) + optional **Checkov** / **tfsec**, normalized into one finding model
- **Cost delta** — static us-east-1 pricing (28 instance sizes plus storage and flat-rate resources); top **3 cost drivers**
- **ML score** — embedded logistic regression producing a high-risk probability
- **LLM explainer** — optional local Ollama summary with JSON-schema output + SHA-256 cache
- **Terminal report** — colored boxes, tables, spinner, banner (`NO_COLOR` aware)
- **CI gates** — `--fail-on` (risk + findings) and `--max-cost-increase` (FinOps)
- **Zero cloud access** — works entirely from the plan file; no AWS credentials required

### Release & automation

- **v0.1.0** published via **GoReleaser** (6 platforms, single static binary)
- **GitHub Action** (`action.yml`) — runs the scan and comments on PRs
- **CI workflow** — `go test` + `go vet` on every push and pull request

---

## How it works

tfguard is a linear pipeline. Each stage is a small package with a single responsibility, and `internal/scan` composes them into one `Report`.

```mermaid
flowchart TB
  subgraph Input
    A[plan.json]
  end

  subgraph Pipeline
    B[tfplan — parse & summarize]
    C[risk — classify changes]
    D[policy — OPA / Checkov / tfsec]
    E[cost — monthly delta]
    F[mlscore — probability]
    G[explain — Ollama optional]
  end

  subgraph Output
    H[render — terminal report]
    I{CI gates}
    J[exit 0]
    K[exit 1]
  end

  A --> B
  B --> C
  B --> D
  B --> E
  C --> scan[scan.Report]
  D --> scan
  E --> scan
  scan --> F
  scan --> G
  F --> H
  G --> H
  H --> I
  I -->|pass| J
  I -->|fail-on / max-cost| K
```

| Layer | Package | Responsibility |
|-------|---------|----------------|
| CLI | `cmd/tfguard` | Cobra commands (`scan`, `version`), flags, exit-code mapping |
| Orchestrate | `internal/scan` | Run the pipeline, build `Report`, evaluate gates |
| Parse | `internal/tfplan` | Typed plan model; collapse actions; sorted mutation summary |
| Risk | `internal/risk` | Action → level; stateful escalation |
| Policy | `internal/policy` | Embedded OPA + optional Checkov/tfsec → unified `Finding` |
| Cost | `internal/cost` | Static price tables; per-resource before/after delta |
| ML score | `internal/mlscore` | Embedded logistic regression inference |
| Explain | `internal/explain` | Prompt, Ollama client, JSON schema, cache |
| Present | `internal/render` + `internal/ui` | Terminal report + styling |
| Policies | `policies/` | Rego sources embedded with `go:embed` |

Optional scanners (Checkov/tfsec) and Ollama emit **warnings** when missing — they never crash the run.

---

## Installation

### Option A — Build from source

**Requirements:** Go 1.26+

```bash
git clone https://github.com/SamyBaouche/tfguard.git
cd tfguard
make build
./bin/tfguard version
```

Or install directly into `$GOPATH/bin`:

```bash
go install github.com/SamyBaouche/tfguard/cmd/tfguard@latest
```

### Option B — Download release binary

Pick your platform from [Releases v0.1.0](https://github.com/SamyBaouche/tfguard/releases/tag/v0.1.0):

| Platform | Archive |
|----------|---------|
| Linux amd64 | `tfguard_0.1.0_linux_amd64.tar.gz` |
| Linux arm64 | `tfguard_0.1.0_linux_arm64.tar.gz` |
| macOS amd64 | `tfguard_0.1.0_darwin_amd64.tar.gz` |
| macOS arm64 | `tfguard_0.1.0_darwin_arm64.tar.gz` |
| Windows amd64 | `tfguard_0.1.0_windows_amd64.tar.gz` |
| Windows arm64 | `tfguard_0.1.0_windows_arm64.tar.gz` |

```bash
tar -xzf tfguard_0.1.0_linux_amd64.tar.gz
./tfguard version
```

### Optional dependencies

| Tool | Purpose | Required? |
|------|---------|-----------|
| **Terraform** | Generate `plan.json` | Only for real infra review |
| **Checkov** | HCL policy scan | No (warning if missing) |
| **tfsec** | HCL security scan | No (warning if missing) |
| **Ollama** | AI explainer | No (`--no-ai` skips it) |

---

## Quick start

### Test with bundled fixtures (no Terraform needed)

```bash
make build

# Full demo plan (create + update + delete + replace)
./bin/tfguard scan --plan testdata/plan_mixed.json --no-ai --no-banner

# Minimal plan
./bin/tfguard scan --plan testdata/plan_minimal.json --no-ai --no-banner

# Replace-only plan
./bin/tfguard scan --plan testdata/plan_replace.json --no-ai --no-banner
```

Expected on `plan_mixed.json`:
- **Max risk:** `CRITICAL` (delete `aws_db_instance.main`)
- **Cost delta:** `+7.59 USD/mo` (t3.micro → t3.small)
- **ML score:** high-risk probability displayed in header

### Full experience (banner + spinner)

```bash
./bin/tfguard scan --plan testdata/plan_mixed.json --no-ai
```

Set `NO_COLOR=1` for plain output without ANSI colors.

---

## Generate a Terraform plan JSON

Run these commands **inside your Terraform project** (folder with `.tf` files), not inside the tfguard repo:

```bash
cd /path/to/your/terraform-project
terraform init
terraform plan -out=plan.tfplan
terraform show -json plan.tfplan > plan.json
```

Then scan:

```bash
tfguard scan --plan plan.json --no-ai
```

With HCL scanners:

```bash
tfguard scan --plan plan.json --dir . --no-ai
```

---

## CLI reference

### Commands

```bash
tfguard scan --plan <path>   # Analyze a plan JSON
tfguard version              # Print version
tfguard --help               # Help
```

### `scan` flags

| Flag | Description | Default |
|------|-------------|---------|
| `--plan` | Path to terraform plan JSON | **required** |
| `--dir` | Terraform root for Checkov/tfsec | — |
| `--fail-on` | Exit 1 at level: `SAFE\|CAUTION\|DANGER\|CRITICAL` | disabled |
| `--max-cost-increase` | Exit 1 if monthly cost delta USD exceeds value | disabled |
| `--no-ai` | Skip Ollama LLM explainer | false |
| `--ollama-url` | Ollama base URL | `http://127.0.0.1:11434` |
| `--ollama-model` | Ollama model name | `llama3.2` |
| `--skip-checkov` | Do not run Checkov | false |
| `--skip-tfsec` | Do not run tfsec | false |
| `--skip-opa` | Do not run OPA policies | false |
| `--skip-cost` | Do not estimate cost delta | false |
| `--no-banner` | Skip animated banner | false |

### Exit codes

| Code | Meaning |
|------|---------|
| `0` | Scan OK; gates passed |
| `1` | Gate triggered (`--fail-on` / `--max-cost-increase`) or runtime error |
| `2` | Usage error (missing `--plan`, invalid flag, unknown command) |

### Common recipes

```bash
# Basic review
tfguard scan --plan plan.json --no-ai

# CI: block dangerous plans
tfguard scan --plan plan.json --fail-on DANGER --no-ai

# CI: block cost increases over $50/mo
tfguard scan --plan plan.json --max-cost-increase 50 --no-ai

# Strict: block at CAUTION
tfguard scan --plan plan.json --fail-on CAUTION --no-ai

# With HCL scanners + all gates
tfguard scan --plan plan.json --dir ./infra --fail-on CRITICAL --max-cost-increase 100 --no-ai

# Local dev with AI summary (requires Ollama)
ollama pull llama3.2
tfguard scan --plan plan.json

# Fast scan: skip cost + AI
tfguard scan --plan plan.json --skip-cost --no-ai --no-banner
```

---

## Report sections

When you run `scan`, the terminal report includes:

| Section | Content |
|---------|---------|
| **Scan report** | Plan path, max risk, ML probability |
| **Summary** | Counts: create / update / replace / delete |
| **Cost estimate** | Monthly delta USD, priced vs unpriced resources |
| **Top cost drivers** | Top 3 resources by \|delta\| |
| **Changes** | Table: RISK · ACTION · TYPE · ADDRESS |
| **Policy findings** | Table: SEV · SOURCE · ID · RESOURCE · TITLE |
| **Warnings** | Missing Checkov/tfsec, Ollama unavailable, etc. |
| **AI summary** | (if Ollama enabled) summary, risks, recommendations |
| **Footer** | Totals + cost delta |

Real output on `testdata/plan_mixed.json`:

```text
╭─ Scan report ────────────────────────────────────────────╮
│ plan   testdata/plan_mixed.json                          │
│ risk   CRITICAL  (highest)                               │
│ ml     100% high-risk probability                        │
╰──────────────────────────────────────────────────────────╯

╭─ Summary ────────────────────────────────────────────────╮
│ create 1   update 1   replace 1   delete 1               │
╰──────────────────────────────────────────────────────────╯

╭─ Cost estimate ──────────────────────────────────────────╮
│ delta  +7.59 USD/mo                                      │
│ priced 1 resources · 3 unpriced                          │
╰──────────────────────────────────────────────────────────╯
▸ Top cost drivers
  DELTA  BEFORE  AFTER  ADDRESS
  +7.59  7.59    15.18  aws_instance.web

▸ Changes
  RISK      ACTION   TYPE                ADDRESS
  CRITICAL  delete   aws_db_instance     aws_db_instance.main
  CAUTION   update   aws_instance        aws_instance.web
  CAUTION   create   aws_s3_bucket       aws_s3_bucket.logs
  DANGER    replace  aws_security_group  aws_security_group.web

▸ Policy findings
  ✓ no findings

  ▸  4 changes scanned · 0 policy findings · max risk CRITICAL · cost +7.59 USD/mo
```

---

## Risk model

Each mutating change gets a base level from its action, then **escalates by one level** if the resource holds durable data. The plan's overall risk is the highest level across all changes.

| Action | Base level | If stateful (+1) |
|--------|------------|------------------|
| create / no-op / read | `SAFE` | `CAUTION` |
| update | `CAUTION` | `DANGER` |
| replace / delete | `DANGER` | `CRITICAL` |

**Stateful types** (data-bearing): `aws_db_instance`, `aws_rds_cluster`, `aws_rds_cluster_instance`, `aws_s3_bucket`, `aws_ebs_volume`, `aws_ebs_snapshot`, `aws_dynamodb_table`, `aws_efs_file_system`, `aws_elasticache_cluster`, `aws_elasticache_replication_group`, `aws_redshift_cluster`, `aws_opensearch_domain`, `aws_elasticsearch_domain`, `aws_docdb_cluster`, `aws_neptune_cluster`.

---

## Security policies

Embedded Rego under `policies/` (compiled into the binary), evaluated against each change's `after` values:

| ID | Severity | Catches |
|----|----------|---------|
| `TFGUARD-S3-001` | HIGH | Public S3 ACL |
| `TFGUARD-SG-001` | CRITICAL | Security group (or rule) open to `0.0.0.0/0` |
| `TFGUARD-RDS-001` | HIGH | RDS without `storage_encrypted` |
| `TFGUARD-IAM-001` | CRITICAL | IAM policy with `Action: "*"` |
| `TFGUARD-EBS-001` | HIGH | Unencrypted EBS volume |

Optional external scanners (require `--dir` pointing to Terraform HCL), merged into the same finding model:

- **Checkov** — broad IaC policy engine
- **tfsec** — Terraform security scanner

**Gate mapping.** `--fail-on` evaluates change risk **and** finding severity on one scale:

| Finding severity | Equivalent risk level |
|---|---|
| `CRITICAL` | `CRITICAL` |
| `HIGH` | `DANGER` |
| `MEDIUM` | `CAUTION` |
| `LOW` / unknown | `SAFE` |

**Adding a policy.** Create `policies/<name>.rego` with `package tfguard.<name>` and a `violation contains finding` rule returning `id`, `severity`, `title`, `description` and `resource`. It is embedded and evaluated automatically on the next build.

---

## Cost estimation

Static **us-east-1 on-demand** prices at ~730 h/month: 28 instance sizes (EC2, RDS, ElastiCache, OpenSearch/Elasticsearch, Redshift), per-GB storage (RDS, EBS) and flat-rate resources (NAT Gateway, ALB, EIP).

For each mutating change, tfguard:
1. Decodes `before` / `after` JSON attributes
2. Extracts billable fields (`instance_type`, `instance_class`, `allocated_storage`, …)
3. Computes monthly USD before, after, and delta
4. Shows the **top 3 drivers** by absolute delta

```bash
tfguard scan --plan plan.json --max-cost-increase 50 --no-ai
```

Supported pricing signals:

| Resource | Attributes read |
|----------|-----------------|
| `aws_instance` | `instance_type` |
| `aws_db_instance` | `instance_class`, `allocated_storage` |
| `aws_ebs_volume` | `size` |
| `aws_nat_gateway`, `aws_lb`, `aws_alb`, `aws_eip` | fixed monthly |
| ElastiCache / Redshift | `node_type`, node count |
| OpenSearch / Elasticsearch | `instance_type` or `cluster_config.instance_type` |

> **Note:** figures are a **review signal**, not a billing forecast. Usage-driven resources (S3, DynamoDB, …) are reported as *unpriced* rather than guessed.

---

## ML risk score

A **logistic regression** model gives the probability that a plan is high-risk, complementing the rule-based levels.

| Item | Detail |
|------|--------|
| Dataset | `scripts/generate_dataset.py` → `data/risk_dataset.json` (250 synthetic plans, seed 42) |
| Training | `scripts/train_model.py` (stdlib gradient descent, 80/20 split) → `internal/mlscore/model.json` |
| Inference | Coefficients embedded via `go:embed`; no Python at runtime |
| Features (10) | creates, updates, replaces, deletes, stateful mutations, critical findings, high findings, cost delta, max action level, change count |
| Hold-out metrics | **precision 1.00 · recall 0.98 · F1 0.99** ([`data/model_metrics.json`](data/model_metrics.json)) |

> **Caveat.** Labels come from a weighted heuristic over the same features, and the hold-out set is small (50 plans, 5 negatives). The metrics show the model reproduces that heuristic; they are not evidence of real-world accuracy. Training on labeled production plans is on the roadmap.

Retrain the model:

```bash
python3 scripts/generate_dataset.py
python3 scripts/train_model.py
make test
```

Commit the regenerated `internal/mlscore/model.json` and `data/model_metrics.json`.

---

## AI explainer (optional)

When **Ollama** is running locally, tfguard sends a compact JSON context (changes + findings + cost drivers — never your HCL) to `/api/generate` and prints:

- `summary` — 2–3 sentence overview
- `risks` — bullet list (max 5)
- `recommendations` — bullet list (max 5)
- `cost_note` — one sentence on cost impact

How it works:

1. The scan context is serialized and hashed with **SHA-256**.
2. A cached answer is reused from the user cache directory (`<user-cache>/tfguard/explain/`, e.g. `~/.cache` on Linux, `%LocalAppData%` on Windows).
3. Otherwise Ollama is called with a **JSON Schema** response format, and the output is validated before display.
4. If Ollama is unavailable, tfguard prints a warning and still renders the report.

```bash
ollama serve          # if not running
ollama pull llama3.2
tfguard scan --plan plan.json
```

The LLM **explains**; it never decides an exit code. For CI, use `--no-ai`.

---

## CI/CD integration

tfguard is a single binary with stable exit codes; call it after `terraform show -json` and let the exit code fail the job.

```yaml
- name: Terraform Plan
  run: |
    terraform plan -out=plan.tfplan
    terraform show -json plan.tfplan > plan.json

- name: tfguard scan
  run: |
    tfguard scan \
      --plan plan.json \
      --dir ./infra \
      --fail-on DANGER \
      --max-cost-increase 50 \
      --no-ai

- name: Terraform Apply
  if: success()
  run: terraform apply plan.tfplan
```

---

## GitHub Action

The repository ships a composite action (`action.yml`) that builds tfguard, runs the scan, and **comments the report on the PR**.

```yaml
name: Terraform Plan Review

on:
  pull_request:

permissions:
  contents: read
  pull-requests: write   # required for the PR comment

jobs:
  tfguard:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: "1.26.x"

      - name: Setup Terraform
        uses: hashicorp/setup-terraform@v3

      - name: Terraform Plan
        working-directory: infra
        run: |
          terraform init
          terraform plan -out=plan.tfplan
          terraform show -json plan.tfplan > ../plan.json

      - name: tfguard scan
        uses: ./
        with:
          plan-path: plan.json
          terraform-dir: infra
          fail-on: DANGER
          max-cost-increase: "50"
          no-ai: "true"
```

| Input | Default | Description |
|---|---|---|
| `plan-path` | — | Plan JSON (**required**) |
| `terraform-dir` | `""` | HCL directory for Checkov/tfsec |
| `fail-on` | `""` | Risk threshold |
| `max-cost-increase` | `""` | USD/month ceiling |
| `no-ai` | `"true"` | Skip the LLM explainer |

---

## Project structure

```text
tfguard/
├── cmd/tfguard/          CLI entrypoint (scan, version)
├── internal/
│   ├── tfplan/           Parse plan JSON + summarize
│   ├── risk/             SAFE → CRITICAL classification
│   ├── policy/           OPA + Checkov + tfsec
│   ├── cost/             Static AWS cost delta
│   ├── mlscore/          Embedded ML model + inference
│   ├── explain/          Ollama LLM explainer + cache
│   ├── scan/             Pipeline orchestration + CI gates
│   ├── render/           Terminal report formatting
│   └── ui/               Colors, boxes, banner, spinner
├── policies/             Embedded Rego rules (.rego)
├── scripts/              Dataset generation, model training, demo GIF
├── testdata/             Plan / scanner fixtures + Terraform fix cases (fix/)
├── docs/
│   ├── demo.gif          Animated CLI demo
│   └── images/           Screenshots (add yours here)
├── data/                 ML dataset + metrics
├── action.yml            GitHub Action composite
├── .goreleaser.yaml      Multi-platform release config
└── .github/workflows/    CI + release pipelines
```

Module path: `github.com/SamyBaouche/tfguard`

---

## Development

```bash
make test      # go test ./... -cover
make vet       # go vet ./...
make fmt       # go fmt ./...
make build     # build bin/tfguard
make clean     # remove bin/
```

**Testing approach**

- Table-driven unit tests with JSON fixtures in `testdata/` — no live Terraform, cloud or LLM calls
- External tools (Checkov, tfsec, Ollama) are tested through their parsed output or a fake HTTP server
- Core packages are close to fully covered (`risk` 100%, `tfplan` ~98%, `mlscore` ~93%)

**Guidelines**

- Wrap errors with `%w`
- Optional tools (Checkov, tfsec, Ollama) soft-fail with warnings
- Packages stay under `internal/` until a public API is intentional

**Releasing.** Pushing a `v*` tag runs GoReleaser (`.github/workflows/release.yml`): tests, then archives for linux/darwin/windows × amd64/arm64 with checksums.

```bash
git tag v0.2.0 && git push origin v0.2.0
```

**Regenerate the demo GIF**

```bash
python3 -m venv .venv
.venv/bin/pip install pillow
.venv/bin/python scripts/generate_demo_gif.py
```

---

## Design decisions

- **Plan JSON, not HCL, as the source of truth.** The plan reflects resolved variables, modules and provider defaults — what will actually happen.
- **Deterministic core, optional AI.** Gates rely only on rules and the embedded model. The LLM explains; it never decides an exit code.
- **Soft-fail integrations.** A missing Checkov, tfsec or Ollama degrades the report instead of breaking CI.
- **Self-contained binary.** Policies and model weights are embedded with `go:embed`; no runtime Python or OPA server.
- **Local-first LLM.** Infrastructure data stays on the machine; no third-party API keys.
- **`internal/` by default.** No public Go API is promised until one is intentionally designed.

---

## Limitations

- AWS only: other providers are parsed but get base-level risk and no pricing.
- Prices are static `us-east-1` on-demand values.
- The ML model is trained on synthetic data (see [ML risk score](#ml-risk-score)).
- The composite action builds from `./cmd/tfguard`, so it is currently meant to run inside this repository.

---

## Roadmap

| Status | Item |
|--------|------|
| ✅ Done | Plan parse, risk, policies, CLI, cost delta, ML score, Ollama explainer |
| ✅ Done | GitHub Action, GoReleaser v0.1.0, demo GIF, LICENSE, CONTRIBUTING |
| 🚧 In progress | **AI auto-fix** — `tfguard fix` repairs invalid Terraform with a local code LLM, a `terraform validate` feedback loop and provider-schema retrieval (RAG); test cases in `testdata/fix/` |
| 🔜 Planned | Evaluation harness for LLM fixes (success rate, minimal diff, latency) |
| 🔜 Planned | Reusable GitHub Action for external repositories |
| 🔜 Planned | Richer ML dataset (real Terraform plans) |
| 🔜 Planned | More AWS SKUs + dynamic pricing API |
| 🔜 Planned | `examples/` mini Terraform project for end-to-end testing |

---

## Contributing

Contributions are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md) for setup, conventions and the pull request checklist (`make test && make vet && make fmt`).

## License

Apache 2.0 — see [LICENSE](LICENSE).
