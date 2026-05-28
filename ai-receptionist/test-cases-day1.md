# Day 1 — Manual test Q&A

Use these with **`ECHO_INTENT=1`** on the VPS. Send each line as a **new WhatsApp DM** to the linked bot number (**+65 8028 6424**). Use a number that is **not** `owner_number` in `config.json` (or use **Message yourself** on the linked device if `reply_to_self_chat` is true).

Expected reply shape:

```text
intent=<name> conf=<0.00-1.00> summary=<short text>
```

---

## Intent classification (echo replies)

| # | You send (question) | Expected `intent=` | Notes |
|---|------------------------|-------------------|--------|
| 1 | What are your pricing plans? | `support` | Pricing / plans / cost |
| 2 | I want to book a meeting | `sales_qualify` | Sales / discovery call |
| 3 | What am I doing tomorrow? | `calendar_check` | User’s own schedule |
| 4 | Research Meta ad trends for F&B | `research_request` | Research, not scraping |
| 5 | Scrape 20 dental clinics in Singapore | `lead_scrape` | List / scrape contacts |
| 6 | Hi there | `general` | Small talk (optional) |
| 7 | Can you add me to the group admins? | `group_manage` | Group ops (optional) |
| 8 | Book a call with the client John for Tuesday | `outbound_book` | Coordinating for someone else (optional) |
| 9 | Generate a logo for my cafe | `image_generate` | Image/creative (optional) |

---

## Context (SQLite last turns)

| # | Steps | Pass if |
|---|--------|---------|
| 10 | Send: `What are your pricing plans?` → wait for echo reply | Reply received |
| 11 | Send again: `What are your pricing plans?` | Reply still `intent=support` and bot did not crash; classifier had prior turn in SQLite (same conv) |

---

## PocketBase persistence

| # | Steps | Pass if |
|---|--------|---------|
| 12 | After test #1–5, open [PocketBase Admin](https://wabot.srv943071.hstgr.cloud/pb-admin/_/) → `agent_sessions` | Row for your `wa_number` / conv id with `last_intent` and `last_summary` |
| 13 | Check `agent_jobs` | Recent rows with `task_type=intent_classify`, `status=classified` |

---

## Resilience

| # | Steps | Pass if |
|---|--------|---------|
| 14 | On VPS: `docker stop pocketbase-mqk9-pocketbase-1` → send any test message → `docker start pocketbase-mqk9-pocketbase-1` | Bot still sends echo reply; errors only in `errors.log` / journal |
| 15 | Unset `GRAPHITI_URL` or stop Graphiti (if used) → send a message | Bot still replies (Graphiti optional) |

---

## VPS regression (no WhatsApp)

On the server:

```bash
cd /opt/ai-receptionist/src
export $(grep -E '^[A-Z]' /opt/ai-receptionist/.env | xargs)
export CONFIG_PATH=/opt/ai-receptionist/config.json APP_DB=/opt/ai-receptionist/database.db
go run ./cmd/day1check/
```

Pass if all five core lines show **`OK`** (not `MISMATCH`).

---

## Turn off echo (after Day 1 sign-off)

Set on VPS `.env`: `ECHO_INTENT=0` (or remove), then `systemctl restart ai-receptionist`. Bot returns to normal receptionist / planner flow.
