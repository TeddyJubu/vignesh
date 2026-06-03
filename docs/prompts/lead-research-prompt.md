# Lead research prompts (Stella / WhatsApp)

Use from **Teddy** or **Vignesh** (admin DM). Send **`/reset`** if the session is stuck on an old task.

Stella should: **numbered PLAN → execute step-by-step → deliver a dossier file** (not a wall of text in chat). See `WHATSAPP REPORT DELIVERY` in SOUL.

**Messy input:** You do not need the template below. One line like `vasan salon maps weak reviews epicware dossier` is enough — see `docs/prompts/messy-prompt-tests.md`.

---

## Primary prompt — deep dive (one lead)

Copy-paste and fill the bracketed fields:

```
Lead research task (Epicware sales).

Business: [Business name]
Location: [City / area, default Singapore]
Phone (if known): [+65 … or +880 …]
Website / Google Maps link (if known): [URL]
Industry (from form or your guess): [e.g. salon, F&B, clinic]
Ad source (if known): [e.g. Meta — Review & Maps Growth]

Rules:
1. Show a numbered PLAN (≤8 steps) in this DM first — then execute; do not stop after step 1.
2. Research using **Scrapling MCP** (`mcp_scrapling_get` / `fetch` / `stealthy_fetch`) on target URLs; avoid raw Google HTML scraping. Cover Maps presence, reviews, website/SEO basics, social links, competitors, decision-maker hints.
3. Score Epicware fit 1–10 using our tiers (Foundation → Enterprise) — Singapore local businesses only unless I say otherwise.
4. Flag disqualifiers silently (wrong vertical, too small, outside SG, competitor) per whatsapp-sales skill — if disqualified, say OPERATOR ONLY at top, do not draft outreach to the lead.
5. Output: write full dossier to ~/.hermes/reports/lead-[slug]-[YYYYMMDD].md and send me the path + 5-bullet executive summary in DM. Attach with MEDIA: if the file is long.
6. End with DONE and these sections in the file:
   - Snapshot (name, phone, location, links)
   - Maps & reviews (count, rating, velocity, gaps)
   - Digital presence (site, social, GBP completeness)
   - Pain hypotheses (visibility vs trust vs ops — 3 bullets)
   - Epicware tier recommendation + why (no pricing unless I asked)
   - Suggested opening angle for WhatsApp Q1 (1 short message, warm, no pricing)
   - Risks / unknowns (what to verify on a call)
7. Delegate heavy web research with delegate_task if helpful.
```

**Example (filled):**

```
Lead research task (Epicware sales).

Business: Vasan's Salon
Location: Singapore
Phone: +65 8366 9443
Website / Google Maps link: [paste Maps URL if you have it]
Industry: Salon
Ad source: SG — Review & Maps Growth

[…same Rules 1–7 as above…]
```

---

## Quick prompt —已有 form lead (`sales_leads.json`)

```
Check ~/.hermes/sales_leads.json for phone [6583669443]. Research that active lead's business for Epicware: Maps/reviews/website, fit score 1–10, tier recommendation, suggested Q1 opener. PLAN first, write report file, DM summary + DONE. Do not WhatsApp the lead.
```

---

## Batch prompt — 3 leads compare

```
Research these 3 Singapore prospects for Epicware (PLAN first, then each lead as steps 2–4, compare in step 5):

1. [Name] — [business] — [phone or Maps URL]
2. [Name] — [business] — [phone or Maps URL]
3. [Name] — [business] — [phone or Maps URL]

One markdown file: ~/.hermes/reports/lead-batch-[YYYYMMDD].md with table (fit score, tier, top pain, Q1 angle) + one paragraph per lead. DM: top pick to call first + DONE.
```

---

## Pass / fail (manual)

| Pass | Fail |
|------|------|
| PLAN before tools | Jumps straight to long DM essay |
| Report file + short DM summary | Only chat, no file |
| Fit score + tier + Q1 draft | Quotes pricing unprompted |
| Says OPERATOR ONLY if disqualified | Messages the lead without ask |
| Uses web/delegate | Mentions wabot / :7777 (violates SOUL system instruction) |

---

## Optional follow-ups

- `Add a “call prep” section: 5 questions Vignesh should ask on a 20-min discovery call.`
- `Turn the Q1 opener into a send_message draft only — do not send until I say send.`
- `Update sales_leads.json last_message and next_action from this dossier.`
