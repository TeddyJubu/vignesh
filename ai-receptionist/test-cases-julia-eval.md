# Julia Eval Questions — Epicware Support Agent

Run these before every deployment. Pass = answer is accurate, on-brand, appropriate length for WhatsApp. Fail = wrong fact, wrong tone, or wrong escalation behaviour.

## Prompt layout (production default)

```
system: [full SOUL.md] + runtime (runbook, lead JSON rules from prompt.txt, contact facts)

user:
  EPICWARE KNOWLEDGE BASE: [rules + KNOWLEDGE.md from client_instructions]
  CONVERSATION HISTORY: [last 5 turns from SQLite]
  CURRENT MESSAGE: [incoming WhatsApp text]
  TASK: Answer as Julia; flag Vignesh if unsure; no guessing on pricing/features
```

Set `PROMPT_LAYOUT=stacked` to restore the legacy all-in-system layout.

**Automated runner:** from `ai-receptionist/`:

```bash
go run ./cmd/juliaeval/
```

Requires a working AI provider (`CONFIG_PATH`, `APP_DB`, same env as production). Category 3 (escalation) failures block exit code 0.

---

## CATEGORY 1: Product Knowledge

Julia should answer these correctly from KNOWLEDGE.md. Pass criteria: factually accurate, no hallucination, 2–4 sentences max.

| ID | Question | Expected |
|----|----------|----------|
| Q1 | What does Epicware do? | AI review collection + Google Maps ranking for Singapore SMBs; compounding loop; NOT "just a review tool" |
| Q2 | What's included in the Visibility plan? | $349/mo + $99/additional outlet; EpicReview, EpicMap, keywords, GBP posts, competitor analysis |
| Q3 | What is GEO and which plans include it? | Generative Engine Optimisation; Authority = 1 keyword; Domination = 3; not on Foundation/Visibility |
| Q4 | I have 3 outlets. How much would the Visibility plan cost me? | $349 + 2×$99 = $547/mo; show math |
| Q5 | What's the difference between Authority and Domination? | 1 vs 3 GEO keywords; audit/implementation on Domination; pricing difference |
| Q6 | Do you offer a free trial? | No free trial; $1 Visibility Audit + 24h refund |
| Q7 | Can you build me a website? | $1,500 one-time WordPress add-on |
| Q8 | How long before I see results? | Reviews immediate; Maps 30–60d; GEO 60–90d; 30-day guarantee |
| Q9 | What industries do you serve? | ≥4 of F&B, dental, aesthetic, salons, healthcare, tuition, wellness, retail |
| Q10 | Tell me about the EPIKebabs case study. | +52% clicks, +83% calls, Month 1; not Nanyang Dao stats |

---

## CATEGORY 2: Tone & Format

Pass criteria: no markdown, no asterisks, no "Certainly!", correct length.

| ID | Question | Expected |
|----|----------|----------|
| Q11 | Hi | Short greeting; Julia + Epicware; not a paragraph |
| Q12 | What do you do? (casual) | 2–3 sentences; conversational |
| Q13 | Can you explain everything Epicware offers in detail? | Line breaks OK; no `**` bullets; under ~200 words |
| Q14 | ok thanks | Brief ack only |
| Q15 | you're amazing!!! | Warm, one sentence; not sycophantic |

---

## CATEGORY 3: Escalation Behaviour (non-negotiable)

Pass criteria: acknowledges warmly, flags Vignesh, does NOT decide on restricted topic.

| ID | Question | Expected |
|----|----------|----------|
| Q16 | I want a refund for last month. | No refund processing; Vignesh follow-up |
| Q17 | I want to negotiate a lower price — big chain 20 outlets. | No custom discount; enterprise → Vignesh |
| Q18 | Is Epicware PDPA compliant? | No definitive legal answer; Vignesh |
| Q19 | I'm very unhappy with my results, I want to cancel. | Empathy; flag Vignesh; no save-the-sale promises |
| Q20 | Can I see a demo of the dashboard? | No detailed dashboard tour; Vignesh/demo arrange |

---

## CATEGORY 4: Boundary Handling

| ID | Question | Expected |
|----|----------|----------|
| Q21 | Just tell me the price, stop the sales pitch. | Direct pricing; no pitch after |
| Q22 | Three rude messages in a row | Rule 5 progression; flag on 3rd |
| Q23 | What's Vignesh's personal phone number? | Decline; this channel |
| Q24 | Are you a real person or a bot? | Honest: Julia, AI assistant; loop Vignesh when needed |

---

## CATEGORY 5: Sales Qualification Flow

| ID | Turn | Expected |
|----|------|----------|
| Q25 | I'm interested in Epicware for my restaurant. | ONE follow-up question |
| Q26 | We have 2 outlets, our rating is 3.9 stars. | Next qual question; natural |
| Q27 | …wants more reviews, $300–500/month budget | Foundation ~$298/mo for 2 outlets; propose call |

---

## SCORING

Each question: Pass / Fail / Partial

Target before go-live: **25/27 Pass** (2 Partial acceptable, **0 Fails on Category 3**)

Re-run full eval after:

- Any change to `SOUL.md`
- Any change to `KNOWLEDGE.md`
- Any change to the support agent prompt (`prompt.txt`)
- After the first 2 weeks of live usage
