#!/usr/bin/env python3
"""Deprecated: use purge-hermes-wabot-references.py instead."""
from pathlib import Path

HERMES = Path("/root/.hermes")
WHATSMEOW_SKILL = HERMES / "skills/social-media/whatsmeow/SKILL.md"
OPERATOR_SKILL = HERMES / "skills/social-media/whatsapp-operator/SKILL.md"
SOUL = HERMES / "SOUL.md"
MEMORY = HERMES / "memories/MEMORY.md"

ARCH_BLOCK = """## Production stack (srv943071 — Jun 2026)

**Do not use or mention wabot (port 7777).** It was disabled 2026-06-03 with wabot-agent and ai-receptionist (duplicate replies on the same WhatsApp number).

| Service | Port | Purpose |
|---------|------|---------|
| whatsmeow-bridge | 3000 | Send/receive DMs (`POST /send`, long-poll `/messages`) |
| hermes-gateway | — | Stella agent (Hermes) |

**Group creation via HTTP API is not available** on this stack. Options for users:
1. Create the group manually in WhatsApp.
2. Ask for a future bridge feature — do not suggest re-enabling wabot unless the owner explicitly requests restoring the old stack.

For Go/library group APIs, see whatsmeow docs below — that is not exposed as HTTP on this VPS.

"""

OPERATOR_NOTE = """## Groups on this VPS (important)

**wabot (:7777) is disabled.** Do not tell users to re-enable wabot unless they explicitly ask to restore the legacy stack.

- **Create/add/remove groups via API:** not available (bridge has no group endpoints).
- **Manual:** owner creates the group in WhatsApp, then Stella can message the group JID via `send_message` once it exists.
- **Messaging:** use Hermes `send_message` + bridge `:3000` only.

"""

SOUL_NOTE = """
### WhatsApp infra (current — not wabot)

- **Active:** whatsmeow-bridge `:3000` + hermes-gateway (Stella).
- **Disabled (Jun 2026):** wabot `:7777`, wabot-agent, ai-receptionist — do not recommend re-enabling for routine tasks; caused duplicate replies.
- **Groups:** no HTTP group-creation API; manual WhatsApp or message an existing group JID.

"""


def patch_whatsmeow_skill(text: str) -> str:
    if "Do not use or mention wabot" in text:
        return text
    # Drop duplicate References + old wabot table
    start = text.find("## Wabot & Bridge Architecture")
    if start >= 0:
        end = text.find("\n## Installation", start)
        if end < 0:
            end = text.find("\n## ", start + 5)
        if end > start:
            text = text[:start] + ARCH_BLOCK + text[end:]
    # Dedupe wabot reference lines in References
    lines = []
    seen_ref = False
    for line in text.splitlines():
        if "wabot-http-api" in line:
            if seen_ref:
                continue
            seen_ref = True
            lines.append(
                "- `references/bridge-http-api.md` — **Production** bridge HTTP API (port 3000, messaging only)"
            )
            continue
        if "Wabot HTTP API on port 7777" in line:
            continue
        lines.append(line)
    return "\n".join(lines) + ("\n" if not text.endswith("\n") else "")


def patch_operator_skill(text: str) -> str:
    if "wabot (:7777) is disabled" in text:
        return text
    lines = []
    for line in text.splitlines():
        if "wabot-api.md" in line:
            lines.append(
                "- `references/bridge-http-api.md` — Bridge :3000 (production messaging; groups not via API)"
            )
            continue
        lines.append(line)
    text = "\n".join(lines)
    marker = "**Don't use for:** WhatsApp Business API"
    if marker in text and OPERATOR_NOTE not in text:
        text = text.replace(marker, OPERATOR_NOTE + marker, 1)
    return text


def patch_soul(text: str) -> str:
    if "WhatsApp infra (current" in text:
        return text
    anchor = "## SINGLE WHATSAPP REPLY"
    if anchor in text:
        return text.replace(anchor, SOUL_NOTE + anchor, 1)
    return SOUL_NOTE + text


def main() -> None:
    for path, fn in (
        (WHATSMEOW_SKILL, patch_whatsmeow_skill),
        (OPERATOR_SKILL, patch_operator_skill),
        (SOUL, patch_soul),
    ):
        if not path.exists():
            print("skip missing", path)
            continue
        raw = path.read_text()
        new = fn(raw)
        if new != raw:
            bak = path.with_suffix(path.suffix + ".pre-no-wabot.bak")
            if not bak.exists():
                bak.write_text(raw)
            path.write_text(new)
            print("patched", path)
        else:
            print("unchanged", path)

    if MEMORY.exists():
        mem = MEMORY.read_text()
        if "wabot:7777" in mem and "disabled 2026-06-03" not in mem.split("wabot")[0][-80:]:
            mem = mem.replace("wabot:7777", "wabot:7777 OFFLINE")
            MEMORY.write_text(mem)
            print("touched MEMORY.md")


if __name__ == "__main__":
    main()
