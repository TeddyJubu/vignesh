#!/usr/bin/env python3
"""Block leaked tool/XML markup from WhatsApp (and all platforms) final replies."""
from __future__ import annotations

from pathlib import Path

RUN = Path("/usr/local/lib/hermes-agent/gateway/run.py")

MARKER = "def _looks_like_leaked_agent_tool_markup"

HELPER = '''
_LEAKED_TOOL_MARKUP_RE = re.compile(
    r"(<invoke\\s+name=|</minimax:tool_call>|<minimax:tool_call>|"
    r"<parameter\\s+name=|mcp_composio[A-Z_]+|"
    r"code_to_execute|COMPOSIO_REMOTE_WORKBENCH)",
    re.IGNORECASE,
)


def _looks_like_leaked_agent_tool_markup(text: str) -> bool:
    """True when model dumped internal tool-call XML into visible content."""
    if not text or len(text.strip()) < 24:
        return False
    if _LEAKED_TOOL_MARKUP_RE.search(text):
        return True
    lowered = text.lower()
    if "<invoke " in lowered and ("</invoke>" in lowered or "<parameter" in lowered):
        return True
    return False


'''

OLD_SANITIZE = '''def _sanitize_gateway_final_response(platform: Any, text: str) -> str:
    """Sanitize final gateway replies before sending them to high-noise chats.

    Telegram is Bob's mobile inbox, so it should receive concise, safe provider
    failure categories instead of raw HTTP bodies, request IDs, or policy text.
    Other platforms keep the existing behaviour for now.
    """
    if not text:
        return text
    if _gateway_platform_value(platform) != "telegram":
        return text

    redacted = _redact_gateway_user_facing_secrets(str(text))
    if _looks_like_gateway_provider_error(redacted):
        return _gateway_provider_error_reply(redacted)
    return redacted'''

NEW_SANITIZE = '''def _sanitize_gateway_final_response(platform: Any, text: str) -> str:
    """Sanitize final gateway replies before platform delivery."""
    if not text:
        return text
    platform_name = _gateway_platform_value(platform)
    redacted = _redact_gateway_user_facing_secrets(str(text))

    if _looks_like_leaked_agent_tool_markup(redacted):
        logger.warning(
            "Blocked leaked tool markup in gateway final response (platform=%s len=%d)",
            platform_name,
            len(redacted),
        )
        return (
            "I hit a glitch on that one — still working on it. "
            "If nothing lands in a minute, ping me again and I'll resend."
        )

    if platform_name != "telegram":
        return redacted

    if _looks_like_gateway_provider_error(redacted):
        return _gateway_provider_error_reply(redacted)
    return redacted'''


def main() -> None:
    text = RUN.read_text(encoding="utf-8")

    if MARKER in text and NEW_SANITIZE.split('"""')[1][:30] in text:
        print("ok: gateway sanitize already patched")
        return

    if MARKER not in text:
        if OLD_SANITIZE not in text:
            raise SystemExit("expected _sanitize_gateway_final_response block missing")
        text = text.replace(OLD_SANITIZE, HELPER + NEW_SANITIZE, 1)
        print("ok: added leak detector + platform-wide sanitize")
    elif OLD_SANITIZE in text:
        text = text.replace(OLD_SANITIZE, NEW_SANITIZE, 1)
        print("ok: updated sanitize body")
    else:
        raise SystemExit("unexpected gateway/run.py state")

    bak = RUN.with_suffix(".py.pre-wa-sanitize.bak")
    if not bak.exists():
        bak.write_text(RUN.read_text(encoding="utf-8"), encoding="utf-8")
    RUN.write_text(text, encoding="utf-8")


if __name__ == "__main__":
    main()
