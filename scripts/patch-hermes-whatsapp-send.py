#!/usr/bin/env python3
"""Patch Hermes send_message_tool for WhatsApp JID targeting + home-channel guard."""
from pathlib import Path

TOOL = Path("/usr/local/lib/hermes-agent/tools/send_message_tool.py")


def main() -> None:
    text = TOOL.read_text()
    changed = False

    if 'if platform_name == "whatsapp":' not in text:
        needle = (
            "    if platform_name in _PHONE_PLATFORMS:\n"
            "        match = _E164_TARGET_RE.fullmatch(target_ref)"
        )
        insert = (
            '    if platform_name == "whatsapp":\n'
            "        trimmed = target_ref.strip()\n"
            '        if "@" in trimmed:\n'
            "            return trimmed, None, True\n"
            "    if platform_name in _PHONE_PLATFORMS:\n"
            "        match = _E164_TARGET_RE.fullmatch(target_ref)"
        )
        if needle not in text:
            raise SystemExit("needle1 missing")
        text = text.replace(needle, insert, 1)
        changed = True
        print("patched _parse_target_ref")

    if "Model mistake: bare WhatsApp JID used as platform" not in text:
        needle_jid = (
            '    target_ref = parts[1].strip() if len(parts) > 1 else None\n'
            "    chat_id = None"
        )
        insert_jid = (
            '    target_ref = parts[1].strip() if len(parts) > 1 else None\n'
            "    # Model mistake: bare WhatsApp JID used as platform (no whatsapp: prefix)\n"
            "    import re as _re_wa\n"
            '    _wa_jid_only = _re_wa.compile(r"^\\+?\\d+@s\\.whatsapp\\.net$", _re_wa.I)\n'
            "    if target_ref is None and _wa_jid_only.match(platform_name):\n"
            "        target_ref = platform_name.strip()\n"
            '        platform_name = "whatsapp"\n'
            "    chat_id = None"
        )
        if needle_jid not in text:
            raise SystemExit("needle_jid missing")
        text = text.replace(needle_jid, insert_jid, 1)
        changed = True
        print("patched JID-as-platform normalization")

    if "Refusing home-channel send" not in text:
        needle2 = (
            "        if home:\n"
            "            chat_id = home.chat_id\n"
            "            used_home_channel = True\n"
            "        else:"
        )
        insert2 = (
            "        if home:\n"
            "            chat_id = home.chat_id\n"
            "            used_home_channel = True\n"
            '            if platform_name == "whatsapp":\n'
            "                try:\n"
            "                    from gateway.session_context import get_session_env\n"
            '                    if not get_session_env("HERMES_CRON_AUTO_DELIVER_PLATFORM", "").strip():\n'
            '                        session_chat = get_session_env("HERMES_SESSION_CHAT_ID", "").strip()\n'
            "                        if session_chat and chat_id:\n"
            "                            def _norm_wa(j):\n"
            '                                j = (j or "").strip().lower()\n'
            '                                return j.split("@")[0].lstrip("+").replace(" ", "").replace("-", "")\n'
            "                            if _norm_wa(session_chat) != _norm_wa(chat_id):\n"
            "                                return json.dumps({\n"
            '                                    "error": "Refusing home-channel send: current chat is not the home channel. "\n'
            '                                    "For a specific recipient use whatsapp:+<E.164> (e.g. whatsapp:+6590016046) "\n'
            '                                    "or whatsapp:<digits>@s.whatsapp.net — never bare whatsapp for third parties."\n'
            "                                })\n"
            "                except Exception:\n"
            "                    pass\n"
            "        else:"
        )
        if needle2 not in text:
            raise SystemExit("needle2 missing")
        text = text.replace(needle2, insert2, 1)
        changed = True
        print("patched home channel guard")

    if changed:
        bak = TOOL.with_suffix(TOOL.suffix + ".pre-patch.bak")
        if not bak.exists():
            bak.write_text(TOOL.read_text())
        TOOL.write_text(text)
    else:
        print("no tool changes needed")


if __name__ == "__main__":
    main()
