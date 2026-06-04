"""Verify assistant claims against logs; optional fast LLM."""
from __future__ import annotations

import json
import re
import urllib.request
from dataclasses import dataclass, field
from pathlib import Path

from .claims import (
    extract_phones,
    has_amnesia_claim,
    has_file_claim,
    has_send_claim,
    needs_verification,
)
from .evidence import TurnEvidence, collect, jid_digits, normalize_jid


@dataclass
class VerifyIssue:
    claim: str
    status: str  # failed | unverified
    proof: str


@dataclass
class VerifyResult:
    ok: bool
    issues: list[VerifyIssue] = field(default_factory=list)
    evidence_summary: str = ""
    used_llm: bool = False


def _owner_chat(chat_id: str) -> bool:
    d = jid_digits(chat_id)
    return d in {"6590013157", "8801521207499"}


def _third_party_targets(user_message: str, response: str, owner_chat: str) -> list[str]:
    owner = jid_digits(owner_chat)
    targets: list[str] = []
    for jid in extract_phones(f"{user_message}\n{response}"):
        if jid_digits(jid) != owner:
            targets.append(normalize_jid(jid))
    return list(dict.fromkeys(targets))


def deterministic_verify(
    response: str,
    user_message: str,
    chat_id: str,
    ev: TurnEvidence,
) -> list[VerifyIssue]:
    issues: list[VerifyIssue] = []
    owner = normalize_jid(chat_id)
    targets = _third_party_targets(user_message, response, owner)

    if has_send_claim(response) or (targets and re.search(r"\bsent\b", response, re.I)):
        if ev.tool_turns == 0:
            issues.append(
                VerifyIssue(
                    claim="Outbound send claimed",
                    status="failed",
                    proof=(
                        f"agent.log Turn ended tool_turns=0 api_calls={ev.api_calls} "
                        f"(no send_message in this turn)"
                    ),
                )
            )
        for t in targets:
            td = jid_digits(t)
            bridge_hits = [j for j in ev.bridge_sent_to if jid_digits(j) == td]
            if not bridge_hits:
                issues.append(
                    VerifyIssue(
                        claim=f"Message to {t}",
                        status="failed",
                        proof=(
                            f"No bridge 'Sent to {t}' in last window. "
                            f"Bridge sent: {ev.bridge_sent_to or 'none'}"
                        ),
                    )
                )
        if ev.send_tool_err:
            issues.append(
                VerifyIssue(
                    claim="send_message tool",
                    status="failed",
                    proof=ev.send_tool_err[-1],
                )
            )

    if has_file_claim(response):
        if "MEDIA:" not in response and not re.search(r"reports/", response, re.I):
            issues.append(
                VerifyIssue(
                    claim="File attachment claimed",
                    status="unverified",
                    proof="No MEDIA: line or reports/ path in response text",
                )
            )

    return issues


def _load_outreach_task(chat_id: str) -> dict | None:
    try:
        import sys
        from pathlib import Path

        home = Path.home() / ".hermes"
        scripts = home / "scripts"
        if str(scripts) not in sys.path:
            sys.path.insert(0, str(scripts))
        from outreach_tasks import load_task

        return load_task(chat_id)
    except Exception:
        return None


def verify_amnesia(
    response: str,
    chat_id: str,
    user_message: str,
) -> list[VerifyIssue]:
    """Flag false 'no context' when outreach task file proves prior outbound."""
    issues: list[VerifyIssue] = []
    if not has_amnesia_claim(response):
        return issues

    task = _load_outreach_task(chat_id)
    if not task and not _owner_chat(chat_id):
        for jid in extract_phones(user_message + "\n" + response):
            if jid_digits(jid) not in {"6590013157", "8801521207499"}:
                task = _load_outreach_task(jid)
                if task:
                    break

    if not task:
        return issues

    issues.append(
        VerifyIssue(
            claim="Amnesia / zero-context claim",
            status="failed",
            proof=(
                f"Outreach task exists for {task.get('jid')}: contact_name={task.get('contact_name')!r}; "
                f"last_outbound={str(task.get('last_outbound', ''))[:200]!r}"
            ),
        )
    )
    return issues


def _load_openrouter_key() -> str:
    env_path = Path.home() / ".hermes" / ".env"
    if not env_path.is_file():
        return ""
    for line in env_path.read_text(encoding="utf-8").splitlines():
        if line.startswith("OPENROUTER_API_KEY="):
            return line.split("=", 1)[1].strip().strip('"').strip("'")
    return ""


def llm_verify(
    model: str,
    response: str,
    user_message: str,
    ev: TurnEvidence,
    det_issues: list[VerifyIssue],
) -> list[VerifyIssue]:
    key = _load_openrouter_key()
    if not key:
        return det_issues

    prompt = {
        "task": "Verify whether the assistant's claims match the evidence.",
        "user_message": user_message[:1500],
        "assistant_response": response[:2500],
        "deterministic_issues": [{"claim": i.claim, "proof": i.proof} for i in det_issues],
        "evidence": {
            "tool_turns": ev.tool_turns,
            "send_tool_ok": ev.send_tool_ok[-3:],
            "send_tool_err": ev.send_tool_err[-3:],
            "bridge_sent_to": ev.bridge_sent_to,
            "gateway_outbound": ev.gateway_outbound,
        },
        "output_schema": {
            "verified": "bool — true only if every material claim is supported",
            "issues": [{"claim": "str", "status": "failed|ok", "proof": "str"}],
        },
    }
    payload = {
        "model": model,
        "messages": [
            {
                "role": "system",
                "content": (
                    "You are a strict action verifier. Output JSON only. "
                    "Flag false 'sent/delivered/booked' claims when logs contradict."
                ),
            },
            {"role": "user", "content": json.dumps(prompt)},
        ],
        "temperature": 0,
        "max_tokens": 400,
    }
    req = urllib.request.Request(
        "https://openrouter.ai/api/v1/chat/completions",
        data=json.dumps(payload).encode(),
        headers={
            "Authorization": f"Bearer {key}",
            "Content-Type": "application/json",
        },
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=25) as resp:
            data = json.loads(resp.read().decode())
        text = data["choices"][0]["message"]["content"]
        m = re.search(r"\{.*\}", text, re.S)
        if not m:
            return det_issues
        parsed = json.loads(m.group(0))
        if parsed.get("verified") is True and not det_issues:
            return []
        out: list[VerifyIssue] = list(det_issues)
        for item in parsed.get("issues") or []:
            if str(item.get("status", "")).lower() != "failed":
                continue
            out.append(
                VerifyIssue(
                    claim=str(item.get("claim", "claim")),
                    status="failed",
                    proof=str(item.get("proof", ""))[:500],
                )
            )
        return out
    except Exception:
        return det_issues


def verify_turn(
    *,
    response: str,
    user_message: str,
    chat_id: str,
    session_id: str,
    platform: str,
    cfg: dict,
) -> VerifyResult:
    if not cfg.get("enabled", True):
        return VerifyResult(ok=True)
    if platform not in (cfg.get("platforms") or ["whatsapp"]):
        return VerifyResult(ok=True)
    if (user_message or "").startswith("[INTERNAL VERIFIER"):
        return VerifyResult(ok=True)

    max_chars = int(cfg.get("max_response_chars", 4000))
    response = (response or "")[:max_chars]
    user_message = (user_message or "")[:max_chars]

    amnesia = verify_amnesia(response, chat_id, user_message)
    if amnesia:
        summary = "outreach_task=exists"
        use_llm = str(cfg.get("use_llm", "on_fail")).lower()
        if use_llm in ("always", "on_fail"):
            ev = collect(session_id, chat_id, int(cfg.get("evidence_window_sec", 300)))
            amnesia = llm_verify(
                str(cfg.get("model", "google/gemma-3-4b-it")),
                response,
                user_message,
                ev,
                amnesia,
            )
        return VerifyResult(ok=False, issues=amnesia, evidence_summary=summary, used_llm=bool(use_llm in ("always", "on_fail")))

    if not needs_verification(response, user_message):
        return VerifyResult(ok=True)

    ev = collect(session_id, chat_id, int(cfg.get("evidence_window_sec", 300)))
    summary = (
        f"tool_turns={ev.tool_turns} bridge={ev.bridge_sent_to} "
        f"send_err={len(ev.send_tool_err)}"
    )
    det = deterministic_verify(response, user_message, chat_id, ev)
    issues = det

    use_llm = str(cfg.get("use_llm", "on_fail")).lower()
    run_llm = use_llm == "always" or (use_llm == "on_fail" and bool(det))
    if run_llm:
        issues = llm_verify(str(cfg.get("model", "google/gemma-3-4b-it")), response, user_message, ev, det)
        return VerifyResult(ok=not issues, issues=issues, evidence_summary=summary, used_llm=True)

    return VerifyResult(ok=not issues, issues=issues, evidence_summary=summary)


def format_correction(issues: list[VerifyIssue], evidence_summary: str) -> str:
    lines = [
        "[INTERNAL VERIFIER — not from the user; do not quote this block verbatim]",
        "Your last reply made claims that are NOT supported by logs/tools in this turn.",
        "",
    ]
    for i, issue in enumerate(issues, 1):
        lines.append(f"{i}. {issue.claim}: {issue.proof}")
    lines.extend(
        [
            "",
            f"Evidence snapshot: {evidence_summary}",
            "",
            "Required now:",
            "- Read ~/.hermes/tasks/outreach/<digits>.json and use contact_name + last_outbound.",
            "- If you claimed you messaged someone: call send_message to whatsapp:+<E.164> and confirm success.",
            "- If the date was wrong: run TZ=Asia/Singapore date, fix the outbound text, then send.",
            "- Reply in ONE short message — correction only, no false confidence or fake amnesia.",
        ]
    )
    return "\n".join(lines)
