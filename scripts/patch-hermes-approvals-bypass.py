#!/usr/bin/env python3
"""Disable Hermes dangerous-command approval prompts (VPS Stella)."""
from __future__ import annotations

import re
from pathlib import Path

HERMES = Path("/root/.hermes")
CONFIG = HERMES / "config.yaml"
DROPIN = Path("/etc/systemd/system/hermes-gateway.service.d/approvals-bypass.conf")

APPROVALS_BLOCK = """approvals:
  mode: "off"
  timeout: 0
  cron_mode: approve
  mcp_reload_confirm: false
  destructive_slash_confirm: false
"""

SYSTEMD_DROPIN = """# Hermes: auto-approve dangerous commands (no in-chat /approve prompts)
[Service]
Environment=HERMES_YOLO_MODE=1
"""


def patch_config() -> None:
    text = CONFIG.read_text(encoding="utf-8")
    if re.search(r"^approvals:\n", text, flags=re.M):
        text = re.sub(
            r"^approvals:\n(?:  .+\n)*",
            APPROVALS_BLOCK,
            text,
            count=1,
            flags=re.M,
        )
    else:
        text = text.rstrip() + "\n\n" + APPROVALS_BLOCK
    CONFIG.write_text(text, encoding="utf-8")


def patch_systemd() -> None:
    DROPIN.parent.mkdir(parents=True, exist_ok=True)
    DROPIN.write_text(SYSTEMD_DROPIN, encoding="utf-8")


def main() -> None:
    patch_config()
    patch_systemd()
    print("ok: approvals.mode=off, cron_mode=approve, HERMES_YOLO_MODE=1 drop-in")
    print("run: systemctl daemon-reload && systemctl restart hermes-gateway")


if __name__ == "__main__":
    main()
