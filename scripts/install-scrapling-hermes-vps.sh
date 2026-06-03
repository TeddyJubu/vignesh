#!/usr/bin/env bash
# Install Scrapling (https://github.com/D4Vinci/Scrapling) into Hermes venv + MCP config.
set -euo pipefail

HOST="${STELLA_SSH_HOST:-vignesh}"
HERMES_VENV="${HERMES_VENV:-/usr/local/lib/hermes-agent/venv}"
PY="${HERMES_VENV}/bin/python"
SCRAPLING="${HERMES_VENV}/bin/scrapling"
UV="${UV:-/usr/local/bin/uv}"

echo "==> Install scrapling[ai,fetchers] into Hermes venv on ${HOST}"
ssh "$HOST" "${UV} pip install --python ${PY} 'scrapling[ai,fetchers]'"

echo "==> Install Playwright/patchright browsers (may take a few minutes)"
ssh "$HOST" "${SCRAPLING} install"

echo "==> Patch Hermes config (MCP server)"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
scp -q "$ROOT/patch-hermes-config-scrapling-mcp.py" "$HOST:/root/.hermes/scripts/"
ssh "$HOST" "python3 /root/.hermes/scripts/patch-hermes-config-scrapling-mcp.py"

echo "==> SOUL: prefer Scrapling for web scrape tasks"
scp -q "$ROOT/patch-hermes-soul-scrapling.py" "$HOST:/root/.hermes/scripts/"
ssh "$HOST" "python3 /root/.hermes/scripts/patch-hermes-soul-scrapling.py"

echo "==> Restart hermes-gateway"
ssh "$HOST" "systemctl restart hermes-gateway && sleep 5 && systemctl is-active hermes-gateway"

echo "==> Verify MCP tools registered (from gateway log)"
ssh "$HOST" "rg -n \"scrapling|mcp_scrapling\" /root/.hermes/logs/agent.log 2>/dev/null | tail -8 || journalctl -u hermes-gateway -n 80 --no-pager | rg -i scrapling | tail -8"

echo "done"
