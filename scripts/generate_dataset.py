#!/usr/bin/env python3
"""Generate a labeled dataset of synthetic Terraform plan features for risk scoring."""

from __future__ import annotations

import json
import random
from pathlib import Path

ACTIONS = ["create", "update", "replace", "delete"]
STATEFUL = {"aws_db_instance", "aws_s3_bucket", "aws_ebs_volume", "aws_dynamodb_table"}
TYPES = list(STATEFUL) + ["aws_instance", "aws_security_group", "aws_subnet", "aws_iam_role"]

# label=1 means "high risk" plan worth blocking in CI
def label_row(features: dict) -> int:
    score = 0.0
    score += features["deletes"] * 2.5
    score += features["replaces"] * 2.0
    score += features["stateful_mutations"] * 1.5
    score += features["critical_findings"] * 3.0
    score += features["high_findings"] * 2.0
    score += features["cost_delta_usd"] / 25.0
    score += features["max_action_level"] * 0.8
    return 1 if score >= 4.0 else 0


def random_plan(n_changes: int, rng: random.Random, force_low: bool = False) -> dict:
    changes = []
    for _ in range(n_changes):
        if force_low:
            rtype = rng.choice(["aws_instance", "aws_subnet", "aws_iam_role"])
            action = rng.choice(["create", "update"])
        else:
            rtype = rng.choice(TYPES)
            action = rng.choice(ACTIONS)
        changes.append({"type": rtype, "action": action})

    deletes = sum(1 for c in changes if c["action"] == "delete")
    replaces = sum(1 for c in changes if c["action"] == "replace")
    updates = sum(1 for c in changes if c["action"] == "update")
    creates = sum(1 for c in changes if c["action"] == "create")
    stateful_mutations = sum(
        1 for c in changes if c["type"] in STATEFUL and c["action"] in {"delete", "replace", "update"}
    )

    action_level = {"create": 0, "update": 1, "replace": 2, "delete": 2}
    max_action_level = max(action_level[c["action"]] for c in changes) if changes else 0

    critical_findings = rng.randint(0, 2) if stateful_mutations else rng.randint(0, 1)
    high_findings = rng.randint(0, 3)
    cost_delta = round(rng.uniform(-20, 120), 2)

    features = {
        "creates": creates,
        "updates": updates,
        "replaces": replaces,
        "deletes": deletes,
        "stateful_mutations": stateful_mutations,
        "critical_findings": critical_findings,
        "high_findings": high_findings,
        "cost_delta_usd": cost_delta,
        "max_action_level": max_action_level,
        "change_count": n_changes,
    }
    features["label"] = label_row(features)
    return features


def main() -> None:
    rng = random.Random(42)
    rows = []
    for i in range(250):
        n = rng.randint(1, 12)
        force_low = i % 2 == 0
        rows.append(random_plan(n, rng, force_low=force_low))

    out = Path(__file__).resolve().parent.parent / "data" / "risk_dataset.json"
    out.parent.mkdir(parents=True, exist_ok=True)
    out.write_text(json.dumps(rows, indent=2), encoding="utf-8")
    pos = sum(r["label"] for r in rows)
    print(f"wrote {len(rows)} rows to {out} ({pos} high-risk)")


if __name__ == "__main__":
    main()
