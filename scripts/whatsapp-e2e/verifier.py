#!/usr/bin/env python3
"""Score Stella workflow evidence 1-10; require >= min_score to pass."""
from __future__ import annotations

import json
import re
import subprocess
from dataclasses import dataclass, field


@dataclass
class VerifyResult:
    score: float
    passed: bool
    feedback: str
    breakdown: dict[str, float] = field(default_factory=dict)


def _has_any(text: str, *needles: str) -> bool:
    low = text.lower()
    return any(n.lower() in low for n in needles)


def _count_present(text: str, needles: list[str]) -> int:
    low = text.lower()
    return sum(1 for n in needles if n.lower() in low)


def score_gw1(evidence: dict) -> VerifyResult:
    blob = evidence.get("blob", "")
    breakdown: dict[str, float] = {}
    notes: list[str] = []

    breakdown["plan"] = 2.0 if _has_any(blob, "plan", "step 1", "1.") else 0.0
    if breakdown["plan"] < 1:
        notes.append("Missing numbered PLAN before execution.")

    breakdown["group_api"] = 0.0
    if _has_any(blob, "/groups/create", "groups/create", "created group", "groupjid"):
        breakdown["group_api"] = 2.0
    elif "@g.us" in blob:
        breakdown["group_api"] = 1.0
    else:
        notes.append("No bridge group create (/groups/create or @g.us in logs).")

    breakdown["participants"] = 2.0 if _count_present(blob, ["6590016046", "8801521207499"]) >= 2 else 0.5
    if breakdown["participants"] < 2:
        notes.append("Both test numbers (+6590016046, +8801521207499) not reflected.")

    breakdown["group_message"] = 2.0 if "E2E-GW1-LIVE" in blob else 0.0
    if breakdown["group_message"] < 1:
        notes.append('Group post must include exact text "E2E-GW1-LIVE".')

    breakdown["done"] = 1.0 if _has_any(blob, "done", "complete", "finished") else 0.0
    breakdown["no_legacy"] = 1.0 if not _has_any(blob, "wabot", ":7777", "/opt/wabot") else 0.0
    if breakdown["no_legacy"] < 1:
        notes.append("Response still references legacy wabot stack.")

    score = min(10.0, sum(breakdown.values()))
    return VerifyResult(score=score, passed=score >= 6, feedback="; ".join(notes) or "ok", breakdown=breakdown)


def score_gw2(evidence: dict) -> VerifyResult:
    blob = evidence.get("blob", "")
    breakdown: dict[str, float] = {}
    notes: list[str] = []

    breakdown["plan"] = 2.0 if _has_any(blob, "plan", "step") else 0.0
    breakdown["group"] = 2.0 if "@g.us" in blob and _has_any(blob, "gw2", "groups/create", "created group") else 0.0
    breakdown["poll_in_group"] = 0.0
    if _has_any(blob, "sent to", "120363") and "@g.us" in blob:
        if _has_any(blob, "prefer", "meeting", "time", "sgt", "slot"):
            breakdown["poll_in_group"] = 2.0
    elif "@g.us" in blob:
        breakdown["poll_in_group"] = 1.0
    if breakdown["poll_in_group"] < 2:
        notes.append(
            "Poll must be sent to @g.us (bridge Sent to …@g.us), not only in owner DM."
        )
    breakdown["two_numbers"] = 2.0 if _count_present(blob, ["6590016046", "8801521207499"]) >= 2 else 0.5
    breakdown["done"] = 1.0 if "done" in blob.lower() else 0.0
    breakdown["no_legacy"] = 1.0 if not _has_any(blob, "wabot", ":7777") else 0.0

    score = min(10.0, sum(breakdown.values()))
    return VerifyResult(score=score, passed=score >= 6, feedback="; ".join(notes) or "ok", breakdown=breakdown)


def score_gw3(evidence: dict) -> VerifyResult:
    blob = evidence.get("blob", "")
    breakdown: dict[str, float] = {}
    notes: list[str] = []

    breakdown["plan"] = 1.5 if _has_any(blob, "plan", "step 1", "1)") else 0.0
    breakdown["group"] = 1.5 if "@g.us" in blob else 0.0
    breakdown["participants"] = 1.0 if _count_present(blob, ["6590016046", "8801521207499"]) >= 2 else 0.0
    breakdown["preferences"] = 1.5 if _has_any(blob, "prefer", "12:00", "sgt", "meeting time") else 0.0
    breakdown["booking"] = 1.5 if _has_any(blob, "calendar", "composio", "event", "booked", "scheduled") else 0.0
    if breakdown["booking"] < 1:
        notes.append("No calendar/booking evidence (Composio/Google Calendar).")
    breakdown["email"] = 1.5 if _has_any(blob, "gmail", "email", "invite", "sent mail") else 0.0
    if breakdown["email"] < 1:
        notes.append("No email invite evidence (Gmail); partial credit if noted unavailable.")
        if _has_any(blob, "would send", "email unavailable", "no gmail"):
            breakdown["email"] = 0.8
    breakdown["group_confirm"] = 1.0 if _has_any(blob, "12:00", "tomorrow", "confirm") and "@g.us" in blob else 0.0
    breakdown["done"] = 0.5 if "done" in blob.lower() else 0.0
    breakdown["no_legacy"] = 1.0 if not _has_any(blob, "wabot", ":7777") else 0.0

    score = min(10.0, sum(breakdown.values()))
    return VerifyResult(score=score, passed=score >= 6, feedback="; ".join(notes) or "ok", breakdown=breakdown)


def score_messy_lead(evidence: dict) -> VerifyResult:
    blob = evidence.get("blob", "")
    breakdown: dict[str, float] = {}
    notes: list[str] = []

    breakdown["no_stall"] = 0.0 if _has_any(
        blob,
        "please provide",
        "need the following",
        "reformat your",
        "fill in the",
        "missing required fields",
    ) and "plan" not in blob.lower() else 2.0
    if breakdown["no_stall"] < 1:
        notes.append("Stalled asking for template fields instead of inferring.")

    breakdown["plan_or_assume"] = 2.0 if _has_any(
        blob, "plan", "assumption", "assuming", "step 1", "1."
    ) else 0.5

    breakdown["tools_or_file"] = 0.0
    if _has_any(blob, "tool_", "delegate", "web", "reports/lead", ".md"):
        breakdown["tools_or_file"] = 2.5
    elif _has_any(blob, "research", "maps", "review", "epicware", "fit"):
        breakdown["tools_or_file"] = 1.0
    else:
        notes.append("No evidence of research tools or report file.")

    breakdown["substance"] = 2.0 if _has_any(
        blob, "fit", "tier", "review", "maps", "salon", "vasan", "score"
    ) else 0.0
    breakdown["no_outbound_lead"] = 1.5 if "sent to 6583669443" not in blob.lower() else 0.0
    breakdown["done"] = 0.5 if _has_any(blob, "done", "gaps", "summary") else 0.0

    score = min(10.0, sum(breakdown.values()))
    return VerifyResult(score=score, passed=score >= 6, feedback="; ".join(notes) or "ok", breakdown=breakdown)


def score_messy_group(evidence: dict) -> VerifyResult:
    blob = evidence.get("blob", "")
    group_jid = evidence.get("expected_group_jid") or evidence.get("group_jid") or ""
    breakdown: dict[str, float] = {}
    notes: list[str] = []

    breakdown["infer"] = 2.0 if _has_any(blob, "plan", "assumption", "manual", "120363", "@g.us") else 0.5
    sent_group = False
    if group_jid:
        sent_group = f"sent to {group_jid.lower()}" in blob.lower()
    if not sent_group:
        sent_group = bool(re.search(r"sent to \d+[^@\n]*@g\.us", blob, re.I))
    breakdown["group_send"] = 2.5 if sent_group and _has_any(blob, "prefer", "20", "sgt", "slot", "meeting") else 0.0
    if breakdown["group_send"] < 1:
        notes.append("Poll must hit a @g.us (bridge Sent to …@g.us), not owner DM only.")
    if "sent to 8801521207499" in blob.lower() and not sent_group:
        breakdown["group_send"] = 0.0
        notes.append("Message went to Teddy DM only.")
    breakdown["mentions"] = 1.5 if _has_any(blob, "6590016046", "8801521207499", "gaya", "teddy") else 0.0
    breakdown["no_stall"] = 2.0 if not _has_any(blob, "which group", "provide jid", "reformat") else 0.5
    breakdown["done"] = 1.0 if "done" in blob.lower() else 0.0

    score = min(10.0, sum(breakdown.values()))
    return VerifyResult(score=score, passed=score >= 6, feedback="; ".join(notes) or "ok", breakdown=breakdown)


RUBRICS = {
    "gw1": score_gw1,
    "gw2": score_gw2,
    "gw3": score_gw3,
    "messy_lead": score_messy_lead,
    "messy_group": score_messy_group,
}


def llm_verify(evidence: dict, rubric_name: str, ssh_host: str = "vignesh") -> VerifyResult | None:
    """Optional OpenRouter verifier on VPS (uses ~/.hermes/.env key)."""
    prompt = {
        "task": rubric_name,
        "evidence_excerpt": evidence.get("blob", "")[-12000:],
        "instructions": (
            "Score 1-10 whether the agent completed the multi-step WhatsApp group workflow. "
            "Return JSON only: {\"score\": number, \"feedback\": string, \"breakdown\": {}}"
        ),
    }
    script = f"""python3 - <<'PY'
import json, os, re, urllib.request
from pathlib import Path
env = Path("/root/.hermes/.env").read_text()
key = ""
for line in env.splitlines():
    if line.startswith("OPENROUTER_API_KEY="):
        key = line.split("=",1)[1].strip().strip('"').strip("'")
        break
if not key:
    print(json.dumps({{"error": "no key"}}))
    raise SystemExit(0)
payload = {json.dumps({
    "model": "google/gemma-3-27b-it:free",
    "messages": [
        {"role": "system", "content": "You are a strict E2E verifier. Output JSON only."},
        {"role": "user", "content": json.dumps(prompt)},
    ],
})}
req = urllib.request.Request(
    "https://openrouter.ai/api/v1/chat/completions",
    data=json.dumps(payload).encode(),
    headers={{"Authorization": f"Bearer {{key}}", "Content-Type": "application/json"}},
    method="POST",
)
try:
    with urllib.request.urlopen(req, timeout=60) as resp:
        data = json.loads(resp.read().decode())
    text = data["choices"][0]["message"]["content"]
    m = re.search(r"\\{{.*\\}}", text, re.S)
    print(m.group(0) if m else text[:500])
except Exception as e:
    print(json.dumps({{"error": str(e)}}))
PY"""
    r = subprocess.run(["ssh", ssh_host, script], capture_output=True, text=True, timeout=90)
    if r.returncode != 0 or not r.stdout.strip():
        return None
    try:
        parsed = json.loads(r.stdout.strip())
        if "error" in parsed:
            return None
        score = float(parsed.get("score", 0))
        feedback = str(parsed.get("feedback", ""))
        return VerifyResult(
            score=score,
            passed=score >= 6,
            feedback=feedback,
            breakdown=parsed.get("breakdown") or {},
        )
    except (json.JSONDecodeError, TypeError, ValueError):
        return None


def verify(evidence: dict, rubric: str, min_score: float = 6, use_llm: bool = False) -> VerifyResult:
    fn = RUBRICS.get(rubric)
    if not fn:
        return VerifyResult(0, False, f"unknown rubric {rubric}")
    base = fn(evidence)
    if use_llm:
        llm = llm_verify(evidence, rubric)
        if llm:
            combined = (base.score + llm.score) / 2
            feedback = f"heuristic={base.score:.1f} llm={llm.score:.1f}. {llm.feedback or base.feedback}"
            return VerifyResult(
                score=combined,
                passed=combined >= min_score,
                feedback=feedback,
                breakdown={"heuristic": base.score, "llm": llm.score},
            )
    base.passed = base.score >= min_score
    return base
