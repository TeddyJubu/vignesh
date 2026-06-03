# Group workflow E2E (GW*)

Automated multi-step tests for Stella: group create/maintenance, meeting prefs, booking, email — with a **verifier loop** (score 1–10, must be ≥6, up to 5 retries).

## Run

```bash
# Deploy bridge /groups API + SOUL plan, then run all GW workflows
./scripts/whatsapp-e2e/run.sh --groups

# Single workflow, no redeploy
./scripts/whatsapp-e2e/run.sh --groups-only --workflow GW1
```

Reports: `scripts/whatsapp-e2e/reports/group-run-*.json`

## Workflows

| ID | Focus |
|----|--------|
| GW1 | Bridge `POST /groups/create`, topic, group message `E2E-GW1-LIVE` |
| GW2 | Group + in-group meeting time poll (no duplicate DMs) |
| GW3 | Full chain: group → preferences (simulated replies) → calendar → email → confirm |

## Verifier

- **Heuristic rubric** (`verifier.py`): plan, `@g.us`, participants, booking/email keywords, no wabot.
- **Retry**: injects `[VERIFIER RETRY n]` with feedback until score ≥ 6 or max retries.
- Optional `--llm`: blends OpenRouter score from VPS (needs `OPENROUTER_API_KEY` in `~/.hermes/.env`).

## Infra

- `scripts/patch-whatsmeow-bridge-groups.py` — `/groups/create`, `/groups/participants/add`, `/groups/topic` (localhost only)
- `scripts/patch-hermes-soul-multistep-plan.py` — PLAN → step-by-step execution

## Personas

Uses `personas.json` (teddy_phone, vignesh_phone, gaya). GW3 injects simulated preference replies into the group once `@g.us` appears in logs.
