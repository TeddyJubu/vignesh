#!/usr/bin/env bash
# Install Hermes gateway hook + patch script on the VPS (or local ~/.hermes).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
HERMES_HOME="${HERMES_HOME:-$HOME/.hermes}"
PATCH_SRC="${PATCH_SRC:-$ROOT/scripts/patch-hermes-whatsapp-send.py}"
HOOK_SRC="${HOOK_SRC:-$ROOT/hermes-hooks/reapply-whatsapp-send-patch}"

mkdir -p "$HERMES_HOME/scripts" "$HERMES_HOME/hooks" "$HERMES_HOME/logs"
install -m 0755 "$PATCH_SRC" "$HERMES_HOME/scripts/patch-hermes-whatsapp-send.py"
rm -rf "$HERMES_HOME/hooks/reapply-whatsapp-send-patch"
cp -R "$HOOK_SRC" "$HERMES_HOME/hooks/reapply-whatsapp-send-patch"
chmod 0644 "$HERMES_HOME/hooks/reapply-whatsapp-send-patch/HOOK.yaml"
chmod 0644 "$HERMES_HOME/hooks/reapply-whatsapp-send-patch/handler.py"

echo "Installed:"
echo "  $HERMES_HOME/scripts/patch-hermes-whatsapp-send.py"
echo "  $HERMES_HOME/hooks/reapply-whatsapp-send-patch/"
echo ""
echo "Hook runs on gateway:startup (after hermes update restarts the gateway)."
echo "Log: $HERMES_HOME/logs/patch-whatsapp-send.log"
echo ""
echo "Restart gateway to load hook: hermes gateway restart"
