#!/usr/bin/env python3
"""Retry when MiniMax dumps tool-call XML into assistant text instead of using tools API."""
from __future__ import annotations

from pathlib import Path

LOOP = Path("/usr/local/lib/hermes-agent/agent/conversation_loop.py")

MARKER = "minimax_tool_leak_retries"
INIT_NEEDLE = "    codex_ack_continuations = 0\n"
INIT_INSERT = "    codex_ack_continuations = 0\n    minimax_tool_leak_retries = 0\n"

RETRY_BLOCK = '''
                if (
                    minimax_tool_leak_retries < 2
                    and (
                        "<invoke " in final_response
                        or "</minimax:tool_call>" in final_response
                        or (
                            "<parameter name=" in final_response
                            and "mcp_" in final_response.lower()
                        )
                    )
                ):
                    minimax_tool_leak_retries += 1
                    logger.warning(
                        "Leaked tool markup in assistant text; retrying (%s/2) session=%s",
                        minimax_tool_leak_retries,
                        getattr(agent, "session_id", "?"),
                    )
                    continue_msg = {
                        "role": "user",
                        "content": (
                            "[System: Your last output was internal tool-call markup, not a reply "
                            "to the user. Do NOT send XML, <invoke>, Python snippets, or Composio blocks. "
                            "Use the tool-calling API silently. Then send a short human summary. "
                            "If they asked for a file/CSV, write it under ~/.hermes/reports/ and "
                            "include MEDIA: attachment lines.]"
                        ),
                    }
                    messages.append(continue_msg)
                    agent._session_messages = messages
                    continue

'''

ANCHOR = "                final_response = agent._strip_think_blocks(final_response).strip()\n                \n                final_msg = agent._build_assistant_message(assistant_message, finish_reason)"


def main() -> None:
    text = LOOP.read_text(encoding="utf-8")

    if MARKER in text:
        print("ok: conversation_loop already has minimax leak retry")
        return

    if INIT_NEEDLE not in text:
        raise SystemExit("codex_ack init needle missing")
    text = text.replace(INIT_NEEDLE, INIT_INSERT, 1)

    if ANCHOR not in text:
        raise SystemExit("strip/final_msg anchor missing")
    text = text.replace(
        ANCHOR,
        "                final_response = agent._strip_think_blocks(final_response).strip()\n"
        + RETRY_BLOCK
        + "\n                final_msg = agent._build_assistant_message(assistant_message, finish_reason)",
        1,
    )

    bak = LOOP.with_suffix(".py.pre-minimax-leak.bak")
    if not bak.exists():
        bak.write_text(LOOP.read_text(encoding="utf-8"), encoding="utf-8")
    LOOP.write_text(text, encoding="utf-8")
    print("ok: minimax tool-leak retry patched")


if __name__ == "__main__":
    main()
