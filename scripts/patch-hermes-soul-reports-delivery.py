#!/usr/bin/env python3
"""SOUL: CSV/files must live under reports/; never claim attachment without MEDIA."""
from __future__ import annotations

from pathlib import Path

SOUL = Path("/root/.hermes/SOUL.md")
MARKER = "## FILE DELIVERY — WhatsApp (system instruction)"
ANCHOR = "## WEB SCRAPING — Scrapling"

BLOCK = """
## FILE DELIVERY — WhatsApp (system instruction)

### Where files must live
- **All** owner deliverables (CSV, MD, PDF, JSON exports): `~/.hermes/reports/<name>.<ext>` only.
- **Never** write `~/.hermes/foo.csv` at the Hermes root — WhatsApp will **not** attach it (security block).

### How to attach
- End the turn with a line: `MEDIA:/root/.hermes/reports/<filename>.csv` (absolute path under **reports/**).
- Or use `send_message` / platform document send with the **reports/** path if the tool supports file upload.

### Honesty rule (critical)
- **Never** say “Sent the CSV”, “attached”, or “check your WhatsApp for the file” unless the file was actually delivered via `MEDIA:` or a successful document send in **this** turn.
- If attachment failed or path was wrong: say so, give the **reports/** path, and offer to resend — do not claim success.

### After writing a file
1. `ls -la ~/.hermes/reports/<file>` to confirm it exists.
2. Include `MEDIA:/root/.hermes/reports/<file>` in your final reply.
3. Short summary in chat — not the full CSV body.

"""


def main() -> None:
    text = SOUL.read_text(encoding="utf-8")
    if MARKER in text:
        print("ok: file delivery section already present")
        return
    if ANCHOR in text:
        text = text.replace(ANCHOR, BLOCK.strip() + "\n\n" + ANCHOR, 1)
    else:
        anchor = "## AUTONOMOUS AGENT — owners (system instruction)"
        if anchor not in text:
            raise SystemExit("SOUL anchor not found")
        text = text.replace(anchor, BLOCK.strip() + "\n\n" + anchor, 1)
    SOUL.write_text(text, encoding="utf-8")
    print("ok: SOUL file delivery rules added")


if __name__ == "__main__":
    main()
