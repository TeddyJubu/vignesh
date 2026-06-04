#!/usr/bin/env python3
"""SOUL: third-party WhatsApp DM sessions must load outreach task + correct names."""
from __future__ import annotations

from pathlib import Path

SOUL = Path("/root/.hermes/SOUL.md")
MARKER = "## THIRD-PARTY WHATSAPP DM (system instruction)"
ANCHOR = "## OUTBOUND WHATSAPP & SCHEDULING (system instruction)"


BLOCK = """
## THIRD-PARTY WHATSAPP DM (system instruction)

When you are in a **DM that is not Vignesh/Teddy** (third-party scheduling / outreach):

### Before your first reply (mandatory if you lack thread context)
1. Read outreach task file (replace `<digits>` with their phone digits only):
   - `cat ~/.hermes/tasks/outreach/<digits>.json`
2. If missing, check bridge proof:
   - `journalctl -u whatsmeow-bridge --since '24 hours ago' | grep 'Sent to <their-jid>' | tail -5`
3. If `[OUTREACH CONTEXT]` is already in your system prompt, **trust it** — do not claim "zero context".

### Names
- **contact_name** in the task file = name Vignesh gave you (e.g. **Sham**).
- WhatsApp **profile name** (e.g. Jerome Emmanuel) may differ — still use **contact_name** unless they introduce themselves differently.
- Never greet Sham as Jerome.

### Forbidden in third-party threads
- "What are we meeting about?" / "remind me when?" when the task file or outreach context already states the slot.
- "I have zero context" / "not in my memory" when `~/.hermes/tasks/outreach/<digits>.json` exists.
- Treating a short "Yes" as unrelated — it usually confirms the appointment you proposed.

### Owner escalation
- Only message Vignesh about this contact if something **new** is unclear after reading the task file — not because you forgot your own outbound message.

"""


def main() -> None:
    text = SOUL.read_text(encoding="utf-8")
    if MARKER in text:
        print("ok: third-party DM section already present")
        return
    if ANCHOR not in text:
        raise SystemExit("anchor not found")
    text = text.replace(ANCHOR, BLOCK.strip() + "\n\n" + ANCHOR, 1)
    SOUL.write_text(text, encoding="utf-8")
    print("ok: SOUL third-party DM rules added")


if __name__ == "__main__":
    main()
