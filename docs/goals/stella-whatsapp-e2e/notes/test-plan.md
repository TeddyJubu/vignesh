# Stella WhatsApp E2E test plan

Reference checklist for the goal oracle. Full matrix from prep conversation.

## Smoke (5 min)

1. Third party / Gaya: `ping` → reply <30s
2. Vignesh: `Message Gaya +6590016046: smoke test` → lands on Gaya, not Teddy
3. Teddy: `/reset` then `Who are you?` → Stella, co-owner
4. Vignesh: `Say UNIQUE-TEST-7742` → exactly one reply
5. Stranger: `/help` → no full admin

## Categories

- **Policy**: open DMs; owners only Teddy + Vignesh (`allow_admin_from`)
- **Identity**: Stella, never impersonate Vignesh
- **Third-party send**: `whatsapp:+E.164` or JID; home-channel guard on bare `whatsapp`
- **LID**: `87192181436622` (Teddy), `56676572987400` (Vignesh) resolve or allowlisted
- **Duplicates**: single gateway stack, one reply per prompt
- **Admin**: `/reset`, `/update` owners only
- **Limits**: iteration cap, secret refusal, long input
- **Post-update**: patch hook re-applies on `gateway:startup`

## VPS targets

- SSH host: `vignesh` (`31.97.188.246`)
- Bridge: `http://127.0.0.1:3000` (`/health`, `/messages`, `/send`)
- Hermes: `/root/.hermes/`, gateway logs `gateway.log`
- Patch: `/root/.hermes/scripts/patch-hermes-whatsapp-send.py`

## Pass criteria

All smoke + sampled cases from each category pass; failures logged with sender JID, time, gateway + bridge log excerpt; fix→retest until green.

## Automated runner

```bash
./scripts/whatsapp-e2e/run.sh          # smoke (S1–S6)
./scripts/whatsapp-e2e/run.sh --full   # + extended (E1–E3)
./scripts/whatsapp-e2e/run.sh --set2   # advanced set 2 (N00–N70) — see test-plan-set2.md
./scripts/whatsapp-e2e/run.sh --all    # everything
```

Requires `ssh vignesh` and bridge `POST /inject` (localhost-only).
