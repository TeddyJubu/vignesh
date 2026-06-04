#!/usr/bin/env bash
# Default Hermes + bridge processes to Asia/Singapore (matches config.yaml timezone).
set -euo pipefail

TZ_NAME="${HERMES_TZ:-Asia/Singapore}"
DROPIN_DIR="/etc/systemd/system/hermes-gateway.service.d"
DROPIN="$DROPIN_DIR/timezone-sgt.conf"

mkdir -p "$DROPIN_DIR"
cat >"$DROPIN" <<EOF
[Service]
Environment=TZ=${TZ_NAME}
Environment=HERMES_TIMEZONE=${TZ_NAME}
EOF

# Bridge reads wall clock for logs; align with SGT for operators.
BRIDGE_DROPIN="/etc/systemd/system/whatsmeow-bridge.service.d"
mkdir -p "$BRIDGE_DROPIN"
cat >"$BRIDGE_DROPIN/timezone-sgt.conf" <<EOF
[Service]
Environment=TZ=${TZ_NAME}
EOF

systemctl daemon-reload
systemctl restart hermes-gateway whatsmeow-bridge || true
echo "ok: TZ=${TZ_NAME} on hermes-gateway + whatsmeow-bridge"
timedatectl show -p Timezone 2>/dev/null || true
