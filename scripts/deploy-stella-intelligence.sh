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

echo "==> Restart hermes-gateway"
ssh "$HOST" "systemctl restart hermes-gateway && sleep 4 && systemctl is-active hermes-gateway"
ssh "$HOST" "rg -n '^  default:' /root/.hermes/config.yaml | head -1"
echo "done"
