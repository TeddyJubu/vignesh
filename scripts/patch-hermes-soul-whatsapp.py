#!/usr/bin/env python3
"""Add WhatsApp third-party targeting rules to SOUL.md."""
from pathlib import Path

SOUL = Path("/root/.hermes/SOUL.md")

BLOCK = """- **Targeting new numbers:** Use `send_message` with `whatsapp:+<country><number>` (E.164, e.g. `whatsapp:+6590016046`) or `whatsapp:<digits>@s.whatsapp.net`. Works even if they are not in `send_message(action='list')`.
- **Never** use bare `whatsapp` (home channel) when asked to message someone else — that misroutes to Teddy.
- Do not say "not saved in contacts"; use the `whatsapp:+…` format if needed.

"""


def main() -> None:
    t = SOUL.read_text()
    if "Targeting new numbers" in t:
        print("SOUL already updated")
        return
    marker = '- NEVER say "I can’t message third parties due to WhatsApp policy"'
    if marker not in t:
        marker = "- NEVER say \"I can't message third parties due to WhatsApp policy\""
    if marker not in t:
        raise SystemExit("SOUL marker not found")
    SOUL.write_text(t.replace(marker, BLOCK + marker, 1))
    print("SOUL.md updated")


if __name__ == "__main__":
    main()
