#!/usr/bin/env bash
# Patch bridge health, build, install watchdog timer on current host (VPS).
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HERMES_HOME="$(cd "$SCRIPT_DIR/.." && pwd)"
HERMES_SCRIPTS="${HERMES_SCRIPTS:-$SCRIPT_DIR}"
DEPLOY_DIR="${DEPLOY_DIR:-$HERMES_HOME/deploy}"

_install_if_different() {
  local src="$1" dst="$2" mode="$3"
  if [ "$(readlink -f "$src")" != "$(readlink -f "$dst")" ]; then
    install -m "$mode" "$src" "$dst"
  fi
}
_install_if_different "$SCRIPT_DIR/whatsapp-bridge-watchdog.sh" "$HERMES_SCRIPTS/whatsapp-bridge-watchdog.sh" 0755
_install_if_different "$SCRIPT_DIR/patch-whatsmeow-bridge-connection-health.py" "$HERMES_SCRIPTS/patch-whatsmeow-bridge-connection-health.py" 0755

python3 "$HERMES_SCRIPTS/patch-whatsmeow-bridge-connection-health.py"
cd /opt/whatsmeow-bridge
go build -o whatsmeow-bridge .
systemctl restart whatsmeow-bridge
sleep 5

for unit in whatsapp-bridge-watchdog.service whatsapp-bridge-watchdog.timer; do
  src="$DEPLOY_DIR/$unit"
  dst="/etc/systemd/system/$unit"
  if [ "$(readlink -f "$src" 2>/dev/null || echo "$src")" != "$(readlink -f "$dst" 2>/dev/null || echo "$dst")" ]; then
    install -m 0644 "$src" "$dst"
  fi
done
systemctl daemon-reload
systemctl enable --now whatsapp-bridge-watchdog.timer

echo "ok: bridge patched, watchdog timer active"
systemctl list-timers whatsapp-bridge-watchdog.timer --no-pager
curl -sS http://127.0.0.1:3000/health
echo
