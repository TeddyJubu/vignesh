#!/usr/bin/env bash
# Deploy model + autonomy + anti-leak patches to vignesh Hermes.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HOST="${STELLA_SSH_HOST:-vignesh}"
HERMES_SCRIPTS="/root/.hermes/scripts"

PATCHES=(
  patch-hermes-config-minimax-m3.py
  patch-hermes-gateway-whatsapp-sanitize.py
  patch-hermes-agent-minimax-tool-leak.py
  patch-hermes-soul-autonomous-agent.py
  patch-hermes-soul-scrapling.py
  patch-hermes-soul-reports-delivery.py
  patch-hermes-soul-whatsapp-access-crosschat.py
  patch-hermes-soul-outbound-messaging.py
  patch-hermes-soul-third-party-session.py
  patch-hermes-gateway-media-reports-fallback.py
  patch-hermes-soul-messy-prompts.py
  patch-hermes-soul-whatsapp-system-instruction.py
)

echo "==> Copy patches to ${HOST}:${HERMES_SCRIPTS}"
ssh "$HOST" "mkdir -p ${HERMES_SCRIPTS}"
for p in "${PATCHES[@]}"; do
  scp -q "$ROOT/$p" "$HOST:${HERMES_SCRIPTS}/"
done

echo "==> Apply patches"
for p in "${PATCHES[@]}"; do
  ssh "$HOST" "python3 ${HERMES_SCRIPTS}/$(basename "$p")"
done

echo "==> Bridge connection health + watchdog"
scp -q "$ROOT/patch-whatsmeow-bridge-connection-health.py" \
  "$ROOT/whatsapp-bridge-watchdog.sh" \
  "$ROOT/install-whatsapp-bridge-watchdog.sh" \
  "$HOST:${HERMES_SCRIPTS}/"
ssh "$HOST" "mkdir -p /root/.hermes/deploy"
scp -q "$ROOT/../deploy/whatsapp-bridge-watchdog.service" \
  "$ROOT/../deploy/whatsapp-bridge-watchdog.timer" \
  "$HOST:/root/.hermes/deploy/"
ssh "$HOST" "bash ${HERMES_SCRIPTS}/install-whatsapp-bridge-watchdog.sh"

echo "==> Timezone SGT + send_message JID fix"
scp -q "$ROOT/patch-hermes-systemd-timezone-sgt.sh" "$ROOT/patch-hermes-whatsapp-send.py" "$HOST:${HERMES_SCRIPTS}/"
ssh "$HOST" "python3 ${HERMES_SCRIPTS}/patch-hermes-whatsapp-send.py && python3 ${HERMES_SCRIPTS}/patch-hermes-whatsapp-send-outreach.py && bash ${HERMES_SCRIPTS}/patch-hermes-systemd-timezone-sgt.sh"
scp -q "$ROOT/outreach_tasks.py" "$HOST:${HERMES_SCRIPTS}/"
ssh "$HOST" "python3 ${HERMES_SCRIPTS}/patch-hermes-gateway-outreach-context.py"

echo "==> Action verifier hook (fast model + log proof)"
scp -q "$ROOT/install-hermes-action-verifier-hook.sh" \
  "$ROOT/patch-hermes-gateway-agent-end-verifier.py" \
  "$HOST:${HERMES_SCRIPTS}/"
ssh "$HOST" "bash ${HERMES_SCRIPTS}/install-hermes-action-verifier-hook.sh"

echo "==> Restart hermes-gateway"
ssh "$HOST" "systemctl restart hermes-gateway && sleep 4 && systemctl is-active hermes-gateway"
ssh "$HOST" "rg -n '^  default:' /root/.hermes/config.yaml | head -1"
ssh "$HOST" "curl -sS http://127.0.0.1:3000/health | python3 -c \"import sys,json; d=json.load(sys.stdin); assert d.get('sendReady'), d\""
echo "done"
