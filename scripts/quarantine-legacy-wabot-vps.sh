#!/usr/bin/env bash
# Quarantine legacy wabot on srv943071 so Stella cannot rediscover/restart it.
set -euo pipefail

ARCHIVE="/opt/archive/whatsapp-legacy-20260603"
STAMP="$(date -u +%Y%m%dT%H%M%SZ)"

mkdir -p "$ARCHIVE"

if [[ -d /opt/wabot ]]; then
  dest="${ARCHIVE}/wabot-recreated-${STAMP}"
  mv /opt/wabot "$dest"
  echo "moved /opt/wabot -> $dest"
fi

for unit in wabot-agent.service wabot.service cloudflared-quick.service; do
  if systemctl list-unit-files "$unit" &>/dev/null; then
    systemctl stop "$unit" 2>/dev/null || true
    systemctl disable "$unit" 2>/dev/null || true
    systemctl mask "$unit" 2>/dev/null || true
    echo "masked $unit"
  fi
done

if [[ -x /usr/local/bin/wabot ]]; then
  mkdir -p "${ARCHIVE}/bin"
  if [[ ! -e "${ARCHIVE}/bin/wabot" ]]; then
    mv /usr/local/bin/wabot "${ARCHIVE}/bin/wabot"
    echo "moved /usr/local/bin/wabot -> ${ARCHIVE}/bin/wabot"
  fi
fi

# Neutralize misleading unit description (agent reads systemctl output).
if [[ -f /etc/systemd/system/cloudflared-quick.service ]]; then
  mkdir -p /etc/systemd/system/cloudflared-quick.service.d
  cat >/etc/systemd/system/cloudflared-quick.service.d/legacy-override.conf <<'EOF'
[Unit]
Description=Legacy cloudflared quick tunnel (disabled; WhatsApp uses whatsmeow-bridge :3000 only)
EOF
  systemctl daemon-reload 2>/dev/null || true
fi

echo "done: legacy wabot quarantined"
