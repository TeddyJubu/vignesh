#!/usr/bin/env bash
# Install action verifier hook + library on Hermes host.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
HERMES_HOME="${HERMES_HOME:-/root/.hermes}"
HOOK_SRC="${HOOK_SRC:-$ROOT/hermes-hooks/action-verifier}"
LIB_SRC="${LIB_SRC:-$SCRIPT_DIR/stella_action_verifier}"

mkdir -p "$HERMES_HOME/hooks" "$HERMES_HOME/logs" "$HERMES_HOME/stella_action_verifier"

if [ -d "$HOOK_SRC" ]; then
  rm -rf "$HERMES_HOME/hooks/action-verifier"
  cp -R "$HOOK_SRC" "$HERMES_HOME/hooks/action-verifier"
fi
if [ ! -f "$HERMES_HOME/hooks/action-verifier/handler.py" ]; then
  echo "error: hook missing at $HERMES_HOME/hooks/action-verifier" >&2
  exit 1
fi
chmod 0644 "$HERMES_HOME/hooks/action-verifier/HOOK.yaml"
chmod 0644 "$HERMES_HOME/hooks/action-verifier/handler.py"

if [ -d "$LIB_SRC" ]; then
  if command -v rsync >/dev/null 2>&1; then
    rsync -a --delete "$LIB_SRC/" "$HERMES_HOME/stella_action_verifier/"
  else
    rm -rf "$HERMES_HOME/stella_action_verifier"
    cp -R "$LIB_SRC" "$HERMES_HOME/stella_action_verifier"
  fi
fi
if [ -f "$SCRIPT_DIR/outreach_tasks.py" ]; then
  install -m 0644 "$SCRIPT_DIR/outreach_tasks.py" "$HERMES_HOME/scripts/outreach_tasks.py"
fi
find "$HERMES_HOME/stella_action_verifier" -name '__pycache__' -type d -exec rm -rf {} + 2>/dev/null || true

if [ ! -f "$HERMES_HOME/action-verifier.json" ]; then
  install -m 0644 "$ROOT/deploy/action-verifier.example.json" "$HERMES_HOME/action-verifier.json"
fi

PATCH="$SCRIPT_DIR/patch-hermes-gateway-agent-end-verifier.py"
if [ -f "$PATCH" ]; then
  python3 "$PATCH"
fi

echo "ok: action-verifier installed under $HERMES_HOME"
echo "config: $HERMES_HOME/action-verifier.json"
echo "log: $HERMES_HOME/logs/action-verifier.log"
echo "restart: systemctl restart hermes-gateway"
