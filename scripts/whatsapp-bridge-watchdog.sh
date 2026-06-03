#!/usr/bin/env bash
# Restart whatsmeow-bridge (+ hermes-gateway) when health says not send-ready.
# Requires bridge patch with sendReady in GET /health JSON.
set -euo pipefail

BRIDGE_URL="${BRIDGE_URL:-http://127.0.0.1:3000/health}"
LOG="${WATCHDOG_LOG:-/root/.hermes/logs/whatsapp-bridge-watchdog.log}"
RESTART_GATEWAY="${RESTART_GATEWAY:-1}"

mkdir -p "$(dirname "$LOG")"
ts() { date -u '+%Y-%m-%dT%H:%M:%SZ'; }

health="$(curl -sf --max-time 10 "$BRIDGE_URL" 2>/dev/null || true)"
if [ -z "$health" ]; then
  echo "$(ts) health fetch failed — restarting bridge" >>"$LOG"
  systemctl restart whatsmeow-bridge
  sleep 8
  if [ "$RESTART_GATEWAY" = "1" ]; then
    systemctl restart hermes-gateway
  fi
  exit 0
fi

send_ready="$(printf '%s' "$health" | python3 -c "import sys,json; d=json.load(sys.stdin); print('true' if d.get('sendReady') else 'false')" 2>/dev/null || echo false)"
status="$(printf '%s' "$health" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('status',''))" 2>/dev/null || echo "")"

if [ "$send_ready" = "true" ]; then
  exit 0
fi

echo "$(ts) not send-ready status=$status health=$health — restarting bridge" >>"$LOG"
systemctl restart whatsmeow-bridge
sleep 8

health2="$(curl -sf --max-time 10 "$BRIDGE_URL" 2>/dev/null || true)"
send_ready2="$(printf '%s' "$health2" | python3 -c "import sys,json; d=json.load(sys.stdin); print('true' if d.get('sendReady') else 'false')" 2>/dev/null || echo false)"
echo "$(ts) after restart sendReady=$send_ready2 health=$health2" >>"$LOG"

if [ "$RESTART_GATEWAY" = "1" ]; then
  systemctl restart hermes-gateway
fi
