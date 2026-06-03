#!/usr/bin/env python3
"""Deliver ~/.hermes/<file> via ~/.hermes/reports/<file> (allowed attach path)."""
from __future__ import annotations

import shutil
from pathlib import Path

BASE = Path("/usr/local/lib/hermes-agent/gateway/platforms/base.py")

MARKER = "# stella: hermes-home reports fallback for MEDIA delivery"
OLD_MARKER_BLOCK = """    # stella: hermes-home reports fallback for MEDIA delivery
    # Models often write CSV/MD to ~/.hermes/ instead of ~/.hermes/reports/.
    # Paths under /root are denied by default; redirect to reports/ sibling.
    _DELIVERABLE_SUFFIXES = {
        ".csv", ".md", ".txt", ".json", ".pdf", ".xlsx", ".xls", ".docx",
    }
    try:
        _hermes_home = _HERMES_HOME.expanduser().resolve(strict=False)
        if (
            resolved.parent == _hermes_home
            and resolved.suffix.lower() in _DELIVERABLE_SUFFIXES
        ):
            _reports_copy = _hermes_home / "reports" / resolved.name
            if _reports_copy.is_file():
                resolved = _reports_copy.resolve(strict=True)
    except (OSError, RuntimeError, ValueError):
        pass

    # Cache / operator allowlist is always honored — these are unconditionally"""

NEEDLE = """    if not resolved.is_file():
        return None

    # Cache / operator allowlist is always honored — these are unconditionally"""

FALLBACK = """    if not resolved.is_file():
        return None

    # stella: hermes-home reports fallback for MEDIA delivery
    # Models often write CSV/MD to ~/.hermes/ instead of ~/.hermes/reports/.
    # /root/* is denied for attachments; mirror into reports/ and deliver from there.
    _DELIVERABLE_SUFFIXES = {
        ".csv", ".md", ".txt", ".json", ".pdf", ".xlsx", ".xls", ".docx",
    }
    try:
        _hermes_home = _HERMES_HOME.expanduser().resolve(strict=False)
        if (
            resolved.parent == _hermes_home
            and resolved.suffix.lower() in _DELIVERABLE_SUFFIXES
        ):
            _reports_dir = _hermes_home / "reports"
            _reports_dir.mkdir(parents=True, exist_ok=True)
            _reports_copy = _reports_dir / resolved.name
            if not _reports_copy.exists() or _reports_copy.stat().st_mtime < resolved.stat().st_mtime:
                shutil.copy2(resolved, _reports_copy)
            resolved = _reports_copy.resolve(strict=True)
    except (OSError, RuntimeError, ValueError):
        pass

    # Cache / operator allowlist is always honored — these are unconditionally"""


def main() -> None:
    text = BASE.read_text(encoding="utf-8")
    if MARKER in text and "shutil.copy2" in text:
        print("ok: reports MEDIA fallback already patched (v2)")
        return
    if OLD_MARKER_BLOCK in text:
        text = text.replace(OLD_MARKER_BLOCK, FALLBACK, 1)
    elif NEEDLE in text:
        text = text.replace(NEEDLE, FALLBACK, 1)
    else:
        raise SystemExit("validate_media_delivery_path anchor missing")
    bak = BASE.with_suffix(".py.pre-media-fallback.bak")
    if not bak.exists():
        bak.write_text(BASE.read_text(encoding="utf-8"), encoding="utf-8")
    BASE.write_text(text, encoding="utf-8")
    print("ok: MEDIA delivery fallback ~/.hermes -> reports/ added (v2)")


if __name__ == "__main__":
    main()
