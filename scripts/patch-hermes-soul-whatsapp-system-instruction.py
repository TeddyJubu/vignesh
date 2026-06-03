#!/usr/bin/env python3
"""Prominent WhatsApp system instruction in SOUL.md — wabot/7777 ban + production stack."""
from __future__ import annotations

import re
from pathlib import Path

SOUL = Path("/root/.hermes/SOUL.md")
MARKER = "## WHATSAPP — production stack (system instruction)"


BLOCK = """
## WHATSAPP — production stack (system instruction)

**Always apply.** Do not wait for the user to repeat this in their message.

### Active stack (only)
- **Messaging:** whatsmeow-bridge `http://127.0.0.1:3000` (`/send`, `/groups/create`, `/groups/participants/add`, `/groups/topic`, `/inject` for tests)
- **Agent:** hermes-gateway (Stella)

### Forbidden — never use, start, recommend, or “fix”
- **wabot**, **wabot-agent**, **port 7777**, **`/opt/wabot`**, **`/usr/local/bin/wabot`**
- `curl` / health checks to `:7777`
- `ps aux | grep wabot`, kill + nohup restart of legacy daemons
- Telling users wabot is “down” or to re-enable the old stack

Legacy stacks are **archived** under `/opt/archive/whatsapp-legacy-20260603/` (Jun 2026) — duplicate replies if revived. If `systemctl` mentions the old agent name, **ignore**; do not try to repair it.

### Groups
- Create via `POST /groups/create` on port **3000**, then `send_message` to the `@g.us` JID.
- Owner DM = PLAN / STATUS / DONE only; **group content** goes to the group JID, not the owner DM.

"""


def remove_stale_lines(text: str) -> str:
    stale = [
        r"- \*\*Groups:\*\* no HTTP group-creation API[^\n]+\n",
        r"- \*\*Cannot create groups via API\.\*\*[^\n]+\n",
        r"- \*\*Posting to a group:\*\* use `send_message`[^\n]+legacy ports[^\n]+\n",
    ]
    for pat in stale:
        text = re.sub(pat, "", text)
    return text


def dedupe_legacy_footer(text: str) -> str:
    """Keep one legacy ban block; remove duplicate footer section."""
    if text.count("### Legacy duplicate WhatsApp stack") > 1:
        first = text.find("### Legacy duplicate WhatsApp stack")
        second = text.find("### Legacy duplicate WhatsApp stack", first + 1)
        if second >= 0:
            text = text[:second].rstrip() + "\n"
    return text


def main() -> None:
    text = SOUL.read_text(encoding="utf-8")
    text = remove_stale_lines(text)
    text = dedupe_legacy_footer(text)

    if MARKER not in text:
        anchor = "## EXTERNAL CONTACT RULES — CRITICAL"
        if anchor in text:
            text = text.replace(anchor, BLOCK.strip() + "\n\n" + anchor, 1)
        else:
            text = BLOCK.strip() + "\n\n" + text
        SOUL.write_text(text, encoding="utf-8")
        print("ok: WhatsApp system instruction added to SOUL.md")
    else:
        # Refresh block content
        text = re.sub(
            rf"{re.escape(MARKER)}.*?(?=\n## EXTERNAL CONTACT|\n## SALES |\Z)",
            BLOCK.strip() + "\n\n",
            text,
            count=1,
            flags=re.S,
        )
        SOUL.write_text(text, encoding="utf-8")
        print("ok: WhatsApp system instruction refreshed in SOUL.md")


if __name__ == "__main__":
    main()
