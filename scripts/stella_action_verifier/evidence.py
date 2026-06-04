"""Collect proof from agent.log and whatsmeow-bridge journal."""
from __future__ import annotations

import re
import subprocess
from dataclasses import dataclass, field
from datetime import datetime, timedelta, timezone
from pathlib import Path


@dataclass
class TurnEvidence:
    session_id: str = ""
    tool_turns: int = -1
    api_calls: int = -1
    send_tool_ok: list[str] = field(default_factory=list)
    send_tool_err: list[str] = field(default_factory=list)
    bridge_sent_to: list[str] = field(default_factory=list)
    gateway_outbound: list[str] = field(default_factory=list)
    raw_lines: list[str] = field(default_factory=list)


TURN_ENDED_RE = re.compile(
    r"Turn ended:.*tool_turns=(\d+).*api_calls=(\d+).*session=(\S+)",
)
SEND_TOOL_OK_RE = re.compile(r"tool send_message completed", re.I)
SEND_TOOL_ERR_RE = re.compile(r"Tool send_message returned error.*?:\s*(\{.*\})", re.I)
BRIDGE_SENT_RE = re.compile(r"Sent to (\S+):", re.I)
GATEWAY_SEND_RE = re.compile(r"Sending response \(\d+ chars\) to (\S+)", re.I)


def _hermes_home() -> Path:
    import os

    return Path(os.environ.get("HERMES_HOME", Path.home() / ".hermes"))


def _since_line_ok(line: str, since: datetime) -> bool:
    m = re.match(r"(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})", line)
    if not m:
        return True
    try:
        ts = datetime.strptime(m.group(1), "%Y-%m-%d %H:%M:%S").replace(tzinfo=timezone.utc)
        return ts >= since
    except ValueError:
        return True


def collect(session_id: str, chat_id: str, window_sec: int = 300) -> TurnEvidence:
    since = datetime.now(timezone.utc) - timedelta(seconds=window_sec)
    ev = TurnEvidence(session_id=session_id)
    agent_log = _hermes_home() / "logs" / "agent.log"
    gateway_log = _hermes_home() / "logs" / "gateway.log"

    for path in (agent_log, gateway_log):
        if not path.is_file():
            continue
        try:
            lines = path.read_text(encoding="utf-8", errors="replace").splitlines()[-8000:]
        except OSError:
            continue
        for line in lines:
            if not _since_line_ok(line, since):
                continue
            if session_id and session_id not in line and path == agent_log:
                continue
            if chat_id and chat_id not in line and path == gateway_log:
                continue
            ev.raw_lines.append(line[:500])
            tm = TURN_ENDED_RE.search(line)
            if tm:
                ev.tool_turns = int(tm.group(1))
                ev.api_calls = int(tm.group(2))
            if SEND_TOOL_OK_RE.search(line):
                ev.send_tool_ok.append(line.strip()[-400:])
            merr = SEND_TOOL_ERR_RE.search(line)
            if merr:
                ev.send_tool_err.append(merr.group(1)[:400])
            gs = GATEWAY_SEND_RE.search(line)
            if gs:
                ev.gateway_outbound.append(gs.group(1))

    try:
        j = subprocess.run(
            [
                "journalctl",
                "-u",
                "whatsmeow-bridge",
                "--since",
                f"{int(window_sec)} sec ago",
                "--no-pager",
            ],
            capture_output=True,
            text=True,
            timeout=15,
            check=False,
        )
        for line in (j.stdout or "").splitlines():
            bm = BRIDGE_SENT_RE.search(line)
            if bm:
                ev.bridge_sent_to.append(bm.group(1))
    except (OSError, subprocess.TimeoutExpired):
        pass

    ev.bridge_sent_to = list(dict.fromkeys(ev.bridge_sent_to))
    ev.gateway_outbound = list(dict.fromkeys(ev.gateway_outbound))
    return ev


def normalize_jid(jid: str) -> str:
    j = (jid or "").strip().lower()
    if "@" not in j:
        d = re.sub(r"\D", "", j)
        return f"{d}@s.whatsapp.net" if d else j
    return j


def jid_digits(jid: str) -> str:
    return re.sub(r"\D", "", jid.split("@")[0])
