#!/usr/bin/env python3
"""Group creation/maintenance E2E with verifier loop (score >= 6, retry until pass)."""
from __future__ import annotations

import argparse
import base64
import json
import re
import sys
import time
import uuid
from pathlib import Path

from verifier import VerifyResult, verify

ROOT = Path(__file__).resolve().parent
PERSONAS = json.loads((ROOT / "personas.json").read_text())
def _load_workflow_config(messy: bool) -> dict:
    name = "messy_workflows.json" if messy else "group_workflows.json"
    return json.loads((ROOT / name).read_text())


DEFAULTS: dict = {}

# Reuse ssh/inject helpers from run_tests
sys.path.insert(0, str(ROOT))
from run_tests import (  # noqa: E402
    AGENT_LOG,
    BRIDGE,
    GATEWAY_LOG,
    SSH_HOST,
    bridge_sent_since,
    inject,
    log_tail,
    ssh,
    wait_log_substr,
    wait_response_after_marker,
)

INTER_RETRY_DELAY = 15


def inject_group(
    persona_key: str,
    body: str,
    group_jid: str,
    msg_id: str | None = None,
) -> str:
    """Inject inbound message into a group chat."""
    p = PERSONAS[persona_key]
    payload = {
        "messageId": msg_id or f"e2e-grp-{uuid.uuid4().hex[:12]}",
        "chatId": group_jid,
        "senderId": p["senderId"],
        "senderName": p["senderName"],
        "chatName": "E2E Group",
        "isGroup": True,
        "body": body,
    }
    data = json.dumps(payload)
    b64 = base64.b64encode(data.encode()).decode()
    cmd = (
        f"echo {b64} | base64 -d | "
        f"curl -sS -X POST {BRIDGE}/inject -H 'Content-Type: application/json' -d @-"
    )
    r = ssh(cmd, timeout=60)
    if r.returncode != 0:
        raise RuntimeError(f"group inject failed: {r.stderr or r.stdout}")
    out = json.loads(r.stdout.strip() or "{}")
    if not out.get("success"):
        raise RuntimeError(f"group inject rejected: {r.stdout}")
    return payload["messageId"]


def extract_group_jid(blob: str) -> str | None:
    m = re.search(r"\b(\d{10,24}-\d{6,12}@g\.us)\b", blob)
    if m:
        return m.group(1)
    m = re.search(r"\b(\d{10,24}@g\.us)\b", blob)
    return m.group(1) if m else None


def collect_evidence(marker: str) -> dict:
    gw = log_tail(800)
    ag = ssh(f"tail -800 {AGENT_LOG}", timeout=30).stdout or ""
    br = bridge_sent_since(marker)
    idx = gw.find(marker)
    window_gw = gw[idx:] if idx >= 0 else gw
    idx_a = ag.find(marker)
    window_ag = ag[idx_a:] if idx_a >= 0 else ag
    blob = "\n".join([window_gw, window_ag, br])
    return {
        "marker": marker,
        "blob": blob,
        "gateway": window_gw,
        "agent": window_ag,
        "bridge": br,
        "group_jid": extract_group_jid(blob),
    }


def wait_for_group_jid(marker: str, max_wait: int = 180) -> str | None:
    deadline = time.time() + max_wait
    while time.time() < deadline:
        ev = collect_evidence(marker)
        if ev.get("group_jid"):
            return ev["group_jid"]
        time.sleep(15)
    return None


def run_simulated_replies(wf: dict, evidence: dict) -> None:
    group_jid = evidence.get("group_jid")
    if not group_jid:
        return
    for sim in wf.get("simulate_replies") or []:
        time.sleep(int(sim.get("delay_sec", 30)))
        persona = sim["persona"]
        body = sim["body"]
        try:
            inject_group(persona, body, group_jid)
            print(f"  sim inject {persona} -> {group_jid}", flush=True)
        except Exception as e:
            print(f"  sim inject failed: {e}", flush=True)


def build_prompt(wf: dict, marker: str, attempt: int, feedback: str) -> str:
    base = wf["prompt"]
    if attempt == 0:
        return f"[{marker}] {base}"
    return (
        f"[{marker}] [VERIFIER RETRY {attempt}] Previous score below 6. "
        f"Fix these issues: {feedback}\n\nOriginal task:\n{base}"
    )


def run_workflow(wf: dict, use_llm: bool) -> tuple[bool, list[dict]]:
    defaults = DEFAULTS
    persona = wf.get("persona", defaults["persona"])
    chat = PERSONAS[persona]["chatId"]
    min_score = float(wf.get("min_score", defaults["min_score"]))
    max_retries = int(wf.get("max_retries", defaults["max_retries"]))
    timeout = int(wf.get("response_timeout_sec", defaults["response_timeout_sec"]))
    post_wait = int(wf.get("post_response_wait_sec", defaults.get("post_response_wait_sec", 60)))

    # Patch global timeout in wait_response — use local loop
    attempts_log: list[dict] = []

    for attempt in range(max_retries):
        marker = f"E2E-{wf['id']}-{uuid.uuid4().hex[:6]}"
        feedback = attempts_log[-1].get("feedback", "") if attempts_log else ""
        body = build_prompt(wf, marker, attempt, feedback)
        print(f"  attempt {attempt + 1}/{max_retries} marker={marker}", flush=True)

        try:
            inject(persona, body)
        except Exception as e:
            attempts_log.append({"attempt": attempt, "error": str(e), "score": 0})
            time.sleep(INTER_RETRY_DELAY)
            continue

        time.sleep(10)
        if not wait_log_substr(marker, timeout=60):
            attempts_log.append(
                {"attempt": attempt, "score": 0, "feedback": "inbound marker never logged"}
            )
            time.sleep(INTER_RETRY_DELAY)
            continue

        if wf.get("simulate_replies"):
            gjid = wait_for_group_jid(marker, max_wait=min(180, timeout // 2))
            if gjid:
                run_simulated_replies(wf, {"group_jid": gjid})
            else:
                print("  warn: no group JID yet for simulated replies", flush=True)

        deadline = time.time() + timeout
        responded = False
        while time.time() < deadline:
            log = log_tail(300)
            if marker not in log:
                time.sleep(5)
                continue
            idx = log.find(marker)
            window = log[idx:]
            if f"chat={chat}" in window and "response ready" in window:
                responded = True
                break
            time.sleep(8)

        if not responded:
            attempts_log.append(
                {"attempt": attempt, "score": 0, "feedback": f"timeout {timeout}s no response"}
            )
            time.sleep(INTER_RETRY_DELAY)
            continue

        time.sleep(post_wait)
        evidence = collect_evidence(marker)
        if wf.get("requires_group_jid"):
            evidence["expected_group_jid"] = wf["requires_group_jid"]
        result: VerifyResult = verify(
            evidence, wf["rubric"], min_score=min_score, use_llm=use_llm
        )
        entry = {
            "attempt": attempt,
            "score": round(result.score, 2),
            "passed": result.passed,
            "feedback": result.feedback,
            "breakdown": result.breakdown,
            "group_jid": evidence.get("group_jid"),
        }
        attempts_log.append(entry)
        print(
            f"  verifier score={result.score:.1f}/10 passed={result.passed} — {result.feedback}",
            flush=True,
        )
        if result.passed:
            return True, attempts_log
        time.sleep(INTER_RETRY_DELAY)

    return False, attempts_log


def deploy_prereqs() -> None:
    import subprocess

    scripts = ROOT.parent
    for name in (
        "patch-whatsmeow-bridge-inject.py",
        "patch-whatsmeow-bridge-groups.py",
        "patch-hermes-soul-multistep-plan.py",
        "patch-hermes-soul-messy-prompts.py",
        "patch-hermes-soul-whatsapp-system-instruction.py",
    ):
        path = scripts / name
        if path.is_file():
            subprocess.run(["scp", "-q", str(path), f"{SSH_HOST}:/tmp/"], check=True)
            ssh(f"python3 /tmp/{name}", timeout=60)
    r = ssh(
        "cd /opt/whatsmeow-bridge && go build -o whatsmeow-bridge . 2>&1",
        timeout=180,
    )
    if r.returncode != 0:
        raise RuntimeError(f"go build failed: {r.stdout}\n{r.stderr}")
    ssh("systemctl restart whatsmeow-bridge", timeout=60)
    time.sleep(3)
    ssh("systemctl restart hermes-gateway", timeout=60)
    time.sleep(5)


def main() -> int:
    ap = argparse.ArgumentParser(description="Group workflow E2E with verifier loop")
    ap.add_argument("--workflow", action="append", help="Run only GW1, GW2, GW3, M1, M2")
    ap.add_argument("--messy", action="store_true", help="Run messy_workflows.json (M*)")
    ap.add_argument("--deploy", action="store_true", help="Deploy bridge + SOUL patches first")
    ap.add_argument("--llm", action="store_true", help="Blend OpenRouter verifier score")
    ap.add_argument("--skip-deploy", action="store_true")
    args = ap.parse_args()

    if args.deploy and not args.skip_deploy:
        print("==> Deploy patches", flush=True)
        deploy_prereqs()

    cfg = _load_workflow_config(args.messy)
    global DEFAULTS
    DEFAULTS = cfg["defaults"]
    workflows = cfg["workflows"]
    if args.workflow:
        wanted = set(args.workflow)
        workflows = [w for w in workflows if w["id"] in wanted]

    results: list[dict] = []
    for wf in workflows:
        print(f"\nRUN {wf['id']}: {wf['title']}", flush=True)
        ok, attempts = run_workflow(wf, use_llm=args.llm)
        results.append({"id": wf["id"], "ok": ok, "attempts": attempts})
        print(f"  {'PASS' if ok else 'FAIL'} {wf['id']}", flush=True)

    report_dir = ROOT / "reports"
    report_dir.mkdir(exist_ok=True)
    ts = time.strftime("%Y%m%d-%H%M%S")
    prefix = "messy-run" if args.messy else "group-run"
    report_path = report_dir / f"{prefix}-{ts}.json"
    report_path.write_text(json.dumps({"workflows": results}, indent=2))
    print(f"\nReport: {report_path}")

    failed = [r["id"] for r in results if not r["ok"]]
    if failed:
        print(f"FAILED workflows: {', '.join(failed)}", file=sys.stderr)
        return 1
    print("ALL GROUP WORKFLOWS PASS")
    return 0


if __name__ == "__main__":
    sys.exit(main())
