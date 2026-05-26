#!/usr/bin/env bash
# Wipe WhatsApp session and restart so a fresh QR is printed.
set -euo pipefail

SSH_HOST="${SSH_HOST:-vignesh}"
REMOTE_DIR="${REMOTE_DIR:-/opt/ai-receptionist}"

echo "→ stop, backup, delete whatsmeow.db, start"
ssh "${SSH_HOST}" "set -euo pipefail; systemctl stop ai-receptionist; \
  if [ -f ${REMOTE_DIR}/whatsmeow.db ]; then cp -f ${REMOTE_DIR}/whatsmeow.db ${REMOTE_DIR}/whatsmeow.db.bak.$(date +%Y%m%d-%H%M%S); fi; \
  rm -f ${REMOTE_DIR}/whatsmeow.db; \
  systemctl start ai-receptionist; sleep 3"

echo "→ latest logs (look for QR block):"
ssh "${SSH_HOST}" "journalctl -u ai-receptionist -n 60 --no-pager | grep -v '^--'"

echo ""
echo "Follow live: ssh ${SSH_HOST} 'journalctl -u ai-receptionist -f'"
