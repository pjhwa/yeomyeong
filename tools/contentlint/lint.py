#!/usr/bin/env python3
"""Content linter for YEOMYEONG.

Checks:
  1. YAML files under content/ parse.
  2. Denylisted proper nouns (PLAN.md §2.1) do not appear in content/ or
     narrative docs (STORY-BIBLE, FORESHADOW, CONTENT-STYLE, lore).
  3. Foreshadow IDs referenced in YAML exist in docs/FORESHADOW.md (when
     that ledger has any registered rows).

Exits 0 when there is nothing to lint yet. That keeps the foundation CI green
before WORLD and NARRATIVE land files.
"""

from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
CONTENT = ROOT / "content"
DENYLIST = Path(__file__).resolve().parent / "denylist.txt"
FORESHADOW = ROOT / "docs" / "FORESHADOW.md"
NARRATIVE_DOCS = [
    ROOT / "docs" / "STORY-BIBLE.md",
    ROOT / "docs" / "FORESHADOW.md",
    ROOT / "docs" / "CONTENT-STYLE.md",
]


def load_denylist() -> list[str]:
    terms: list[str] = []
    if not DENYLIST.exists():
        return terms
    for raw in DENYLIST.read_text(encoding="utf-8").splitlines():
        line = raw.split("#", 1)[0].strip()
        if line:
            terms.append(line)
    return terms


def iter_lint_targets() -> list[Path]:
    files: list[Path] = []
    if CONTENT.exists():
        files.extend(p for p in CONTENT.rglob("*") if p.is_file() and not p.name.startswith("."))
    for doc in NARRATIVE_DOCS:
        if doc.exists():
            files.append(doc)
    return files


def check_yaml(files: list[Path]) -> list[str]:
    try:
        import yaml  # type: ignore
    except ImportError:
        return ["PyYAML is not installed; cannot parse content YAML."]

    errors: list[str] = []
    for path in files:
        if path.suffix not in {".yaml", ".yml"}:
            continue
        try:
            yaml.safe_load(path.read_text(encoding="utf-8"))
        except yaml.YAMLError as exc:
            errors.append(f"{path.relative_to(ROOT)}: YAML parse error: {exc}")
    return errors


def check_denylist(files: list[Path], terms: list[str]) -> list[str]:
    errors: list[str] = []
    for path in files:
        if path.suffix.lower() not in {".yaml", ".yml", ".md", ".txt"}:
            continue
        text = path.read_text(encoding="utf-8")
        for term in terms:
            if term in text:
                rel = path.relative_to(ROOT)
                errors.append(f"{rel}: forbidden proper noun {term!r} (PLAN.md §2.1)")
    return errors


def registered_foreshadow_ids() -> set[str]:
    if not FORESHADOW.exists():
        return set()
    text = FORESHADOW.read_text(encoding="utf-8")
    return set(re.findall(r"\bFS-\d{3,}\b", text))


def check_foreshadow_refs(files: list[Path], known: set[str]) -> list[str]:
    if not known:
        return []
    errors: list[str] = []
    for path in files:
        if path.suffix not in {".yaml", ".yml"}:
            continue
        text = path.read_text(encoding="utf-8")
        for match in re.findall(r"\bFS-\d{3,}\b", text):
            if match not in known:
                rel = path.relative_to(ROOT)
                errors.append(f"{rel}: foreshadow id {match} is not in docs/FORESHADOW.md")
    return errors


def main() -> int:
    files = iter_lint_targets()
    if not files:
        print("content-lint: nothing to lint yet.")
        return 0

    terms = load_denylist()
    errors: list[str] = []
    errors.extend(check_yaml(files))
    errors.extend(check_denylist(files, terms))
    errors.extend(check_foreshadow_refs(files, registered_foreshadow_ids()))

    if errors:
        print("content-lint: FAIL")
        for err in errors:
            print(f"  - {err}")
        return 1

    print(f"content-lint: OK ({len(files)} file(s), {len(terms)} denylist terms)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
