# T001 — WhatsApp inbound injection map

## Architecture

```text
E2E runner (local/CI)
  → POST http://127.0.0.1:3000/inject  (new, localhost-only)
  → whatsmeow-bridge messageQueue
  → Hermes gateway GET /messages (long-poll)
  → _build_message_event → agent → POST /send
```

No `/inject` today; only real WhatsApp events append to `messageQueue` (`handleEvent`).

## Message shape (bridge → Hermes)

Required fields (from `main.go` handleEvent + `whatsapp.py` `_should_process_message`):

```json
{
  "messageId": "e2e-<uuid>",
  "chatId": "8801521207499@s.whatsapp.net",
  "senderId": "8801521207499@s.whatsapp.net",
  "senderName": "Teddy",
  "chatName": "Teddy",
  "isGroup": false,
  "body": "ping",
  "hasMedia": false,
  "mediaType": "",
  "mediaUrls": [],
  "mentionedIds": [],
  "quotedMessageId": "",
  "quotedParticipant": "",
  "quotedRemoteJid": "",
  "hasQuotedMessage": false,
  "botIds": [],
  "timestamp": 1717420000
}
```

`dm_policy: open` — any `senderId` accepted. Admin checks use `allow_admin_from` only.

## Personas

| Role | senderId / chatId |
|------|-------------------|
| Vignesh (phone) | `6590013157@s.whatsapp.net` |
| Vignesh (LID) | `56676572987400:31@lid` / chat `56676572987400@lid` |
| Teddy (phone) | `8801521207499@s.whatsapp.net` |
| Teddy (LID) | `87192181436622:34@lid` / chat `87192181436622@lid` |
| Gaya | `6590016046@s.whatsapp.net` |
| Stranger | `15550009999@s.whatsapp.net` |

## Pass/fail signals

| Check | Command / pattern |
|-------|-------------------|
| Gateway inbound | `grep "inbound message.*chat=<jid>" gateway.log` after inject |
| Gateway reply | `grep "response ready.*chat=<jid>" gateway.log` within 90s |
| Outbound target | `journalctl -u whatsmeow-bridge \| grep "Sent to <jid>"` |
| Single reply | count `Sending response.*<jid>` for one inject id / unique marker |
| Admin deny | response text lacks owner-only success OR no `inbound.*15550009999` for blocked |

Debounce: wait **4s** after inject before asserting inbound (text batch ~2–3s).

## Implementation slice (T004)

1. Add `POST /inject` to `/opt/whatsmeow-bridge/main.go` (127.0.0.1 only).
2. `scripts/whatsapp-e2e/` runner via `ssh vignesh` curl inject + log tail.
3. Smoke 5 cases first; `--full` for extended matrix.

## Baseline (T002)

- `hermes-gateway` + `whatsmeow-bridge`: **active**
- `/health`: connected
- Recent live traffic: Teddy phone JID replying (12:40 UTC)
