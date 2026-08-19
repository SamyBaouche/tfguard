# Contributing to tfguard

Thanks for helping improve tfguard.

## Development setup

```bash
git clone https://github.com/SamyBaouche/tfguard.git
cd tfguard
make test
make build
./bin/tfguard scan --plan testdata/plan_mixed.json --no-ai
```

## Pull requests

1. Keep changes focused and small.
2. Run `make test && make vet && make fmt` before opening a PR.
3. Add or update tests for behavior changes.
4. Update README when CLI flags or output change.

## ML model refresh

```bash
python3 scripts/generate_dataset.py
python3 scripts/train_model.py
make test
```

Commit the updated `internal/mlscore/model.json` and `data/model_metrics.json`.

## Code style

- Prefer table-driven tests and fixtures over live Terraform.
- Wrap errors with `%w`.
- Optional tools (Checkov, tfsec, Ollama) must soft-fail with warnings.

## Questions

Open a GitHub issue for bugs, feature ideas, or design questions.
