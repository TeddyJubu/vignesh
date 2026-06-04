#!/usr/bin/env python3
"""SOUL: third-party WhatsApp sends + SGT scheduling (no false sent, no wrong tomorrow)."""
from __future__ import annotations

from pathlib import Path

SOUL = Path("/root/.hermes/SOUL.md")
MARKER = "## OUTBOUND WHATSAPP & SCHEDULING (system instruction)"
ANCHOR = "## WHATSAPP ACCESS & CROSS-CHAT (system instruction)"


BLOCK = """
## OUTBOUND WHATSAPP & SCHEDULING (system instruction)

### Not a bridge bug
- **whatsmeow-bridge** only delivers what Hermes asks it to send. If someone did not get a DM, the failure is almost always: no successful `send_message` tool call, wrong target string, or a reply only sent to the owner.

### Messaging someone else (e.g. "message +65 … for an appointment")
1. **Always** call `send_message` with target **`whatsapp:+<E.164>`** (e.g. `whatsapp:+6588691488`) or **`whatsapp:<digits>@s.whatsapp.net`**.
   - **Never** use the JID as the platform (`6588691488@s.whatsapp.net` alone → tool error).
   - **Never** use bare `whatsapp` when the owner asked you to text a third party.
2. **Before** telling the owner "sent", "delivered", or "asked them":
   - Confirm the tool returned **success** in **this** turn (not a plan, not memory).
   - Optionally verify: `journalctl -u whatsmeow-bridge --since '5 min ago' | grep 'Sent to <their-jid>' | tail -3`
3. If `send_message` failed or you have not called it yet: say that honestly — do not claim delivery.

### Dates and times (Singapore default)
- **Timezone:** `Asia/Singapore` (SGT, UTC+8). Vignesh and Teddy work in SGT unless they say otherwise.
- **Before** writing "tomorrow", "today", "Thursday", or a calendar date in an outbound message, run:
  - `TZ=Asia/Singapore date '+%A %Y-%m-%d %H:%M %Z'`
- **Tomorrow** = calendar day **after** the date that command prints as today — never label today's date as "tomorrow".
- In messages to third parties, include: weekday + date + time + **SGT** (e.g. "Friday 5 June 2026, 9pm SGT").
- After session compression, **re-run** `date` if you are scheduling — do not rely on memory for "what day is it".

"""


def main() -> None:
    text = SOUL.read_text(encoding="utf-8")
    if MARKER in text:
        print("ok: outbound messaging section already present")
        return
    if ANCHOR not in text:
        raise SystemExit("anchor not found — apply whatsapp-access-crosschat patch first")
    text = text.replace(ANCHOR, BLOCK.strip() + "\n\n" + ANCHOR, 1)
    SOUL.write_text(text, encoding="utf-8")
    print("ok: SOUL outbound + SGT scheduling rules added")


if __name__ == "__main__":
    main()
