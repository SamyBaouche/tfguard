#!/usr/bin/env python3
"""Generate docs/demo.gif from a captured tfguard scan (no external tools)."""

from __future__ import annotations

import subprocess
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont

ROOT = Path(__file__).resolve().parent.parent
OUT = ROOT / "docs" / "demo.gif"
FONT = ImageFont.load_default()
BG = (24, 24, 32)
FG = (220, 224, 232)
ACCENT = (137, 180, 250)
DIM = (120, 130, 150)
W, H = 920, 560
PAD = 16
LINE = 14


def capture() -> list[str]:
    cmd = [
        str(ROOT / "bin" / "tfguard"),
        "scan",
        "--plan",
        str(ROOT / "testdata" / "plan_mixed.json"),
        "--no-ai",
        "--no-banner",
    ]
    env = {"NO_COLOR": "1", "PATH": "/usr/bin:/bin"}
    out = subprocess.check_output(cmd, cwd=ROOT, env=env, stderr=subprocess.DEVNULL, text=True)
    return out.splitlines()


def render(lines: list[str]) -> Image.Image:
    img = Image.new("RGB", (W, H), BG)
    draw = ImageDraw.Draw(img)
    draw.rectangle((0, 0, W, 28), fill=(40, 40, 52))
    draw.text((PAD, 8), "tfguard scan --plan plan.json --no-ai", fill=ACCENT, font=FONT)
    y = 36
    for line in lines:
        if y > H - LINE:
            break
        color = FG
        if "CRITICAL" in line or "+7.59" in line:
            color = (240, 120, 120)
        elif "Cost estimate" in line or "Scan report" in line:
            color = ACCENT
        elif line.strip().startswith("▸") or "──" in line:
            color = DIM
        draw.text((PAD, y), line[:110], fill=color, font=FONT)
        y += LINE
    return img


def typing_frames(full: list[str]) -> list[Image.Image]:
    frames: list[Image.Image] = []
    for end in range(8, len(full) + 1, 3):
        frames.append(render(full[:end]))
        frames.extend([frames[-1]] * 2)
    frames.append(render(full))
    frames.extend([frames[-1]] * 12)
    return frames


def main() -> None:
    subprocess.run(
        ["go", "build", "-ldflags=-X main.Version=0.1.0", "-o", "bin/tfguard", "./cmd/tfguard"],
        cwd=ROOT,
        check=True,
    )
    lines = capture()
    OUT.parent.mkdir(parents=True, exist_ok=True)
    frames = typing_frames(lines)
    frames[0].save(
        OUT,
        save_all=True,
        append_images=frames[1:],
        duration=120,
        loop=0,
        optimize=True,
    )
    print(f"wrote {OUT} ({len(frames)} frames)")


if __name__ == "__main__":
    main()
