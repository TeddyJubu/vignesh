#!/usr/bin/env python3
"""Pass full response + session_key to agent:end hook for action verifier."""
from __future__ import annotations

from pathlib import Path

RUN = Path("/usr/local/lib/hermes-agent/gateway/run.py")

OLD = '''            await self.hooks.emit("agent:end", {
                **hook_ctx,
                "response": (response or "")[:500],
            })'''

NEW = '''            await self.hooks.emit("agent:end", {
                **hook_ctx,
                "session_key": session_key,
                "response": (response or "")[:500],
                "response_full": (response or "")[:4000],
            })'''


def main() -> None:
    text = RUN.read_text(encoding="utf-8")
    if "response_full" in text and OLD not in text:
        print("ok: agent:end verifier context already patched")
        return
    if OLD not in text:
        raise SystemExit("agent:end emit block not found")
    bak = RUN.with_suffix(".py.pre-agent-end-verifier.bak")
    if not bak.exists():
        bak.write_text(text, encoding="utf-8")
    RUN.write_text(text.replace(OLD, NEW, 1), encoding="utf-8")
    print("ok: agent:end now includes response_full + session_key")


if __name__ == "__main__":
    main()
