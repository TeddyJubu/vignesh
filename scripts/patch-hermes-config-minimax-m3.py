#!/usr/bin/env python3
"""Set Hermes default model to MiniMax M3 on OpenRouter."""
from __future__ import annotations

import re
from pathlib import Path

CONFIG = Path("/root/.hermes/config.yaml")

OLD_DEFAULT = "minimax/minimax-m2.7"
NEW_DEFAULT = "minimax/minimax-m3"
NEW_CONTEXT = 1048576


def main() -> None:
    text = CONFIG.read_text(encoding="utf-8")
    if NEW_DEFAULT in text.splitlines()[1:3] if "default:" in text[:200] else NEW_DEFAULT in text:
        if f"context_length: {NEW_CONTEXT}" in text:
            print("ok: config already on minimax-m3")
            return

    bak = CONFIG.with_suffix(".yaml.pre-m3.bak")
    if not bak.exists():
        bak.write_text(text, encoding="utf-8")

    if OLD_DEFAULT in text:
        text = text.replace(OLD_DEFAULT, NEW_DEFAULT, 1)
        print(f"ok: default model {OLD_DEFAULT} -> {NEW_DEFAULT}")
    elif NEW_DEFAULT not in text:
        raise SystemExit("model.default not found in config.yaml")

    text, n = re.subn(
        r"(^  context_length: )\d+",
        rf"\g<1>{NEW_CONTEXT}",
        text,
        count=1,
        flags=re.MULTILINE,
    )
    if n:
        print(f"ok: context_length -> {NEW_CONTEXT}")

    if "minimax/minimax-m2.7" not in text:
        needle = "fallback_providers:\n"
        insert = (
            "fallback_providers:\n"
            "- provider: openrouter\n"
            "  model: minimax/minimax-m2.7\n"
            "  base_url: https://openrouter.ai/api/v1\n"
        )
        if needle in text:
            text = text.replace(needle, insert, 1)
            print("ok: added m2.7 fallback")

    CONFIG.write_text(text, encoding="utf-8")
    print("ok: config.yaml written — restart hermes-gateway")


if __name__ == "__main__":
    main()
