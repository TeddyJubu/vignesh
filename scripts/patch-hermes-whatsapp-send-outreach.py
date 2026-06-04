#!/usr/bin/env python3
"""Record outreach task file after successful WhatsApp send_message."""
from __future__ import annotations

import sys
from pathlib import Path

TOOL = Path("/usr/local/lib/hermes-agent/tools/send_message_tool.py")
SCRIPTS = Path("/root/.hermes/scripts")
MARKER = "outreach_tasks.record_whatsapp_outreach"


def main() -> None:
    text = TOOL.read_text(encoding="utf-8")
    if text.count(MARKER) >= 1:
        print("ok: send outreach record already patched")
        return

    needle = (
        "        if isinstance(result, dict) and result.get(\"success\") and mirror_text:\n"
        "            try:\n"
        "                from gateway.mirror import mirror_to_session\n"
    )
    insert = (
        "        if platform_name == \"whatsapp\" and isinstance(result, dict) and result.get(\"success\") and mirror_text:\n"
        "            try:\n"
        "                import sys as _sys_outreach\n"
        f"                _op = {repr(str(SCRIPTS))}\n"
        "                if _op not in _sys_outreach.path:\n"
        "                    _sys_outreach.path.insert(0, _op)\n"
        "                from outreach_tasks import record_whatsapp_outreach\n"
        "                from gateway.session_context import get_session_env as _gse_outreach\n"
        "                record_whatsapp_outreach(\n"
        "                    chat_id=chat_id,\n"
        "                    message_text=mirror_text,\n"
        "                    owner_chat_id=_gse_outreach(\"HERMES_SESSION_CHAT_ID\", \"\").strip(),\n"
        "                    owner_request=_gse_outreach(\"HERMES_LAST_USER_MESSAGE\", \"\").strip(),\n"
        "                )\n"
        "            except Exception:\n"
        "                pass\n"
        "\n"
        "        if isinstance(result, dict) and result.get(\"success\") and mirror_text:\n"
        "            try:\n"
        "                from gateway.mirror import mirror_to_session\n"
    )
    if needle not in text:
        raise SystemExit("mirror block needle missing")
    text = text.replace(needle, insert, 1)
    TOOL.write_text(text, encoding="utf-8")
    print("ok: send_message records outreach tasks on WhatsApp success")


if __name__ == "__main__":
    main()
