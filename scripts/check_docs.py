"""Check local inline Markdown/HTML links and heading anchors, without network I/O."""

from pathlib import Path
import re
import subprocess
import sys
from urllib.parse import unquote, urlsplit


def anchors(text):
    result = set()
    counts = {}
    for heading in re.findall(r"^#{1,6}\s+(.+?)\s*#*\s*$", text, re.MULTILINE):
        slug = re.sub(r"[^\w\- ]", "", heading.lower()).replace(" ", "-")
        count = counts.get(slug, 0)
        counts[slug] = count + 1
        result.add(slug if count == 0 else f"{slug}-{count}")
    return result


def check_taskfile(root):
    taskfile = (root / "Taskfile.yml").read_text()
    task_names = set(re.findall(r"^  ([a-zA-Z0-9_:-]+):\s*$", taskfile, re.MULTILINE))
    task_names.discard("default")
    expected = set()
    makefile_map = [
        (root / "Makefile", ""),
        (root / "bubbletea" / "Makefile", "bubbletea:"),
        (root / "web" / "Makefile", "web:"),
        (root / "devices" / "terminal" / "Makefile", "terminal:"),
    ]
    for path, prefix in makefile_map:
        text = path.read_text()
        for line in re.findall(r"^\.PHONY:\s*(.+)$", text, re.MULTILINE):
            for target in line.split():
                expected.add(f"{prefix}{target}")
    errors = []
    for missing in sorted(expected - task_names):
        errors.append(f"Taskfile.yml: missing task '{missing}' for Makefile target")
    for extra in sorted(task_names - expected):
        errors.append(f"Taskfile.yml: task '{extra}' has no matching Makefile .PHONY target")
    return errors, len(expected)


def main():
    root = Path(__file__).resolve().parent.parent
    names = subprocess.check_output(
        ["git", "ls-files", "-z", "--cached", "--others", "--exclude-standard", "--", "*.md"], cwd=root
    ).decode().split("\0")
    errors = []
    checked = 0
    for name in sorted(set(filter(None, names))):
        path = root / name
        if not path.is_file():
            continue
        text = path.read_text()
        if text.count("```") % 2:
            errors.append(f"{name}: unmatched code fence")
        prose = re.sub(r"```[^\n]*\n.*?```", "", text, flags=re.DOTALL)
        links = re.findall(r"\]\(([^\s)]+)\)", prose)
        links += re.findall(r'(?:href|src)="([^"]+)"', prose)
        for link in links:
            url = urlsplit(link)
            if url.scheme or url.netloc:
                continue
            checked += 1
            target = path.parent / unquote(url.path) if url.path else path
            if not target.exists():
                errors.append(f"{name}: missing {link}")
            elif url.fragment and target.suffix == ".md":
                if unquote(url.fragment) not in anchors(target.read_text()):
                    errors.append(f"{name}: missing heading in {link}")
    task_errors, task_count = check_taskfile(root)
    errors.extend(task_errors)
    if errors:
        print("\n".join(errors), file=sys.stderr)
        return 1
    print(f"Checked {checked} local documentation links/anchors and {task_count} Makefile/Taskfile targets.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
