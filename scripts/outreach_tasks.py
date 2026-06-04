#!/usr/bin/env python3
"""Persist WhatsApp outreach context so third-party DM sessions are not amnesiac."""
from __future__ import annotations

import json
import re
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Optional


def _home() -> Path:
    import os

    return Path(os.environ.get("HERMES_HOME", Path.home() / ".hermes"))


def _tasks_dir() -> Path:
    d = _home() / "tasks" / "outreach"
    d.mkdir(parents=True, exist_ok=True)
    return d


def jid_digits(jid: str) -> str:
    return re.sub(r"\D", "", (jid or "").split("@")[0])


def normalize_wa_jid(chat_id: str) -> str:
    c = (chat_id or "").strip()
    if "@" in c:
        return c.lower()
    d = jid_digits(c)
    return f"{d}@s.whatsapp.net" if d else c


def is_owner_jid(chat_id: str) -> bool:
    return jid_digits(chat_id) in {"6590013157", "8801521207499"}


def task_path(chat_id: str) -> Path:
    return _tasks_dir() / f"{jid_digits(chat_id)}.json"


def load_task(chat_id: str) -> Optional[dict[str, Any]]:
    path = task_path(chat_id)
    if not path.is_file():
        return None
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except (json.JSONDecodeError, OSError):
        return None


def _extract_contact_name(message: str) -> str:
    m = re.search(r"\bHi\s+([A-Za-z][A-Za-z\s.'-]{0,40}?)[,!.]", message or "", re.I)
    if m:
        return m.group(1).strip()
    return ""


def record_whatsapp_outreach(
    *,
    chat_id: str,
    message_text: str,
    owner_chat_id: str = "",
    owner_request: str = "",
) -> dict[str, Any]:
    """After a successful send_message to a third party, persist task + try mirror."""
    jid = normalize_wa_jid(chat_id)
    if is_owner_jid(jid):
        return {}

    now = datetime.now(timezone.utc).isoformat()
    existing = load_task(jid) or {}
    contact_name = _extract_contact_name(message_text) or existing.get("contact_name") or ""

    task: dict[str, Any] = {
        "jid": jid,
        "contact_name": contact_name,
        "whatsapp_display_name": existing.get("whatsapp_display_name") or "",
        "last_outbound": (message_text or "")[:4000],
        "last_outbound_at": now,
        "owner_chat_id": owner_chat_id or existing.get("owner_chat_id") or "",
        "owner_request": (owner_request or existing.get("owner_request") or "")[:2000],
        "mirrored": False,
    }

    try:
        from gateway.mirror import mirror_to_session
        from gateway.session_context import get_session_env

        owner_chat_id = owner_chat_id or get_session_env("HERMES_SESSION_CHAT_ID", "").strip()
        user_id = jid_digits(jid)
        if mirror_to_session(
            "whatsapp",
            jid,
            message_text,
            source_label="outreach",
            user_id=user_id,
        ):
            task["mirrored"] = True
    except Exception:
        pass

    path = task_path(jid)
    path.write_text(json.dumps(task, indent=2), encoding="utf-8")
    write_outreach_memory_md(task)
    sync_outreach_to_honcho(task)
    return task


def honcho_session_key(jid: str) -> str:
    return f"agent-main-whatsapp-dm-{jid_digits(jid)}"


def write_outreach_memory_md(task: dict[str, Any]) -> None:
    """File-based memory Hermes can load on session start."""
    mem_dir = _home() / "memories"
    mem_dir.mkdir(parents=True, exist_ok=True)
    digits = jid_digits(task.get("jid", ""))
    path = mem_dir / f"outreach-{digits}.md"
    name = task.get("contact_name") or "contact"
    display = task.get("whatsapp_display_name") or ""
    lines = [
        f"# Outreach: {name} ({task.get('jid', '')})",
        "",
        f"- **Use name:** {name}",
    ]
    if display and display.lower() != name.lower():
        lines.append(f"- **WhatsApp profile name:** {display} (still call them {name})")
    if task.get("owner_request"):
        lines.append(f"- **Owner request:** {task['owner_request']}")
    if task.get("last_outbound"):
        lines.append(f"- **Last outbound message:** {task['last_outbound']}")
    lines.append("")
    path.write_text("\n".join(lines), encoding="utf-8")


def sync_outreach_to_honcho(task: dict[str, Any]) -> bool:
    """Push outreach facts + assistant outbound into Honcho for this DM session."""
    jid = task.get("jid") or ""
    if not jid or is_owner_jid(jid):
        return False
    key = honcho_session_key(jid)
    outbound = (task.get("last_outbound") or "").strip()
    if not outbound:
        return False

    try:
        from plugins.memory.honcho.client import HonchoClientConfig, get_honcho_client
        from plugins.memory.honcho.session import HonchoSessionManager
    except ImportError:
        return False

    try:
        cfg = HonchoClientConfig.from_global_config()
        if not cfg.enabled:
            return False
        client = get_honcho_client(cfg)
        manager = HonchoSessionManager(
            honcho=client,
            config=cfg,
            context_tokens=cfg.context_tokens,
            runtime_user_peer_name=jid_digits(jid),
        )
        session = manager.get_or_create(key)

        snippet = outbound[:100]
        if not any(snippet in str(m.get("content") or "") for m in session.messages):
            session.add_message("assistant", outbound, mirror=True, mirror_source="outreach_honcho")
            manager._flush_session(session)

        name = task.get("contact_name") or "contact"
        display = task.get("whatsapp_display_name") or ""
        facts = [
            f"Preferred name: {name}",
            f"WhatsApp JID: {jid}",
        ]
        if display and display.lower() != name.lower():
            facts.append(f"WhatsApp display name: {display} (call them {name})")
        if task.get("owner_request"):
            facts.append(f"Scheduling context: {task['owner_request'][:500]}")
        facts.append(f"Last outbound: {outbound[:400]}")

        existing = manager.get_peer_card(key) or []
        merged = list(dict.fromkeys([*existing, *facts]))[:40]
        manager.set_peer_card(key, merged, peer="user")

        try:
            summary = (
                f"Active WhatsApp outreach to {name}: {task.get('owner_request', '')[:300]}. "
                f"Last message sent: {outbound[:200]}"
            )
            manager.create_conclusion(key, summary, peer="user")
        except Exception:
            pass
        return True
    except Exception:
        return False


def format_context_block(task: dict[str, Any], whatsapp_display_name: str = "") -> str:
    name = task.get("contact_name") or "the contact"
    display = whatsapp_display_name or task.get("whatsapp_display_name") or ""
    lines = [
        "[OUTREACH CONTEXT — authoritative; do not claim zero context]",
        f"- Contact (use this name): **{name}**",
        f"- JID: {task.get('jid', '')}",
    ]
    if display and display.lower() != name.lower():
        lines.append(
            f"- WhatsApp profile name is «{display}» — still call them **{name}** unless they correct you."
        )
    if task.get("owner_request"):
        lines.append(f"- Vignesh/Teddy asked: {task['owner_request'][:800]}")
    if task.get("last_outbound"):
        lines.append(f"- You already sent them: «{task['last_outbound'][:600]}»")
    lines.append(
        "- Do NOT ask what the meeting is about if the above already states it. "
        "Continue scheduling from their reply."
    )
    return "\n".join(lines)


def maybe_inject_whatsapp_outreach(
    *,
    platform: str,
    chat_id: str,
    user_display_name: str,
    session_id: str,
    history: list,
    context_prompt: str,
    session_store: Any,
) -> tuple[list, str]:
    """Prepend outreach context and seed transcript when the DM session is empty."""
    if (platform or "").lower() != "whatsapp":
        return history, context_prompt
    jid = normalize_wa_jid(chat_id)
    if is_owner_jid(jid):
        return history, context_prompt

    task = load_task(jid)
    if not task:
        return history, context_prompt

    if user_display_name:
        task["whatsapp_display_name"] = user_display_name
        try:
            task_path(jid).write_text(json.dumps(task, indent=2), encoding="utf-8")
        except OSError:
            pass

    block = format_context_block(task, user_display_name)
    mem_md = _home() / "memories" / f"outreach-{jid_digits(jid)}.md"
    if mem_md.is_file():
        try:
            block += "\n\n[OUTREACH MEMORY FILE]\n" + mem_md.read_text(encoding="utf-8")[:3000]
        except OSError:
            pass
    context_prompt = block + "\n\n" + (context_prompt or "")

    sync_outreach_to_honcho(task)

    outbound = (task.get("last_outbound") or "").strip()
    if not outbound:
        return history, context_prompt

    snippet = outbound[:80]
    already = any(
        snippet and snippet in str(m.get("content") or "")
        for m in history
        if m.get("role") == "assistant"
    )
    if not already and session_id and session_store is not None:
        try:
            session_store.append_to_transcript(
                session_id,
                {
                    "role": "assistant",
                    "content": outbound,
                    "mirror": True,
                    "mirror_source": "outreach_seed",
                },
            )
            history = session_store.load_transcript(session_id)
        except Exception:
            pass

    return history, context_prompt
