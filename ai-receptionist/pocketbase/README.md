# PocketBase (agent persistence)

Production instance (Hostinger / Traefik):

- **Admin UI (public HTTPS):** https://wabot.srv943071.hstgr.cloud/pb-admin/_/
- **API health:** `GET https://wabot.srv943071.hstgr.cloud/pb-admin/api/health`

The Hostinger subdomain `pocketbase-mqk9.srv943071.hstgr.cloud` cannot get a new Let’s Encrypt cert (shared `hstgr.cloud` rate limit), so admin is served under the existing `wabot` certificate at `/pb-admin/`.
- **Health:** `GET {POCKETBASE_URL}/api/health`

On the `vignesh` host, PocketBase runs in Docker (`pocketbase-mqk9-pocketbase-1`) behind Traefik (`pocketbase-mqk9.srv943071.hstgr.cloud`). Compose at `/docker/pocketbase-mqk9/docker-compose.yml` pins **`127.0.0.1:8090:8090`** so the bot uses:

```bash
POCKETBASE_URL=http://127.0.0.1:8090
```

**CLI on the server** must use the real data dir: `pocketbase superuser ... --dir=/pb_data` (not the default `/usr/local/bin/pb_data`).

## Environment (ai-receptionist)

Set in local `.env` only — **never commit tokens or passwords**.

```bash
# On the VPS (recommended): http://127.0.0.1:8090
# Public admin UI: https://wabot.srv943071.hstgr.cloud/pb-admin/_/
POCKETBASE_URL=http://127.0.0.1:8090
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
