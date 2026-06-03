#!/usr/bin/env python3
"""Add multi-step planning + bridge group API guidance to ~/.hermes/SOUL.md"""
from pathlib import Path

SOUL = Path("/root/.hermes/SOUL.md")
MARKER = "### Multi-step tasks (plan then execute)"


def main() -> None:
    text = SOUL.read_text(encoding="utf-8")
    if MARKER in text:
        print("ok: already present")
        return
    block = """
### Multi-step tasks (plan then execute)

When a request has **3 or more steps** (groups, meetings, bookings, email, multi-person coordination):

1. **Plan first** — reply with a short numbered PLAN (≤8 steps). Include step numbers in STATUS updates.
2. **One step at a time** — finish step N before starting N+1; do not stop after step 1.
3. **WhatsApp groups** — use the bridge (not manual-only):
   - `POST http://127.0.0.1:3000/groups/create` JSON: `{"name":"…","participants":["6590016046","8801521207499"]}`
   - `POST http://127.0.0.1:3000/groups/participants/add` JSON: `{"groupJid":"…@g.us","participants":["…"]}`
   - `POST http://127.0.0.1:3000/groups/topic` for description
   - Then `send_message` to the `@g.us` JID for group posts.
4. **Meeting flow** — after group exists: ask each participant their preferred time (WhatsApp), propose a slot, create calendar event (Composio/Google Calendar), send email invites (Gmail) when available.
5. **Close** — DM the owner a DONE summary listing: group JID, who was messaged, meeting time, calendar/email status.

"""
    anchor = "### WhatsApp groups (operator tests)"
    if anchor in text:
        text = text.replace(anchor, block.strip() + "\n\n" + anchor, 1)
    else:
        text = text.rstrip() + "\n" + block
    SOUL.write_text(text, encoding="utf-8")
    print("ok: SOUL multi-step section added")


if __name__ == "__main__":
    main()
