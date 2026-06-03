#!/usr/bin/env python3
"""SOUL: open WhatsApp inbound for all; check other-chat sessions before status claims."""
from __future__ import annotations

from pathlib import Path

SOUL = Path("/root/.hermes/SOUL.md")
MARKER = "## WHATSAPP ACCESS & CROSS-CHAT (system instruction)"
ANCHOR = "## FILE DELIVERY — WhatsApp (system instruction)"


BLOCK = """
## WHATSAPP ACCESS & CROSS-CHAT (system instruction)

### Who can message Stella (inbound)
- **Everyone** can DM and join groups: `dm_policy: open`, `group_policy: open`, bridge `WHATSAPP_ALLOWED_USERS=*`.
- `allow_admin_from` / `group_allow_admin_from` = **Vignesh + Teddy only** for `/admin` slash commands — **not** an inbound allowlist.
- **Never** tell owners a number is "not on the allowlist" for messaging unless `dm_policy` is actually `allowlist` (it is not).
- Legacy `bridge.log` `allowlist_mismatch` from old Node bridge is stale if the Go bridge on port 3000 is running — trust **gateway.log** + **agent.log** instead.

### Owner asks you to talk to someone else (e.g. "message +880… and tell me if human/bot")
1. **Send** via `send_message` to that JID (e.g. `whatsapp:+8801334962621` or `8801334962621@s.whatsapp.net`).
2. Their replies live in a **separate session** (`agent:main:whatsapp:dm:<their-jid>`) — **not** in the owner's DM transcript.
3. Before telling the owner "no reply yet", "still waiting", or your **guess** — **check facts** (pick one or more):
   - `grep '<digits>' /root/.hermes/logs/gateway.log | tail -40`
   - `sqlite3 /root/.hermes/state.db "SELECT role, substr(content,1,120) FROM messages WHERE session_id IN (SELECT session_id FROM sessions WHERE origin LIKE '%<jid>%') ORDER BY id DESC LIMIT 15"`
   - `sessions.json` / session list for that JID
4. Report to owner: what **they** said, timestamps, and your human/bot assessment — not a playbook of what you *would* watch for.
5. **Never** claim you only sent the first message if logs show a multi-turn exchange.

### Honesty
- If you have not checked logs/sessions yet, say "Let me check the other chat" and run the commands — do not improvise status.

"""


def main() -> None:
    text = SOUL.read_text(encoding="utf-8")
    if MARKER in text:
        print("ok: WhatsApp access/cross-chat section already present")
        return
    if ANCHOR in text:
        text = text.replace(ANCHOR, BLOCK.strip() + "\n\n" + ANCHOR, 1)
    else:
        anchor = "## WHATSAPP — production stack (system instruction)"
        if anchor not in text:
            raise SystemExit("SOUL anchor not found")
        text = text.replace(anchor, BLOCK.strip() + "\n\n" + anchor, 1)
    SOUL.write_text(text, encoding="utf-8")
    print("ok: SOUL WhatsApp access + cross-chat rules added")


if __name__ == "__main__":
    main()
