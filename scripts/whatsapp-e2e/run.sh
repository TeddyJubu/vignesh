#!/usr/bin/env bash
# Run WhatsApp E2E on vignesh. Requires: ssh vignesh, bridge /inject patched.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT"

echo "==> Deploy inject patch to vignesh (if needed)"
ssh vignesh 'test -f /tmp/patch-whatsmeow-bridge-inject.py || true'
scp -q "$ROOT/../patch-whatsmeow-bridge-inject.py" vignesh:/tmp/ 2>/dev/null || \
  scp -q "$(dirname "$ROOT")/patch-whatsmeow-bridge-inject.py" vignesh:/tmp/
ssh vignesh 'python3 /tmp/patch-whatsmeow-bridge-inject.py 2>/dev/null; cd /opt/whatsmeow-bridge && go build -o whatsmeow-bridge . && systemctl restart whatsmeow-bridge'
sleep 2
ssh vignesh 'curl -sS -o /dev/null -w "%{http_code}" -X POST http://127.0.0.1:3000/inject -H "Content-Type: application/json" -d "{\"body\":\"probe\",\"senderId\":\"8801521207499@s.whatsapp.net\",\"chatId\":\"8801521207499@s.whatsapp.net\"}"' | grep -q 200 || {
  echo "inject endpoint not healthy" >&2
  exit 2
}

echo "==> Health"
ssh vignesh 'systemctl is-active hermes-gateway whatsmeow-bridge; curl -sS http://127.0.0.1:3000/health'

case "${1:-}" in
  --groups)
    shift
    exec python3 "$ROOT/run_group_tests.py" --deploy "$@"
    ;;
  --groups-only)
    shift
    exec python3 "$ROOT/run_group_tests.py" --skip-deploy "$@"
    ;;
  --messy)
    shift
    exec python3 "$ROOT/run_group_tests.py" --deploy --messy "$@"
    ;;
  --messy-only)
    shift
    exec python3 "$ROOT/run_group_tests.py" --skip-deploy --messy "$@"
    ;;
  --full)       exec python3 "$ROOT/run_tests.py" --full --skip-inject-check ;;
  --set2)       exec python3 "$ROOT/run_tests.py" --set2 --skip-inject-check ;;
  --all)        exec python3 "$ROOT/run_tests.py" --all-suites --skip-inject-check ;;
  --smoke)      exec python3 "$ROOT/run_tests.py" --smoke --skip-inject-check ;;
  "")           exec python3 "$ROOT/run_tests.py" --skip-inject-check ;;
  *)            exec python3 "$ROOT/run_tests.py" "$@" --skip-inject-check ;;
esac
