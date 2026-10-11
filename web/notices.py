#!/usr/bin/env python3
"""Preserve dependency notices alongside the generated static distribution."""
import json
from pathlib import Path
import subprocess
import sys

destination = Path(sys.argv[1]) / "THIRD_PARTY_NOTICES.txt"
parts = ["TinyBubbles browser demo — dependency license notices\n"]


def include(name, path):
    parts.append(f"\n{'=' * 72}\n{name}\n{path.name}\n{'=' * 72}\n")
    parts.append(path.read_text())


modules = subprocess.check_output(["go", "list", "-m", "-json", "all"], text=True)
decoder = json.JSONDecoder()
while modules.strip():
    module, end = decoder.raw_decode(modules.lstrip())
    modules = modules.lstrip()[end:]
    source = module.get("Replace", module)
    if not source.get("Dir"):
        continue
    directory = Path(source["Dir"])
    for path in sorted(directory.iterdir()):
        if path.is_file() and path.name.upper().startswith(("LICENSE", "COPYING", "NOTICE")):
            include(module["Path"] + " " + source.get("Version", "local"), path)

goroot = subprocess.check_output(["go", "env", "GOROOT"], text=True).strip()
include("Go runtime", Path(goroot) / "LICENSE")
for path in sorted(Path("licenses").glob("*.txt")):
    include("Browser dependency: " + path.stem, path)
if len(sys.argv) > 2:
    for relative in ("lib/compiler-rt-builtins/LICENSE.TXT", "lib/wasi-libc/libc-bottom-half/cloudlibc/LICENSE"):
        path = Path(sys.argv[2]) / relative
        if path.is_file():
            include("TinyGo bundled runtime: " + relative, path)
destination.write_text("\n".join(parts))
