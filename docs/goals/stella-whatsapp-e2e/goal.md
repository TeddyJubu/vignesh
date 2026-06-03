# Stella WhatsApp E2E — test agent, fix loop, green oracle

## Objective

Build a temporary WhatsApp test simulator (synthetic senders: Teddy, Vignesh, third parties), run the full regression matrix against the live Hermes + whatsmeow-bridge stack on VPS `vignesh`, fix failures, and retest until all smoke and category checks pass.

## Original Request

Create an agent that temporarily acts as WhatsApp, runs all limit/regression tests, fixes anything wrong, and retests until everything looks perfect — tracked as a GoalBuddy goal.

## Intake Summary

- Input shape: `specific` (clear outcome + existing test plan)
- Audience: Teddy + Vignesh (Stella operators)
- Authority: `requested` (VPS SSH `vignesh`, Hermes config already tuned)
- Proof type: `test`
- Completion proof: Automated test run (or scripted matrix) reports all required cases PASS; manual spot-check only if automation cannot cover a case (documented in receipt)
- Goal oracle: `scripts/whatsapp-e2e/run.sh` (or equivalent) exits 0 on vignesh against live gateway; gateway.log shows inbound→response for each synthetic sender; no duplicate replies on UNIQUE-TEST probe; third-party send does not land on Teddy home channel
- Likely misfire: Writing test docs only, or testing bridge `/send` without driving gateway inbound path; declaring done after one smoke pass while LID/admin/third-party cases still fail
- Blind spots considered: LID vs phone JID, open DM vs owner-only admin, home-channel guard after patch, post-`hermes update` patch hook, rate limits / API cost during full matrix
- Existing plan facts: See `notes/test-plan.md`; VPS paths and policies from prior session (open `dm_policy`, owners Teddy+Vignesh, bridge LID fallback, patch hook on `gateway:startup`)

## Goal Oracle

The oracle for this goal is:

**Exit code 0 from the repo’s WhatsApp E2E runner against `vignesh`, with a JSON/markdown report showing PASS for every required row in `notes/test-plan.md` (smoke + all categories), after at least one fix→retest cycle if any case failed initially.**

The PM must keep comparing task receipts to this oracle. Planning or a partial smoke pass is not enough.

## Goal Kind

`specific`

## Current Tranche

1. Scout maps how to inject synthetic inbound messages (bridge API, gateway test hooks, or minimal `/inject` if missing).
2. Worker implements test simulator + runner.
3. Worker runs matrix on VPS, fixes Hermes/bridge/config/scripts, restarts services as needed, reruns until green.
4. Final Judge audit maps receipts to oracle.

## Non-Negotiable Constraints

- Do not re-enable archived legacy WhatsApp stacks under `/opt/archive/whatsapp-legacy-20260603/` (duplicate-reply risk).
- Owners remain Teddy + Vignesh only; DMs stay open for everyone.
- Preserve `reapply-whatsapp-send-patch` hook and `patch-hermes-whatsapp-send.py`.
- SSH changes on VPS only; keep runnable artifacts in this repo under `scripts/whatsapp-e2e/`.
- No force-push; no committing secrets from `.env`.

## Stop Rule

Stop only when T999 records `full_outcome_complete: true` and the oracle runner passes.

## Canonical Board

`docs/goals/stella-whatsapp-e2e/state.yaml`

## Run Command

```text
/goal Follow docs/goals/stella-whatsapp-e2e/goal.md.
```
