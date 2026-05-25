# WhatsApp AI Receptionist

Standalone Go app that connects to WhatsApp via [whatsmeow](https://github.com/tulir/whatsmeow), runs an AI receptionist (OpenRouter), stores conversation memory in SQLite, qualifies leads, and alerts the business owner on WhatsApp.

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
- OpenRouter API key
- A phone number for WhatsApp Business or personal (Linked Devices)

## Quick start

```bash
cd ai-receptionist
cp .env.example .env
# Edit config.json — set owner_number (digits + country code, no +)
export OPENROUTER_API_KEY=sk-or-v1-...

go run .
```

On first run, scan the QR printed in the terminal with **WhatsApp → Linked devices**. Session is stored in `whatsmeow.db`. App data lives in `database.db`.

Send a **private text DM** to the linked number (not the owner number in config). The bot replies with AI, remembers the last 10 turns, collects lead fields, and notifies the owner once when qualified.

## Configuration

| File | Purpose |
|------|---------|
| `config.json` | Business name, owner phone, AI model |
| `prompt.txt` | Receptionist system prompt (editable without rebuild) |
| `.env` | `OPENROUTER_API_KEY` (optional path overrides) |

Environment overrides:

- `WHATSMEOW_DB` — default `whatsmeow.db`
- `APP_DB` — default `database.db`
- `CONFIG_PATH`, `PROMPT_PATH`

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
     "business_name": "Teddy",
     "business_description": "Describe how you actually text: tone, topics you handle, what you defer (pricing, meetings, etc.). Paste 3–5 example replies you would send."
   }
   ```
2. Point at the personal prompt:
   ```bash
   export PROMPT_PATH=prompt-personal.txt
   ```
3. Run as usual (`export OPENROUTER_API_KEY=...` then `go run .`).

**What changes in personal mode**

| Setting | Behavior |
|---------|----------|
| `mode: personal` | Plain-text replies (no lead JSON); no owner “new lead” alerts |
| `reply_to_groups: true` | Also replies in group chats (sender prefixed in context) |
| `owner_number` | Still skipped — messages *from* your own number are ignored (avoids loops) |

**Limits**

- Only **text** (and captions) trigger replies — bare images/audio are skipped unless you extend the code.
- It cannot perfectly clone you without a strong `business_description` + examples in `prompt-personal.txt`.
- WhatsApp may flag heavy automation; use on a business/secondary line if possible.

## Out of scope (v1)

Dashboard, CRM, HTTP API, calendar booking.
