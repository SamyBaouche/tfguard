#!/usr/bin/env python3
"""Train a logistic regression risk scorer (stdlib) and export coefficients for Go."""

from __future__ import annotations

import json
import math
import random
from pathlib import Path

FEATURES = [
    "creates",
    "updates",
    "replaces",
    "deletes",
    "stateful_mutations",
    "critical_findings",
    "high_findings",
    "cost_delta_usd",
    "max_action_level",
    "change_count",
]

ROOT = Path(__file__).resolve().parent.parent
DATA = ROOT / "data" / "risk_dataset.json"
OUT = ROOT / "internal" / "mlscore" / "model.json"
METRICS = ROOT / "data" / "model_metrics.json"


def sigmoid(x: float) -> float:
    if x >= 0:
        z = math.exp(-x)
        return 1.0 / (1.0 + z)
    z = math.exp(x)
    return z / (1.0 + z)


def train(rows: list[dict], epochs: int = 800, lr: float = 0.05) -> tuple[list[float], float]:
    rng = random.Random(0)
    w = [0.0 for _ in FEATURES]
    b = 0.0
    for _ in range(epochs):
        for row in rows:
            x = [row[f] for f in FEATURES]
            y = float(row["label"])
            z = b + sum(wi * xi for wi, xi in zip(w, x))
            p = sigmoid(z)
            err = p - y
            for i in range(len(w)):
                w[i] -= lr * err * x[i]
            b -= lr * err
        rng.shuffle(rows)
    return w, b


def metrics(rows: list[dict], w: list[float], b: float) -> dict:
    tp = fp = fn = tn = 0
    for row in rows:
        x = [row[f] for f in FEATURES]
        z = b + sum(wi * xi for wi, xi in zip(w, x))
        pred = 1 if sigmoid(z) >= 0.5 else 0
        y = row["label"]
        if pred == 1 and y == 1:
            tp += 1
        elif pred == 1 and y == 0:
            fp += 1
        elif pred == 0 and y == 1:
            fn += 1
        else:
            tn += 1
    precision = tp / (tp + fp) if tp + fp else 0.0
    recall = tp / (tp + fn) if tp + fn else 0.0
    f1 = 2 * precision * recall / (precision + recall) if precision + recall else 0.0
    return {
        "precision": round(precision, 4),
        "recall": round(recall, 4),
        "f1": round(f1, 4),
        "tp": tp,
        "fp": fp,
        "fn": fn,
        "tn": tn,
    }


def main() -> None:
    rows = json.loads(DATA.read_text(encoding="utf-8"))
    split = int(len(rows) * 0.8)
    train_rows = rows[:split]
    test_rows = rows[split:]

    w, b = train(train_rows)
    m = metrics(test_rows, w, b)
    m["train_size"] = len(train_rows)
    m["test_size"] = len(test_rows)

    payload = {
        "features": FEATURES,
        "intercept": b,
        "weights": w,
    }
    OUT.parent.mkdir(parents=True, exist_ok=True)
    OUT.write_text(json.dumps(payload, indent=2), encoding="utf-8")
    METRICS.write_text(json.dumps(m, indent=2), encoding="utf-8")
    print(json.dumps(m, indent=2))
    print(f"model -> {OUT}")


if __name__ == "__main__":
    main()
