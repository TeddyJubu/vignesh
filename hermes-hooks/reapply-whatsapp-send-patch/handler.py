"""Re-run ~/.hermes/scripts/patch-hermes-whatsapp-send.py on every gateway startup."""

from __future__ import annotations

import logging
import subprocess
import sys
from datetime import datetime, timezone
from pathlib import Path

logger = logging.getLogger("hooks.reapply-whatsapp-send-patch")

PATCH_SCRIPT = Path.home() / ".hermes" / "scripts" / "patch-hermes-whatsapp-send.py"
LOG_FILE = Path.home() / ".hermes" / "logs" / "patch-whatsapp-send.log"


def _append_log(event_type: str, result: subprocess.CompletedProcess[str] | None, error: str | None = None) -> None:
    LOG_FILE.parent.mkdir(parents=True, exist_ok=True)
    ts = datetime.now(timezone.utc).isoformat()
    with LOG_FILE.open("a", encoding="utf-8") as f:
        f.write(f"\n--- {ts} event={event_type} ---\n")
        if error:
            f.write(f"error: {error}\n")
            return
        if result is None:
            f.write("skipped: patch script missing\n")
            return
        f.write(f"exit={result.returncode}\n")
        if result.stdout:
            f.write(result.stdout)
            if not result.stdout.endswith("\n"):
                f.write("\n")
        if result.stderr:
            f.write(result.stderr)
            if not result.stderr.endswith("\n"):
                f.write("\n")


async def handle(event_type: str, context: dict) -> None:
    if not PATCH_SCRIPT.is_file():
        logger.warning("patch script not found: %s", PATCH_SCRIPT)
        _append_log(event_type, None)
        return

    try:
        result = subprocess.run(
            [sys.executable, str(PATCH_SCRIPT)],
            capture_output=True,
            text=True,
            timeout=30,
            check=False,
        )
    except Exception as exc:
        logger.error("failed to run whatsapp send patch: %s", exc)
        _append_log(event_type, None, error=str(exc))
        return

    _append_log(event_type, result)
    if result.returncode != 0:
        logger.error(
            "whatsapp send patch failed (exit %s): %s",
            result.returncode,
            (result.stderr or result.stdout or "").strip(),
        )
    else:
        logger.info(
            "whatsapp send patch: %s",
            (result.stdout or "ok").strip(),
        )
