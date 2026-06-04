#!/usr/bin/env python3
"""Remove duplicate outreach record blocks from send_message_tool (post re-patch)."""
from __future__ import annotations

from pathlib import Path

TOOL = Path("/usr/local/lib/hermes-agent/tools/send_message_tool.py")

BLOCK_START = (
    '        if platform_name == "whatsapp" and isinstance(result, dict) and result.get("success") and mirror_text:\n'
    "            try:\n"
    "                import sys as _sys_outreach\n"
)
MIRROR_START = '\n\n        if isinstance(result, dict) and result.get("success") and mirror_text:'


def main() -> None:
    text = TOOL.read_text(encoding="utf-8")
    first = text.find(BLOCK_START)
    if first == -1:
        print("ok: no whatsapp outreach block")
        return
    second = text.find(BLOCK_START, first + len(BLOCK_START))
    if second == -1:
        print("ok: single outreach block")
        return
    end = text.find(MIRROR_START, second)
    if end == -1:
        raise SystemExit("mirror block not found after duplicate outreach")
    text = text[:second] + text[end:]
    TOOL.write_text(text, encoding="utf-8")
    print("ok: removed duplicate outreach record block")


if __name__ == "__main__":
    main()
