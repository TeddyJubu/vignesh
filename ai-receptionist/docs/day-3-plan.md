# Day 3 — Personal Mode Quality (Phase 3)

**Status:** Not started  
**Source:** `plan.md` Phase 3 + merged main as of `d204ac4`

## Goal

Ship the missing personal-mode features so Vignesh can trust “reply as me” on real threads:

| # | Feature | Status |
|---|---------|--------|
| 3.1 | `style-examples.txt` | Done |
| 3.2 | `never_reply[]` per-chat opt-out | **Todo** |
| 3.3 | `draft_only` mode (drafts to owner only) | **Todo** |
| 3.4 | Voice note → text transcription | **Todo** |

**Done when:** Test personal mode on 5 real threads; >80% of drafts are sendable with light edits.

---

## Build order (fastest value first)

### 1. Config surface
Add to `config.go`, `config.example.json`, `.env.example`:
- `never_reply: []` — JIDs/phones to ignore
- `draft_only: false`
- `TRANSCRIBE_PROVIDER`, `TRANSCRIBE_MODEL` (optional; can reuse OpenAI key)

### 2. `never_reply[]` filter
- Extend `internal/whatsapp/inbound.go` (same pattern as `blocked_numbers`)
- Wire through handler if needed
- **Acceptance:** listed chats get no reply; allow/block/group unchanged

### 3. `draft_only` routing
- In `handler.go` send path: if `draft_only=true`, do **not** send to customer
- Send to owner self-chat with prefix: `[DRAFT for +<recipient>] <message>`
- **Acceptance:** customer chat receives nothing; owner sees draft with recipient context

### 4. Voice note transcription
- `internal/ai/transcribe.go` — small transcription helper
- `internal/whatsapp/media.go` — download audio from WhatsApp
- Replace `[audio]` placeholder in inbound preprocessing
- **Acceptance:** voice notes become text in the normal pipeline; failures degrade gracefully (no crash)

---

## Test commands (in order)

```bash
cd ai-receptionist
go test ./internal/whatsapp/ -count=1
go test ./internal/receptionist/ -run 'TestSimulated|TestFinalizeCustomerReply' -count=1
./scripts/smoke-sim.sh
go test ./... -count=1
```

---

## What's already done (Days 1–2)

- **Day 1:** Intent classifier, PocketBase, echo mode, session turns
- **Day 2:** Planner → parallel tools → collate, provider hardening, empathy UX, Composio calendar/Gmail, Julia eval gate
- **Production:** VPS on `main`, owner = Vignesh (`6590013157`), Composio live

## Not Day 3 (defer)

- Phase 4 ops (`doctor` CLI, health endpoint, logrotate)
- OTP dashboard login (incomplete — remove or finish separately)
- Owner escalation leaking into customer chat (separate bugfix)
