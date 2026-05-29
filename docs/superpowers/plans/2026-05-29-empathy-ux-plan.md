# Empathy & UX Hardening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Elevate Julia's empathy and communication style by suppressing noisy logs/acks on greetings, preventing amnesia for names/emails via conversation history scans, dynamically preventing "last question" lies, adding email to required fields, and allowing post-qualification appointment updates.

**Architecture:** 
1. **Suppression (Go)**: Match common greetings and short strings in `handler.go` to bypass acknowledgments.
2. **Context Fallback (Go/SQLite)**: Scan historical turns for name/email backfilling before missing fields are generated in the prompt.
3. **Prompt injection**: Inject a constraint directive dynamically in `buildSystemPrompt` restricting "last question" phrase usage based on missing field count.
4. **Post-qualification (Go)**: Intercept messages in `"notified"` status to execute the calendar booking tool directly and provide cohesive confirmations without re-triggering qualification.

**Tech Stack:** Go 1.22+, SQLite3, Anthropic Provider API

---

### Task 1: Suppress acknowledgments on greetings and short messages

**Files:**
- Modify: `ai-receptionist/internal/receptionist/handler.go`
- Test: `ai-receptionist/internal/receptionist/handler_sim_test.go`

- [ ] **Step 1: Write a failing test for greeting ack suppression**
Add a test in `handler_sim_test.go` that simulates sending `"hi"` and ensures no acknowledgment ("Got it — checking now.") is dispatched even if the AI takes longer than `ackDelay`.

```go
func TestHandleMessage_GreetingDoesNotTriggerAck(t *testing.T) {
	// Adjust delay so we trigger acks on delay
	ackDelay = 2 * time.Millisecond
	defer func() { ackDelay = 10 * time.Millisecond }()

	store, clean := setupTestDB(t)
	defer clean()

	var sentText []string
	whatsapp.SetTestHooks(func(ctx context.Context, chat types.JID, text string) error {
		sentText = append(sentText, text)
		return nil
	}, func(ctx context.Context, chat types.JID, typing bool) {})

	aiClient := &scriptProvider{
		responses: []string{`{"reply":"Hello! How can I help you today?","lead_updates":{},"qualified":false}`},
	}

	h := New(testConfig(), store, aiClient, aiClient, aiClient, aiClient, nil, nil, "System prompt", "", "")

	// Simulate "hi" turn with artificial delay in completion
	aiClient.delay = 5 * time.Millisecond
	evt := makeFakeMessage("6591111111", "hi")
	h.HandleMessage(context.Background(), evt)

	for _, txt := range sentText {
		if strings.Contains(txt, "checking now") || strings.Contains(txt, "second") {
			t.Errorf("Ack sent on greeting: %q", txt)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**
Run: `go test -v -run TestHandleMessage_GreetingDoesNotTriggerAck ./internal/receptionist/...`
Expected: FAIL (ack is sent because greeting matching is not yet implemented)

- [ ] **Step 3: Modify `shouldSendAck` and ack text in `handler.go`**
Update `handler.go` around line 688 and 706 to match greetings (case-insensitive "hi", "hello", "hey", "yo", "morning") or messages `< 7 characters` and return `false`. Also change the ack text to `"Just a second, let me check that for you... 🔍"`.

```go
// In internal/receptionist/handler.go:
func isGreetingOrShort(text string) bool {
	clean := strings.ToLower(strings.TrimSpace(text))
	if len(clean) < 7 {
		return true
	}
	greetings := []string{"hi", "hello", "hey", "yo", "morning", "afternoon", "evening"}
	for _, g := range greetings {
		if clean == g {
			return true
		}
	}
	return false
}

func (h *Handler) maybeSendAck(ctx context.Context, chat types.JID, convID string, userText string) {
	defer func() { recover() }()
	if isGreetingOrShort(userText) {
		return
	}
	select {
	case <-ctx.Done():
		return
	case <-time.After(ackDelay):
	}
	if ctx.Err() != nil {
		return
	}
	if !h.shouldSendAck(convID) {
		return
	}
	if err := whatsapp.SendText(ctx, h.wa, chat, "Just a second, let me check that for you... 🔍"); err == nil {
		h.markAckSent(convID)
	}
}
```
Update the goroutine launch in `HandleMessage`:
```go
	go h.maybeSendAck(ackCtx, v.Info.Chat, convID, text)
```

- [ ] **Step 4: Run test to verify it passes**
Run: `go test -v -run TestHandleMessage_GreetingDoesNotTriggerAck ./internal/receptionist/...`
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add ai-receptionist/internal/receptionist/handler.go
git commit -m "feat: suppress background acknowledgments on greetings and short messages"
```

---

### Task 2: Add `email` as a required field in qualification funnel

**Files:**
- Modify: `ai-receptionist/internal/lead/fields.go`
- Modify: `ai-receptionist/prompt.txt`
- Test: `ai-receptionist/internal/lead/fields_test.go` (if exists) or inline checks.

- [ ] **Step 1: Write a failing test or verify fields list**
Ensure we modify `internal/lead/fields.go` to include `"email"` in `Required`.

```go
// In ai-receptionist/internal/lead/fields.go:
var Required = []string{
	"name",
	"email",
	"business_type",
	"service_needed",
	"budget",
	"timeline",
	"current_website",
}
```

- [ ] **Step 2: Update the system prompt rules in `prompt.txt`**
Add strict guidelines for email capture during booking in `prompt.txt`:
```markdown
## Lead fields to collect (in natural order)
- name
- email
- business_type (what they do)
- service_needed (what they want built)
- budget (rough range is fine)
- timeline (when they want to start/finish)
- current_website (URL or "none")
- best_time (optional — when to call)
```
Update Rules for replies:
```markdown
- For booking: Ask for preferred day, time, and timezone first. Once provided, capture their name, email, and business name to send the calendar invitation and lock it in. Never promise a calendar invite until a real slot is agreed.
```
Update Output format:
```markdown
- Set qualified=true only when name, email, business_type, service_needed, budget, timeline, and current_website are all known.
```

- [ ] **Step 3: Run full tests to check compilation**
Run: `go test ./...` in `ai-receptionist`
Expected: PASS

- [ ] **Step 4: Commit**
```bash
git add ai-receptionist/internal/lead/fields.go ai-receptionist/prompt.txt
git commit -m "feat: add email to required lead qualification fields"
```

---

### Task 3: Implement History-Informed Memory Recovery (Goldfish Amnesia Protection)

**Files:**
- Modify: `ai-receptionist/internal/receptionist/handler.go`
- Test: `ai-receptionist/internal/receptionist/handler_sim_test.go`

- [ ] **Step 1: Write failing test for name recovery from history**
Add a test in `handler_sim_test.go` where we store a recent user message saying `"My name is Teddy"` but the SQL lead data is currently empty. The system prompt generated should NOT list `"name"` in `missing_fields` because it is backfilled dynamically from the database message log.

```go
func TestBuildSystemPrompt_BackfillsNameFromHistory(t *testing.T) {
	store, clean := setupTestDB(t)
	defer clean()

	convID := "6591234567"
	// Insert previous user message explicitly stating name
	_ = store.InsertMessage(convID, "user", "hi, I am Teddy")

	h := New(testConfig(), store, nil, nil, nil, nil, nil, nil, "System prompt", "", "")

	leadData := map[string]string{} // Name is currently missing in lead state
	prompt, err := h.buildSystemPrompt(convID, leadData, whatsapp.InboundContext{}, "en", "receptionist", "book a call", false)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(prompt, `"name"`) && strings.Contains(prompt, "missing_fields") {
		t.Errorf("System prompt still lists name as missing after being stated in history")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**
Run: `go test -v -run TestBuildSystemPrompt_BackfillsNameFromHistory ./internal/receptionist/...`
Expected: FAIL

- [ ] **Step 3: Implement dynamic backfilling in `buildSystemPrompt`**
Before calling `lead.Missing(leadData)` in `buildSystemPrompt`, check the recent messages of the chat in SQL. Use regexes to detect email addresses (`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b`) and a heuristic for name retrieval (scanning for `"name is [Name]"` or `"i am [Name]"` or `"teddy, marketing agency"`). If found, inject them directly back into `leadData`.

```go
// In internal/receptionist/handler.go, inside buildSystemPrompt (or as a helper):
import "regexp"

var emailRegex = regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b`)
var nameRegex1 = regexp.MustCompile(`(?i)(?:my name is|i am|i'm)\s+([A-Za-z]+)`)

func backfillLeadFromHistory(history []store.Message, data map[string]string) map[string]string {
	out := make(map[string]string, len(data))
	for k, v := range data {
		out[k] = v
	}
	if out["name"] != "" && out["email"] != "" {
		return out
	}
	for i := len(history) - 1; i >= 0; i-- {
		msg := history[i]
		if msg.Role != "user" {
			continue
		}
		txt := msg.Message
		if out["email"] == "" {
			if matches := emailRegex.FindString(txt); matches != "" {
				out["email"] = matches
			}
		}
		if out["name"] == "" {
			if m := nameRegex1.FindStringSubmatch(txt); len(m) > 1 {
				out["name"] = m[1]
			} else if strings.Contains(txt, ",") {
				// E.g. "Teddy, Marketing Agency" -> first part is likely name
				parts := strings.Split(txt, ",")
				if len(parts) > 0 {
					candidate := strings.TrimSpace(parts[0])
					if len(candidate) > 0 && len(candidate) < 20 && !strings.Contains(candidate, " ") {
						out["name"] = candidate
					}
				}
			}
		}
	}
	return out
}
```
Call this right at the start of `buildSystemPrompt` or `HandleMessage`. Ensure it backfills the state passed to both the prompt and lead updates.

- [ ] **Step 4: Run test to verify it passes**
Run: `go test -v -run TestBuildSystemPrompt_BackfillsNameFromHistory ./internal/receptionist/...`
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add ai-receptionist/internal/receptionist/handler.go
git commit -m "feat: backfill missing lead name and email dynamically from message history"
```

---

### Task 4: Dynamic "Last Question" Constraints Injection

**Files:**
- Modify: `ai-receptionist/internal/receptionist/handler.go`
- Modify: `ai-receptionist/prompt.txt`
- Test: `ai-receptionist/internal/receptionist/handler_sim_test.go`

- [ ] **Step 1: Write a test ensuring directives are injected**
Verify that when multiple fields are missing, the system prompt contains a strict `CRITICAL CONSTRAINT` against saying "last question".

```go
func TestBuildSystemPrompt_InjectsLastQuestionConstraint(t *testing.T) {
	store, clean := setupTestDB(t)
	defer clean()

	h := New(testConfig(), store, nil, nil, nil, nil, nil, nil, "System prompt", "", "")
	leadData := map[string]string{} // All fields missing

	prompt, err := h.buildSystemPrompt("6591234567", leadData, whatsapp.InboundContext{}, "en", "receptionist", "hi", false)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(prompt, "CRITICAL CONSTRAINT") || !strings.Contains(prompt, "forbidden from saying") {
		t.Errorf("Constraint against lying about last question not injected")
	}
}
```

- [ ] **Step 2: Run test to verify failure**
Run: `go test -v -run TestBuildSystemPrompt_InjectsLastQuestionConstraint ./internal/receptionist/...`
Expected: FAIL

- [ ] **Step 3: Implement constraint injection in `buildSystemPrompt`**
In `buildSystemPrompt`, calculate `len(lead.Missing(leadData))`. If this is `> 1`, append the exact dynamic constraint text.

```go
// In internal/receptionist/handler.go, inside buildSystemPrompt:
	if h.cfg.LeadTrackingEnabled() {
		missing := lead.Missing(leadData)
		leadJSON, _ := json.Marshal(leadData)
		b.WriteString("\n\n## Runtime context\n")
		b.WriteString("missing_fields: ")
		missJSON, _ := json.Marshal(missing)
		b.Write(missJSON)
		b.WriteString("\ncurrent_lead_data: ")
		b.Write(leadJSON)
		b.WriteString("\n")

		if len(missing) > 1 {
			b.WriteString("\n## Crucial Phrasing Constraints\n")
			b.WriteString("CRITICAL CONSTRAINT: You have multiple missing fields to collect. You are strictly forbidden from saying \"last question\", \"one last question\" or \"last quick one\". Instead, use phrases like \"Just a couple of quick details...\" or \"Next...\". Only refer to a \"final question\" when exactly one required field remains.\n")
		}
	}
```

- [ ] **Step 4: Run test to verify passes**
Run: `go test -v -run TestBuildSystemPrompt_InjectsLastQuestionConstraint ./internal/receptionist/...`
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add ai-receptionist/internal/receptionist/handler.go
git commit -m "feat: programmatically inject last question constraint based on field counts"
```

---

### Task 5: Post-Qualification Dynamic Updates

**Files:**
- Modify: `ai-receptionist/internal/receptionist/handler.go`
- Test: `ai-receptionist/internal/receptionist/handler_sim_test.go`

- [ ] **Step 1: Write failing test for post-qualification slot updates**
Create a test in `handler_sim_test.go` where the user is already `"notified"`. When they send `"Tuesday 3pm Singapore"`, the receptionist should direct-run the calendar tool instead of launching the generic planner, and confirm the slot warmly.

```go
func TestHandleMessage_PostQualificationUpdate(t *testing.T) {
	store, clean := setupTestDB(t)
	defer clean()

	convID := "6591111111"
	// Save as qualified lead with notified status
	leadData := map[string]string{"name": "Teddy", "email": "teddy@example.com"}
	leadJSON, _ := json.Marshal(leadData)
	_ = store.UpdateContactWithScore(convID, "Teddy", string(leadJSON), "notified", "high")

	var sentText []string
	whatsapp.SetTestHooks(func(ctx context.Context, chat types.JID, text string) error {
		sentText = append(sentText, text)
		return nil
	}, func(ctx context.Context, chat types.JID, typing bool) {})

	aiClient := &scriptProvider{
		responses: []string{`{"reply":"Got it, Tuesday 3pm SGT works! Vignesh will confirm.","lead_updates":{},"qualified":true}`},
	}

	h := New(testConfig(), store, aiClient, aiClient, aiClient, aiClient, nil, nil, "System prompt", "", "")

	evt := makeFakeMessage(convID, "Tuesday 3pm Singapore")
	h.HandleMessage(context.Background(), evt)

	found := false
	for _, txt := range sentText {
		if strings.Contains(txt, "Tuesday 3pm") {
			found = true
		}
	}
	if !found {
		t.Errorf("Did not find expected warm confirmation: %v", sentText)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**
Run: `go test -v -run TestHandleMessage_PostQualificationUpdate ./internal/receptionist/...`
Expected: FAIL

- [ ] **Step 3: Modify `completeWithPlanner` in `handler.go`**
If `contact.Status == "notified"`, intercept execution. If they provide scheduling text, execute the calendar booking tool directly, updating the reservation, then return the confirmation reply.

```go
// In completeWithPlanner in handler.go:
	if mode == modeBooking && hasPendingPlan == false {
		contact, _ := h.store.GetContact(convID)
		if contact != nil && contact.Status == "notified" {
			// Fast path for qualified leads updating slots.
			// Direct run of booking time parsing or appointment tool
			// Let's execute tool directly or pass to collation to keep robust.
			// We can bypass planner and go straight to calendar book tool:
			plan := &agent.Plan{
				Goal: "Book/update appointment slot for qualified lead",
				Agents: []agent.AgentTask{
					{
						Name:           "BookSlot",
						Tool:           "book_appointment",
						Input:          userText,
						ExpectedOutput: "appointment confirmation",
					},
				},
				FinalResponseMode: "structured",
			}
			out, results, err := h.runPlanAndCollate(ctx, convID, plan, nil, structured, provider)
			if err == nil {
				return out, true, false, results, nil
			}
		}
	}
```

- [ ] **Step 4: Run test to verify it passes**
Run: `go test -v -run TestHandleMessage_PostQualificationUpdate ./internal/receptionist/...`
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add ai-receptionist/internal/receptionist/handler.go
git commit -m "feat: intercept post-qualification updates to directly trigger calendar booking"
```

---

### Task 6: Verify and Run All Tests Locally

- [ ] **Step 1: Execute complete test suite**
Run: `go test ./...` in `ai-receptionist`
Expected: ALL PASS

- [ ] **Step 2: Final local git verification**
Run: `git status`
Expected: No untracked/dirty files except our intended changes.

- [ ] **Step 3: Deploy to VPS and Run Smoke Tests**
Deploy the updated executable and prompt files.
Run: `bash scripts/smoke-sim.sh` on VPS or run local check scripts.
Verify logs for correctness.
