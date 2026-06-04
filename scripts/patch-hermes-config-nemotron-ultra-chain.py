#!/usr/bin/env python3
"""Set Stella model chain: OpenRouter Nemotron 3 Ultra → Kilocode Ultra → kilo-auto/free."""
from __future__ import annotations

import re
from pathlib import Path

CONFIG = Path("/root/.hermes/config.yaml")

OR_MODEL = "nvidia/nemotron-3-ultra-550b-a55b"
KILO_ULTRA = "nvidia/nemotron-3-ultra-550b-a55b"
KILO_AUTO = "kilo-auto/free"
OR_BASE = "https://openrouter.ai/api/v1"
KILO_BASE = "https://api.kilo.ai/api/gateway"
CONTEXT = 1048576

FALLBACK_BLOCK = f"""fallback_providers:
- provider: kilocode
  model: {KILO_ULTRA}
  base_url: {KILO_BASE}
- provider: kilocode
  model: {KILO_AUTO}
  base_url: {KILO_BASE}
"""

MARKER = f"  default: {OR_MODEL}"


def main() -> None:
    text = CONFIG.read_text(encoding="utf-8")
    if MARKER in text and f"model: {KILO_ULTRA}" in text.split("fallback_providers", 1)[-1][:400]:
        print("ok: nemotron ultra chain already configured")
        return

    bak = CONFIG.with_suffix(".yaml.pre-nemotron-ultra.bak")
    if not bak.exists():
        bak.write_text(text, encoding="utf-8")

    # Primary: OpenRouter Nemotron 3 Ultra
    text = re.sub(
        r"(?m)^  default: .+$",
        f"  default: {OR_MODEL}",
        text,
        count=1,
    )
    text = re.sub(
        r"(?m)^  provider: .+$",
        "  provider: openrouter",
        text,
        count=1,
    )
    if "  base_url: https://openrouter.ai/api/v1" not in text[:800]:
        # Replace kilocode base_url under model: with openrouter
        text = re.sub(
            r"(?m)^  base_url: https://api\.kilo\.ai/api/gateway\s*$",
            f"  base_url: {OR_BASE}",
            text,
            count=1,
        )
    else:
        text = re.sub(
            r"(?m)^  base_url: .+$",
            f"  base_url: {OR_BASE}",
            text,
            count=1,
        )

    text = re.sub(
        r"(?m)^  context_length: \d+",
        f"  context_length: {CONTEXT}",
        text,
        count=1,
    )

    # Replace entire fallback_providers block (until credential_pool or toolsets)
    fb_match = re.search(
        r"fallback_providers:\n(?:- .+\n(?:  .+\n)*)+",
        text,
    )
    if fb_match:
        text = text[: fb_match.start()] + FALLBACK_BLOCK + text[fb_match.end() :]
        print("ok: fallback_providers -> kilocode ultra, kilo-auto/free")
    else:
        insert_after = "providers: {}\n"
        if insert_after not in text:
            raise SystemExit("providers: {} anchor missing")
        text = text.replace(insert_after, insert_after + FALLBACK_BLOCK, 1)
        print("ok: inserted fallback_providers")

    CONFIG.write_text(text, encoding="utf-8")
    print(
        f"ok: primary openrouter/{OR_MODEL}; "
        f"fallback kilocode/{KILO_ULTRA} then {KILO_AUTO} — restart hermes-gateway"
    )


if __name__ == "__main__":
    main()
