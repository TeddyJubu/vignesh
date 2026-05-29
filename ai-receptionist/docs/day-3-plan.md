# Day 3 — Power Agents (Epicware 3-Day Sprint)

**Source:** `Epicware_3Day_Sprint_Plan.docx` (Developer Handoff)  
**Repo reality:** Single Go binary `ai-receptionist` (not Node monorepo). PocketBase + SQLite replace Supabase + Redis from the brief, but Day 3 **intent and tests are the same**.

**Rule:** Do not mark Day 3 complete until every test below passes.

---

## Sprint context (all 3 days)

| Day | Theme | End state |
|-----|-------|-----------|
| **Day 1** | Foundation — listener, orchestrator, memory, intent routing | Any WA message classified and persisted |
| **Day 2** | Core agents — Support, Calendar, Sales SDR, email, reminders | Support Q&A, live calendar, qualify + book leads |
| **Day 3** | **Power agents** — Lead Scraper, Research, Outbound Booker, Group Manager | Async jobs, CSV email, third-party booking, group ops |

---

## Day 3 goal

> Lead Scraper (5-pass parallel) · Research Agent · Outbound Booker · Group Manager

Everything runs **async** via the job queue (Day 1 foundation). Lead scraper is the hardest — budget **4+ hours**.

---

## Already in repo (stubs / partial)

| Sprint item | Go equivalent | Status |
|-------------|---------------|--------|
| Job queue | `async_jobs` table + `internal/ops/async_worker.go` | ✅ exists |
| Research agent | `research_marketing` tool + `handleResearchMarketing` | ⚠️ placeholder brief only |
| Lead scrape | `scrape_leads` tool + `handleScrapeLeads` | ⚠️ placeholder CSV only |
| Email CSV | `email_csv_to_owner` tool + Composio Gmail | ⚠️ partial |
| Outbound book | `outbound_book` intent + `group_commands.go` booking coord | ⚠️ partial |
| Group manage | `group_manage` intent + `handleGroupAdmin` | ⚠️ partial, `group_admin` off in config |
| Intent routing | `internal/intent/classifier.go` | ✅ all Day 3 intents defined |

---

## Morning — Lead Scraper (5-pass pipeline)

### Tasks

- [ ] **Job dispatcher:** `intent === lead_scrape` → insert `async_jobs` row (`task_type=lead_scrape`, payload `{query, count, vertical}`, `status=pending`)
- [ ] **Pass 1 — broad search:** web search → `[{name, company, url}]`
- [ ] **Pass 2 — enrich:** per lead, parallel lookups (site, email, phone, LinkedIn, social) — use concurrency limit (~10)
- [ ] **Pass 3 — classify:** ICP fit score 1–10, `icp_match`
- [ ] **Pass 4 — pitch angle:** for `fit_score >= 7` only
- [ ] **Pass 5 — QA:** dedupe, drop non-ICP, clean array (stronger model)
- [ ] **CSV export:** name, company, email, fit_score, pitch_angle columns
- [ ] **Deliver:** email CSV to Vignesh (Composio Gmail), WhatsApp confirmation, write rows to `lead_contacts` / PocketBase
- [ ] **Wire planner:** route `lead_scrape` intent → queue job (not inline reply)

### Critical note (from brief)

Pass 2 **must** run field lookups in parallel per lead (not sequential `await` loops). Batch with a concurrency cap to avoid Anthropic rate limits.

---

## Afternoon — Research + Outbound Booker + Group Manager

### Research Agent

- [ ] Dispatcher: `research_request` → `async_jobs` (`task_type=research_marketing`)
- [ ] Worker: multi-step web search + synthesis (Anthropic web search tool or Composio)
- [ ] Output: Executive Summary, Key Findings, Sources (markdown)
- [ ] Deliver: WhatsApp DM to Vignesh (split if >4000 chars), update job `done`

### Outbound Booker

- [ ] Dispatcher: `outbound_book` → job with `{contact_name, wa_number, meeting_purpose}`
- [ ] Fetch Vignesh free slots (Composio calendar)
- [ ] WhatsApp contact with 3 slot options
- [ ] **Reply matching:** store `{wa_number, job_id}` in session so contact's reply routes to booking worker, not generic classifier
- [ ] On confirm: `createEvent`, log `booking_log`, confirm both parties
- [ ] No-reply after 24h: follow-up + notify Vignesh

### Group Manager

- [ ] `group_manage` intent → parse create / add / remove / rename / announce
- [ ] whatsmeow APIs: `createGroup`, `addToGroup`, `removeFromGroup`, announce
- [ ] Enable `capabilities.group_admin` in production config when ready
- [ ] Contact must exist before adding to group

---

## Day 3 pass/fail tests (from sprint doc)

| # | Input | Expected |
|---|-------|----------|
| 1 | "Scrape 10 F&B consultants in Singapore with email and pitch angle" | Job queued → CSV email within ~15 min, 10 rows with all columns |
| 2 | Check DB | 10 `lead_contacts` rows linked to job |
| 3 | "Research what Meta ad angles are working for dental clinics in Singapore" | Structured report via WhatsApp within ~5 min |
| 4 | "Book a meeting with John Tan, +6598765432, about Epicware partnership" | John gets WA with 3 slots |
| 5 | Reply as John confirming slot | Calendar event + confirmations both sides |
| 6 | "Create a WhatsApp group called Epicware VIP and add +6591234567" | Group created with correct name + member |
| 7 | 50-lead scrape | Completes without rate-limit blowup |
| 8 | Kill + restart service | Cron/worker resumes; no jobs stuck in `running` |

---

## End-of-sprint full system test (Day 1–3)

Run after Day 3 tests pass:

| Area | Input | Expected |
|------|-------|----------|
| Support | "What is the Authority plan?" | Correct plan details <5s |
| Escalate | "I want to cancel" | Vignesh DM flag <5s |
| Calendar | "Am I free Monday 3pm?" | Real availability |
| Sales | 3 qualifying messages | Qualified → slots → book on confirm |
| Scrape | "Scrape 10 salons in Singapore with email" | CSV email, 10 rows |
| Research | "Research TikTok ads for beauty salons Singapore" | Report via WA |
| Outbound | "Book a call with [name] [number] re: partnership" | Contact messaged, book on reply |
| Group | "Create group Epicware Test and add [number]" | Group exists |
| Memory | 5 msgs → restart → 6th references msg 1 | Context retained |
| Queue | Trigger scrape → check `async_jobs` | pending → running → done |

---

## Architecture note (doc vs this repo)

The sprint brief describes **Go whatsmeow listener + Node.js orchestrator + Redis + Supabase**. This repo consolidated into:

- **Go only:** whatsmeow + receptionist handler + planner/tools
- **SQLite:** messages, contacts, `async_jobs`
- **PocketBase:** sessions/jobs (optional mirror)
- **Composio:** Google Calendar + Gmail (instead of service account + nodemailer)
- **Anthropic:** via dashboard provider config

Day 3 **features and acceptance tests** still apply — implement them in Go against existing tables/tools.

---

## Suggested build order (Day 3 only)

1. Harden async worker + job status transitions (pending → running → done/failed)
2. Lead scraper pipeline (biggest chunk)
3. Research worker (replace placeholder)
4. Outbound book reply-routing in Redis/SQLite session
5. Group manager + enable capability flag
6. Run full sprint test table

---

## Reference

Original doc: `/Users/teddyburtonburger/Downloads/Epicware_3Day_Sprint_Plan.docx`  
Day 1 checklist: `test-cases-day1.md`  
Manual QA: `test-cases.md`
