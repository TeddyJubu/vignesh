#!/usr/bin/env python3
"""Enable generous Honcho auto-injection (context + dialectic) for Stella."""
from __future__ import annotations

import json
from pathlib import Path

HONCHO = Path("/root/.hermes/honcho.json")


def main() -> None:
    if not HONCHO.is_file():
        raise SystemExit(f"missing {HONCHO}")
    data = json.loads(HONCHO.read_text(encoding="utf-8"))
    data["contextTokens"] = 8000
    data["injectionFrequency"] = "every-turn"
    data["dialecticMaxChars"] = 2000
    data["dialecticReasoningLevel"] = "medium"
    data["dialecticCadence"] = 1

    for host in (data.get("hosts") or {}).values():
        if not isinstance(host, dict):
            continue
        host["recallMode"] = "hybrid"
        host["contextTokens"] = 8000
        host["dialecticReasoningLevel"] = "medium"
        host["dialecticCadence"] = 1
        host["dialecticMaxChars"] = 2000
        host.setdefault("enabled", True)

    HONCHO.write_text(json.dumps(data, indent=2) + "\n", encoding="utf-8")
    print("ok: honcho.json tuned for full memory injection (8k context, medium dialectic, every-turn)")


if __name__ == "__main__":
    main()
