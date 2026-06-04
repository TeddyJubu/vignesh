#!/usr/bin/env bash
# Quick post-hermes-update check for Stella VPS patches.
set -euo pipefail

FAIL=0
check() {
  if eval "$2" >/dev/null 2>&1; then
    echo "  ok: $1"
  else
    echo "  FAIL: $1"
    FAIL=1
  fi
}

echo "==> Hermes version"
hermes version 2>/dev/null | head -2 || true

echo "==> Core patches in installed Hermes"
check "send_message outreach" 'rg -q "record_whatsapp_outreach" /usr/local/lib/hermes-agent/tools/send_message_tool.py'
check "send_message home guard" 'rg -q "Refusing home-channel" /usr/local/lib/hermes-agent/tools/send_message_tool.py'
check "gateway outreach inject" 'rg -q "maybe_inject_whatsapp_outreach" /usr/local/lib/hermes-agent/gateway/run.py'
check "gateway agent:end verifier" 'rg -q "response_full" /usr/local/lib/hermes-agent/gateway/run.py'
check "gateway sanitize" 'rg -q "_looks_like_leaked_agent_tool_markup" /usr/local/lib/hermes-agent/gateway/run.py'

echo "==> Duplicate outreach block"
COUNT=$(rg -c "from outreach_tasks import record_whatsapp_outreach" /usr/local/lib/hermes-agent/tools/send_message_tool.py 2>/dev/null || echo 0)
if [ "$COUNT" -eq 1 ]; then
  echo "  ok: single outreach record block"
else
  echo "  FAIL: outreach record appears $COUNT times (expected 1)"
  FAIL=1
fi

echo "==> Hooks"
for h in action-verifier reapply-gateway-sanitize reapply-whatsapp-send-patch; do
  check "hook $h" "test -f /root/.hermes/hooks/$h/handler.py"
done

echo "==> Scripts on VPS"
check "outreach_tasks.py" 'test -f /root/.hermes/scripts/outreach_tasks.py'
check "honcho-ai in Hermes venv" '/usr/local/lib/hermes-agent/venv/bin/python -c "from honcho import Honcho"'

echo "==> Honcho memory injection config"
check "honcho contextTokens 8000" 'python3 -c "import json; d=json.load(open(\"/root/.hermes/honcho.json\")); assert d.get(\"contextTokens\")==8000"'
check "honcho every-turn injection" 'grep -q "\"injectionFrequency\": \"every-turn\"" /root/.hermes/honcho.json'

echo "==> Services"
check "hermes-gateway" 'systemctl is-active --quiet hermes-gateway'
check "whatsmeow-bridge" 'systemctl is-active --quiet whatsmeow-bridge'
curl -sf http://127.0.0.1:3000/health | python3 -c "import sys,json; d=json.load(sys.stdin); assert d.get('sendReady'), d; print('  ok: bridge sendReady')" || { echo "  FAIL: bridge health"; FAIL=1; }

if [ "$FAIL" -eq 0 ]; then
  echo "==> All checks passed"
else
  echo "==> Some checks failed — run deploy-stella-intelligence.sh or install hooks"
  exit 1
fi
