# PocketBase (agent persistence)

Production instance (Hostinger / Traefik):

- **URL:** `https://pocketbase-mqk9.srv943071.hstgr.cloud`
- **Admin UI:** open the URL in a browser (PocketBase redirects to `/_/`).
- **Health:** `GET {POCKETBASE_URL}/api/health`

On the `vignesh` host, PocketBase runs in Docker (`pocketbase-mqk9-pocketbase-1`, internal port 8090) behind Traefik with host rule `pocketbase-mqk9.srv943071.hstgr.cloud`. Local direct port (if needed on the server): `32769` → 8090.

## Environment (ai-receptionist)

Set in local `.env` only — **never commit tokens or passwords**.

```bash
POCKETBASE_URL=https://pocketbase-mqk9.srv943071.hstgr.cloud
POCKETBASE_TOKEN=                    # preferred: long-lived admin API token
# POCKETBASE_ADMIN_EMAIL=            # dev: refresh token at startup
# POCKETBASE_ADMIN_PASSWORD=
ECHO_INTENT=1                        # Day 1: echo classifier instead of planner
INTENT_CLASSIFY_MODEL=               # optional override
```

Auth order:

1. `POCKETBASE_TOKEN` (Bearer on every request)
2. Else `POCKETBASE_ADMIN_EMAIL` + `POCKETBASE_ADMIN_PASSWORD` → `POST /api/collections/_superusers/auth-with-password` (cached for process lifetime)

PocketBase errors are logged to `errors.log` and do **not** block WhatsApp replies.

## Collections (create once in Admin UI)

| Collection | Fields (minimum) |
|------------|------------------|
| `agent_sessions` | `wa_number` (text, unique), `last_summary`, `last_intent`, `last_updated_at` |
| `agent_jobs` | `wa_number`, `task_type`, `payload` (json), `status`, `result` (json), `error`, `created`, `updated` |
| `lead_contacts` | `wa_number`, lead fields, `qualified`, `notified_at` |
| `support_log` | event fields (Day 1 stub) |
| `booking_log` | event fields (Day 1 stub) |

`wa_number` matches WhatsApp `convID` from inbound handling (phone digits, `self:…`, or group JID).

Day 1 writes: `agent_sessions` upsert + `agent_jobs` insert with `status=classified` after each intent classification.
