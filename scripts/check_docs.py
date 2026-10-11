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


def main():
    root = Path(__file__).resolve().parent.parent
    names = subprocess.check_output(
        ["git", "ls-files", "-z", "--", "*.md"], cwd=root
    ).decode().split("\0")
    errors = []
    checked = 0
    for name in filter(None, names):
        path = root / name
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
    if errors:
        print("\n".join(errors), file=sys.stderr)
        return 1
    print(f"Checked {checked} local documentation links and anchors.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
