# WhatsApp AI Receptionist

Standalone Go app that connects to WhatsApp via [whatsmeow](https://github.com/tulir/whatsmeow), runs an AI receptionist via [Ollama Cloud](https://ollama.com), stores conversation memory in SQLite, qualifies leads, and alerts the business owner on WhatsApp.

## Version control

This project lives in the parent git repo at `whatsmeow/`. Commit from the repo root:

```bash
cd ..   # whatsmeow/
git add ai-receptionist/
git commit -m "your message"
```

Never commit `.env`, `*.db`, or the built `ai-receptionist` binary (see `.gitignore`).

## Requirements

- Go 1.25+
- SQLite (via `github.com/mattn/go-sqlite3`)
- Ollama Cloud API key ([ollama.com/settings/keys](https://ollama.com/settings/keys)) **or** OpenAI API key
- A phone number for WhatsApp Business or personal (Linked Devices)

## Quick start

```bash
cd ai-receptionist
cp .env.example .env
# cp config.example.json config.json — set owner_number (digits + country code, no +)
export AI_PROVIDER=ollama
export OLLAMA_API_KEY=your_key_from_ollama_com
# Or:
# export AI_PROVIDER=openai
# export OPENAI_API_KEY=...
# export OPENAI_BASE_URL=https://sg.api.openai.com   # default
# export OPENAI_MODEL=gpt-4.1-mini                   # default

go run .
```

On first run, scan the QR printed in the terminal with **WhatsApp → Linked devices**. Session is stored in `whatsmeow.db`. App data lives in `database.db`.

Send a **private text DM** to the linked number (not the owner number in config). The bot replies with AI, remembers the last 10 turns, collects lead fields, and notifies the owner once when qualified.

## Configuration

| File | Purpose |
|------|---------|
| `config.json` | Business name, owner phone, AI model |
| `prompt.txt` | Receptionist system prompt (editable without rebuild) |
| `.env` | `AI_PROVIDER`, `OLLAMA_API_KEY` or `OPENAI_API_KEY` (optional; can export in shell instead) |

## Julia (identity & instructions)

The bot runs as **Julia** with a three-layer prompt stack on every AI turn:

1. **Soul** — `knowledge/SOUL.md` → `identity_soul` (system prompt).
2. **Knowledge** — operational rules + `knowledge/KNOWLEDGE.md` → `client_instructions` (injected in the **user** turn as `EPICWARE KNOWLEDGE BASE`, with last 5 turns + current message + TASK).

Default layout matches the support-agent spec (see `test-cases-julia-eval.md`). Set `PROMPT_LAYOUT=stacked` for the older all-in-system layout.
3. **Per-contact facts** — `contact_facts` table (`conv_id`, `fact_key`, `fact_value`).

Optional runbook keys in `agent_notes`: `julia-cs`, `julia-sales`, `julia-booking` (placeholder content; wire into planner modes later).

Edit soul in git (`knowledge/SOUL.md`), redeploy (schema v7+ refreshes `identity_soul`), or in the Julia dashboard (**Settings → Instructions → Identity**), or in `database.db`:

```sql
UPDATE agent_notes SET content = '...' WHERE key = 'identity_soul';
```

Environment overrides:

- `WHATSMEOW_DB` — default `whatsmeow.db`
- `APP_DB` — default `database.db` (settings, dream proposals, conversations)
- `HTTP_ADDR` — enable dashboard/API (default off; use `127.0.0.1:8080`)
- `DASHBOARD_AUTH_TOKEN` — when set, requires token auth for the dashboard + all `/api/*` endpoints (send `Authorization: Bearer <token>` or `X-Admin-Token: <token>`)
- `DASHBOARD_BASIC_USER`, `DASHBOARD_BASIC_PASS` — when set, requires HTTP Basic auth for the dashboard + all `/api/*` endpoints
- `GRAPHITI_URL` — Graphiti sidecar base URL for memory ingest/recall and dream drafts (e.g. `http://127.0.0.1:8333`; see `graphiti/README.md`)
- `MEMORY_RECALL_IN_PROMPT` — set `1` to inject Graphiti recall into the WhatsApp prompt
- `CONFIG_PATH`, `PROMPT_PATH`, `INSTRUCTIONS_PATH` (default `knowledge/instructions.md`); soul source file `knowledge/SOUL.md` (embedded at build, synced to DB on migrate v7)
- `AI_PROVIDER` — `ollama` (default for local dev) or `openai`; production VPS uses dashboard **Settings → Providers** (`ai.provider=anthropic` recommended)
- `OPENAI_API_KEY`, `OPENAI_BASE_URL` (default `https://sg.api.openai.com`), `OPENAI_MODEL`
- **Model routing** (`internal/models/GetModel`): `intent_classify` uses a fast model (Haiku on Anthropic, config model on Ollama); `planner` / `collate` use the main dashboard model (Sonnet on Anthropic). Override with `INTENT_CLASSIFY_MODEL`, `PLANNER_MODEL`, `COLLATE_MODEL`.
- Optional latency budgets (seconds): `PLANNER_TIMEOUT_SEC`, `TOOLS_TIMEOUT_SEC`, `COLLATE_TIMEOUT_SEC`, `FAST_COMPLETE_TIMEOUT_SEC`, `OVERALL_AI_TIMEOUT_SEC`, `ACK_DELAY_SEC`, `AGENT_STATE_MAX_AGE_SEC`

## Julia eval (pre-deploy)

Before shipping prompt/soul/knowledge changes, run the support eval suite (`test-cases-julia-eval.md`):

```bash
cd ai-receptionist
go run ./cmd/juliaeval/
# or
./scripts/predeploy-eval.sh
```

`scripts/deploy.sh` runs this gate unless `SKIP_JULIA_EVAL=1`. Category 3 (escalation) failures block deploy.

## VPS deployment (systemd)

```bash
sudo mkdir -p /opt/ai-receptionist
sudo cp ai-receptionist config.json prompt.txt /opt/ai-receptionist/
# Build on server or copy binary:
cd ai-receptionist && go build -o ai-receptionist .
sudo cp ai-receptionist /opt/ai-receptionist/
```

`/etc/systemd/system/ai-receptionist.service`:

```ini
[Unit]
Description=WhatsApp AI Receptionist
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=ai-receptionist
WorkingDirectory=/opt/ai-receptionist
EnvironmentFile=/opt/ai-receptionist/.env
ExecStart=/opt/ai-receptionist/ai-receptionist
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now ai-receptionist
sudo journalctl -u ai-receptionist -f
```

**Backup `whatsmeow.db`** — losing it requires QR re-link. Backup `database.db` for lead history.

## Deploy to VPS

```bash
# From ai-receptionist/
export SSH_HOST=vignesh   # optional, default vignesh
./scripts/deploy.sh
```

Ensure `/opt/ai-receptionist/.env` on the server contains `AI_PROVIDER` plus the matching provider key (`OLLAMA_API_KEY` or `OPENAI_API_KEY`). Never commit `.env`.

## Troubleshooting: no reply

1. **Watch the terminal** when you send a message. You should see:
   ```txt
   inbound conv=self:1000000000000 chat=... text="hi"
   ```
   If you see `skip inbound` with `DEBUG_INBOUND=1`, the filter blocked the message.

2. **Ollama errors** — if you see `Ollama HTTP 401/403/429`, WhatsApp is working but AI is not. Check [ollama.com/settings/keys](https://ollama.com/settings/keys) and that `config.json` model is a cloud model (e.g. `gemma4:31b-cloud`).

3. **Startup check** — on launch you should see `Ollama Cloud OK`. If you see `WARNING: Ollama Cloud check failed`, fix `OLLAMA_API_KEY` before testing WhatsApp.

4. **`owner_number`** — use your full WhatsApp number (digits + country code, no `+`), not a placeholder. On link, the terminal prints `Linked account JID: ...`.

5. **Self-chat test** — use **Message yourself** in WhatsApp, not a random DM to your number.

### Avoiding repeated QR rescans

QR rescans happen when WhatsApp revokes the linked-device session or when `whatsmeow.db` is deleted.

- **Keep `/opt/ai-receptionist/whatsmeow.db`** across deploys (it contains the linked-device session).
- Only run `scripts/relink.sh` when you see a hard logout / deleted-device state; it now **backs up** `whatsmeow.db` before wiping it.

## Verification checklist

1. Cold start → QR pairing succeeds
2. Private DM → AI reply within ~5s (network dependent)
3. Restart process → bot still references prior messages (`database.db`)
4. Fill all required fields → owner gets **one** WhatsApp alert; contact `status` becomes `notified`
5. Message from `owner_number` or a group → no bot reply
6. Pricing/booking pressure → reply defers to team (safety post-check)

## Architecture

- `whatsmeow.db` — whatsmeow `sqlstore` (session keys)
- `database.db` — `contacts` + `messages`
- Per-chat mutex prevents parallel AI calls on double-texts
- Go merges `lead_updates` and injects `missing_fields` / `current_lead_data` into the system prompt

## Personal mode (reply as you)

To auto-reply to **all private DMs** in your voice (not the agency receptionist funnel):

1. Edit `config.json`:
   ```json
   {
     "mode": "personal",
     "reply_to_groups": false,
     "business_name": "Your Name",
     "business_description": "Describe how you actually text: tone, topics you handle, what you defer (pricing, meetings, etc.). Paste 3–5 example replies you would send."
   }
   ```
2. Point at the personal prompt:
   ```bash
   export PROMPT_PATH=prompt-personal.txt
   ```
3. Run as usual (`export OLLAMA_API_KEY=...` then `go run .`).

**What changes in personal mode**

| Setting | Behavior |
|---------|----------|
| `mode: personal` | Plain-text replies (no lead JSON); no owner “new lead” alerts |
| `reply_to_groups: true` | Also replies in group chats (sender prefixed in context) |
| `reply_to_self_chat: true` | Replies in **Message yourself** (good for testing; bot ignores its own sends) |
| `owner_number` | Skipped in normal DMs; still works in self-chat when `reply_to_self_chat` is on |

### Test in "Message yourself"

1. Set `owner_number` to your WhatsApp number (same as the linked account).
2. Keep `"reply_to_self_chat": true` (default).
3. Run the bot, open WhatsApp → **Message yourself**, send `hi`.
4. You should get an AI reply in that thread (not in other chats you message from the same phone).

**Limits**

- Only **text** (and captions) trigger replies — bare images/audio are skipped unless you extend the code.
- It cannot perfectly clone you without a strong `business_description` + examples in `prompt-personal.txt`.
- WhatsApp may flag heavy automation; use on a business/secondary line if possible.

## Out of scope (v1)

Dashboard, CRM, HTTP API, calendar booking.
