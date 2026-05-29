#!/usr/bin/env bash
# Run Julia eval suite before deploy. Exits non-zero on critical failures.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

echo "→ Julia eval (live AI — may take several minutes)"
go run ./cmd/juliaeval/

echo "✓ eval gate passed"
