#!/usr/bin/env bash
# Run simulated WhatsApp handler tests (no real WA/AI). Use on VPS after deploy.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "${ROOT}"
echo "→ simulated WhatsApp handler tests"
go test ./internal/receptionist/ -run 'TestSimulated|TestCompleteWithPlanner' -count=1
