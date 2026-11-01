#!/usr/bin/env python3
"""Explain the English spell check without altering any generated findings."""

import sys
from pathlib import Path

NOTE = (
    "> `misspell` checks English. Suggestions for translated documentation and "
    "locale catalogs need language-specific review; the raw findings below are "
    "retained. See [report notes](quality-report-notes.md).\n\n"
)


def annotate(path: Path) -> None:
    report = path.read_text(encoding="utf-8")
    heading = "# Go Report Card\n\n"
    if not report.startswith(heading):
        raise ValueError("unexpected Go Report Card heading")
    if report.startswith(heading + NOTE):
        return
    path.write_text(heading + NOTE + report[len(heading):], encoding="utf-8")


if __name__ == "__main__":
    annotate(Path(sys.argv[1]))
