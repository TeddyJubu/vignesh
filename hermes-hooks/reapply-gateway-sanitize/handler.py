"""Re-run gateway final-response sanitizer patch on gateway startup."""

from __future__ import annotations

import logging
import subprocess
import sys
from pathlib import Path

logger = logging.getLogger("hooks.reapply-gateway-sanitize")

PATCH_SCRIPTS = (
    Path.home() / ".hermes" / "scripts" / "patch-hermes-gateway-whatsapp-sanitize.py",
    Path.home() / ".hermes" / "scripts" / "patch-hermes-agent-minimax-tool-leak.py",
    Path.home() / ".hermes" / "scripts" / "patch-hermes-gateway-media-reports-fallback.py",
    Path.home() / ".hermes" / "scripts" / "patch-hermes-gateway-agent-end-verifier.py",
)


async def handle(event_type: str, context: dict) -> None:
    for patch_script in PATCH_SCRIPTS:
        if not patch_script.is_file():
            logger.warning("patch script not found: %s", patch_script)
            continue
        try:
            result = subprocess.run(
                [sys.executable, str(patch_script)],
                capture_output=True,
                text=True,
                timeout=30,
                check=False,
            )
        except Exception as exc:
            logger.error("%s failed: %s", patch_script.name, exc)
            continue
        if result.returncode != 0:
            logger.error(
                "%s exit %s: %s",
                patch_script.name,
                result.returncode,
                (result.stderr or result.stdout or "").strip(),
            )
        else:
            logger.info("%s: %s", patch_script.name, (result.stdout or "ok").strip())
