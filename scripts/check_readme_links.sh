#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root_dir"

python3 - <<'PY'
import pathlib
import re
import sys

files = ["README.md", "README_zh.md"]
link_re = re.compile(r"\[[^\]]+\]\(([^)]+)\)")
code_fence_re = re.compile(r"```.*?```", re.S)
inline_code_re = re.compile(r"`[^`]*`")

failed = False

for rel in files:
    path = pathlib.Path(rel)
    text = path.read_text(encoding="utf-8")
    text = code_fence_re.sub("", text)
    text = inline_code_re.sub("", text)

    for target in link_re.findall(text):
        target = target.strip()
        if not target or target.startswith(("http://", "https://", "mailto:", "#")):
            continue
        target = target.split("#", 1)[0].split("?", 1)[0].strip()
        if not target:
            continue
        resolved = (path.parent / target).resolve()
        if not resolved.exists():
            print(f"Broken local link in {rel} -> {target}")
            failed = True

if failed:
    sys.exit(1)

print("README local links are valid.")
PY
