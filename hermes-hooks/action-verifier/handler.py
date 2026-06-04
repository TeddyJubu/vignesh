"""Post-turn verifier: claims vs logs; inject correction for Stella on failure."""

from __future__ import annotations

import asyncio
import json
import logging
import sys
import urllib.request
from datetime import datetime, timezone
from pathlib import Path

logger = logging.getLogger("hooks.action-verifier")

_HERMES_HOME = Path.home() / ".hermes"
if str(_HERMES_HOME) not in sys.path:
    sys.path.insert(0, str(_HERMES_HOME))

LOG_PATH = Path.home() / ".hermes" / "logs" / "action-verifier.log"
INJECT_URL = "http://127.0.0.1:3000/inject"


def _log(entry: dict) -> None:
    LOG_PATH.parent.mkdir(parents=True, exist_ok=True)
    entry["ts"] = datetime.now(timezone.utc).isoformat()
    with LOG_PATH.open("a", encoding="utf-8") as f:
        f.write(json.dumps(entry, ensure_ascii=False) + "\n")


def _inject_followup(chat_id: str, sender_id: str, body: str) -> bool:
    payload = {
        "body": body,
        "chatId": chat_id,
        "senderId": sender_id,
        "senderName": "ActionVerifier",
        "chatName": "ActionVerifier",
        "isGroup": False,
        "hasMedia": False,
        "mediaType": "",
        "mediaUrls": [],
        "mentionedIds": [],
        "quotedMessageId": "",
        "quotedParticipant": "",
        "quotedRemoteJid": "",
        "hasQuotedMessage": False,
        "botIds": [],
        "timestamp": int(datetime.now(timezone.utc).timestamp()),
    }
    req = urllib.request.Request(
        INJECT_URL,
        data=json.dumps(payload).encode(),
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            return 200 <= resp.status < 300
    except Exception as exc:
        logger.warning("inject failed: %s", exc)
        return False


async def _verify_turn(context: dict) -> None:
    try:
        from stella_action_verifier.config import load_config
        from stella_action_verifier.verify import format_correction, verify_turn
    except ImportError as exc:
        logger.error("verifier module missing: %s", exc)
        return

    cfg = load_config()
    platform = (context.get("platform") or "").lower()
    chat_id = (context.get("chat_id") or "").strip()
    session_id = (context.get("session_id") or "").strip()
    user_message = context.get("message") or ""
    response = context.get("response_full") or context.get("response") or ""

    result = verify_turn(
        response=response,
        user_message=user_message,
        chat_id=chat_id,
        session_id=session_id,
        platform=platform,
        cfg=cfg,
    )

    entry = {
        "platform": platform,
        "chat_id": chat_id,
        "session_id": session_id,
        "ok": result.ok,
        "used_llm": result.used_llm,
        "issues": [{"claim": i.claim, "proof": i.proof} for i in result.issues],
        "evidence": result.evidence_summary,
    }
    _log(entry)

    if result.ok or not result.issues:
        return
    if not cfg.get("inject_correction", True):
        logger.info("verifier failed but inject_correction=false")
        return

    correction = format_correction(result.issues, result.evidence_summary)
    sender = chat_id or "6590013157@s.whatsapp.net"
    ok = await asyncio.to_thread(_inject_followup, chat_id or sender, sender, correction)
    _log({"inject_correction": ok, "chat_id": chat_id})
    if ok:
        logger.info("verifier injected correction (%d issues)", len(result.issues))
    else:
        logger.warning("verifier could not inject correction")


async def handle(event_type: str, context: dict) -> None:
    if event_type != "agent:end":
        return
    asyncio.create_task(_verify_turn(context or {}))
