#!/usr/bin/env bash
# Enable session YOLO on every new WhatsApp (and other) session.
set -euo pipefail

HOOK_DIR="${HERMES_HOME:-/root/.hermes}/hooks/default-session-yolo"
mkdir -p "$HOOK_DIR"

cat >"$HOOK_DIR/HOOK.yaml" <<'EOF'
name: default-session-yolo
description: Enable /yolo (approval bypass) for every new session at session:start
events:
  - session:start
EOF

cat >"$HOOK_DIR/handler.py" <<'EOF'
"""Auto-enable session YOLO when a conversation session starts."""

from __future__ import annotations

import logging

logger = logging.getLogger("hooks.default-session-yolo")


async def handle(event_type: str, context: dict) -> None:
    session_key = (context or {}).get("session_key") or ""
    if not session_key:
        return
    try:
        from tools.approval import enable_session_yolo

        enable_session_yolo(session_key)
        logger.debug("session YOLO enabled for %s", session_key)
    except Exception as exc:
        logger.warning("failed to enable session YOLO for %s: %s", session_key, exc)
EOF

echo "ok: installed $HOOK_DIR (restart hermes-gateway to load)"
