#!/usr/bin/env bash
# Deploy ai-receptionist to vignesh (or SSH_HOST).
set -euo pipefail

SSH_HOST="${SSH_HOST:-vignesh}"
REMOTE_DIR="${REMOTE_DIR:-/opt/ai-receptionist}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"

echo "→ rsync source to ${SSH_HOST}:${REMOTE_DIR}/src/"
rsync -az --delete \
  --exclude '*.db' \
  --exclude 'ai-receptionist' \
  --exclude 'ai-receptionist-linux' \
  --exclude '.env' \
  "${ROOT}/" "${SSH_HOST}:${REMOTE_DIR}/src/"

echo "→ copy config + prompts"
scp "${ROOT}/config.json" "${SSH_HOST}:${REMOTE_DIR}/config.json"
scp "${ROOT}/prompt.txt" "${ROOT}/prompt-personal.txt" "${SSH_HOST}:${REMOTE_DIR}/"

echo "→ build on server (CGO/sqlite)"
ssh "${SSH_HOST}" "cd ${REMOTE_DIR}/src && go build -o ${REMOTE_DIR}/ai-receptionist ."

echo "→ restart systemd"
ssh "${SSH_HOST}" "systemctl restart ai-receptionist && sleep 2 && systemctl is-active ai-receptionist" || true

echo "✓ deployed. Configure secrets on server, then restart:"
echo "  ssh ${SSH_HOST} 'nano /opt/ai-receptionist/.env /opt/ai-receptionist/config.json'"
echo "  ssh ${SSH_HOST} 'systemctl restart ai-receptionist'"
echo "  ssh ${SSH_HOST} 'journalctl -u ai-receptionist -f'"
