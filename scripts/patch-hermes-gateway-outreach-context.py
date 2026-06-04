#!/usr/bin/env python3
"""Inject outreach task context + seed transcript on third-party WhatsApp inbound."""
from __future__ import annotations

from pathlib import Path

RUN = Path("/usr/local/lib/hermes-agent/gateway/run.py")
MARKER = "outreach_tasks.maybe_inject_whatsapp_outreach"

NEEDLE = (
    "        # Load conversation history from transcript\n"
    "        history = self.session_store.load_transcript(session_entry.session_id)\n"
)

INSERT = """        # Load conversation history from transcript
        history = self.session_store.load_transcript(session_entry.session_id)

        try:
            import sys as _sys_or
            _orp = "/root/.hermes/scripts"
            if _orp not in _sys_or.path:
                _sys_or.path.insert(0, _orp)
            from outreach_tasks import maybe_inject_whatsapp_outreach
            _plat = source.platform.value if source.platform else ""
            history, context_prompt = maybe_inject_whatsapp_outreach(
                platform=_plat,
                chat_id=source.chat_id or "",
                user_display_name=getattr(source, "user_name", "") or "",
                session_id=session_entry.session_id,
                history=history,
                context_prompt=context_prompt,
                session_store=self.session_store,
            )
        except Exception as _or_err:
            logger.debug("outreach context inject failed: %s", _or_err)

"""


def main() -> None:
    text = RUN.read_text(encoding="utf-8")
    if MARKER in text:
        print("ok: gateway outreach inject already patched")
        return
    if NEEDLE not in text:
        raise SystemExit("load_transcript needle missing")
    if text.count(NEEDLE) != 1:
        raise SystemExit("ambiguous load_transcript needle (already patched?)")
    bak = RUN.with_suffix(".py.pre-outreach-context.bak")
    if not bak.exists():
        bak.write_text(text, encoding="utf-8")
    RUN.write_text(text.replace(NEEDLE, INSERT, 1), encoding="utf-8")
    print("ok: gateway injects outreach context on third-party WhatsApp inbound")


if __name__ == "__main__":
    main()
