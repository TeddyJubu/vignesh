#!/usr/bin/env python3
"""Backfill Honcho + memory markdown from existing outreach task JSON files.

Run with Hermes venv Python (honcho-ai is installed there):
  /usr/local/lib/hermes-agent/venv/bin/python backfill-outreach-honcho.py
"""
from __future__ import annotations

import json
import sys
from pathlib import Path

SCRIPTS = Path("/root/.hermes/scripts")
sys.path.insert(0, "/usr/local/lib/hermes-agent")
sys.path.insert(0, str(SCRIPTS))

from outreach_tasks import sync_outreach_to_honcho, write_outreach_memory_md  # noqa: E402


def main() -> None:
    tasks_dir = Path("/root/.hermes/tasks/outreach")
    if not tasks_dir.is_dir():
        print("no outreach tasks dir")
        return
    n = 0
    for path in sorted(tasks_dir.glob("*.json")):
        try:
            task = json.loads(path.read_text(encoding="utf-8"))
        except (json.JSONDecodeError, OSError):
            continue
        write_outreach_memory_md(task)
        if sync_outreach_to_honcho(task):
            n += 1
            print(f"ok: {path.name}")
        else:
            print(f"skip honcho: {path.name}")
    print(f"done: {n} synced")


if __name__ == "__main__":
    main()
