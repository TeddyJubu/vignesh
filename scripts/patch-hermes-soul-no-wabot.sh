#!/usr/bin/env bash
# Append hard ban on wabot recovery commands to ~/.hermes/SOUL.md
set -euo pipefail

SOUL="/root/.hermes/SOUL.md"
MARKER="### Legacy wabot — FORBIDDEN"

if grep -qF "$MARKER" "$SOUL" 2>/dev/null; then
  echo "ok: SOUL already has wabot ban"
  exit 0
fi

cat >>"$SOUL" <<'EOF'

### Legacy wabot — FORBIDDEN (Jun 2026)

- **Production WhatsApp:** whatsmeow-bridge `:3000` + hermes-gateway only.
- **Never** run, suggest, or debug: `wabot`, `/opt/wabot`, `wabot.env`, port `7777`, `wabot-agent`, or `ps aux | grep wabot`.
- **Never** recreate `/opt/wabot` or start `/usr/local/bin/wabot` — archived under `/opt/archive/whatsapp-legacy-20260603/`.
- If `systemctl` output mentions wabot in a description, **ignore it** — do not try to “fix” legacy services.
- Group tasks: `send_message` + `@g.us` JID only (see WhatsApp groups section).

EOF

echo "ok: SOUL wabot ban appended"
