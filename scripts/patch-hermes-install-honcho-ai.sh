#!/usr/bin/env bash
# Ensure honcho-ai is importable from the Hermes gateway venv (post-upgrade).
set -euo pipefail
PY="/usr/local/lib/hermes-agent/venv/bin/python"
if [ ! -x "$PY" ]; then
  echo "missing $PY" >&2
  exit 1
fi
if "$PY" -c "from honcho import Honcho" 2>/dev/null; then
  echo "ok: honcho-ai already available"
  exit 0
fi
uv pip install --python "$PY" honcho-ai
"$PY" -c "from honcho import Honcho; print('ok: honcho-ai installed')"
