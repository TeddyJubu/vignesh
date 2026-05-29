# Specification: Empathy & UX Hardening for Julia (WhatsApp AI Receptionist)

- **Date**: 2026-05-29
- **Status**: Approved / Drafted

## 1. Overview
This specification details the changes required to elevate the conversational quality, human empathy, and user experience (UX) of Julia, the WhatsApp AI Receptionist. Based on live conversations, several friction points were identified:
1. **Robotic acknowledgements** (such as "Got it — checking now.") on simple greetings.
2. **Contextual amnesia** regarding the user's name or other previously declared fields, leading to repetitive or redundant questions.
3. **Conversational "lies"** where the bot says "one last question" but continues to ask multiple follow-up questions.
4. **Failure to collect critical contact info** (email address) during the qualification funnel.
5. **Rigid loops post-qualification** where the user tries to update their scheduled appointment slot but the bot re-triggers the generic lead intake planner.

---

## 2. Goals & Success Criteria
- **Empathy-First Interactivity**: Julia must greet warmly, respond naturally, and respect conversational cues.
- **Accurate Memory State**: Programmatically guard against asking for fields the user has already provided in chat history.
- **Funnel Truthfulness**: Align phrasing programmatically with the true number of remaining fields.
- **Critical Information Capture**: Systematically require and collect email addresses during booking and qualification.
- **Post-Qualification Agility**: Enable seamless, warm appointment updates for qualified leads without looping back to the intake form.

---

## 3. Detailed Technical Components

### 3.1. Dynamic Ack & Greeting Suppressor (`internal/receptionist/handler.go`)
- **Greeting Detection**: We will define standard greeting patterns to match greetings (e.g., `hi`, `hello`, `hey`, `morning`, `yo`).
- **Short Message Filtering**: If an incoming message is `< 7 characters` or matches a greeting, `shouldSendAck` must return `false` immediately.
- **Empathetic Copy**: The ack text for longer queries (where the background delay triggers) will be warmed up from `"Got it — checking now."` to:
  `"Just a second, let me check that for you... 🔍"`

### 3.2. History-Informed Memory Recovery (`internal/receptionist/handler.go`)
- **Scan Last Turns**: Before compiling the system prompt and injecting `missing_fields`, if `name` or `email` is absent from `leadData`, the handler will check the last 6 messages in the database.
- **Extraction Fallback**:
  - If a message matches a name pattern or contains a valid email address (via regex `\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b`), it is programmatically injected into the active `leadData` in SQL before rendering the prompt.
  - This ensures `missing_fields` never contains fields already explicitly declared in the conversation history, protecting against model extraction failures.

### 3.3. Funnel Extension & Strict "Last Question" Constraints (`internal/lead/fields.go` & Prompts)
- **Email as Required**: Add `"email"` to `lead.Required` in `internal/lead/fields.go`.
- **Constraint Directive injection**: In `buildSystemPrompt`, we calculate `len(Missing(leadData))`. If this is `> 1`, we dynamically append a strict constraint to the prompt stack:
  ```
  CRITICAL CONSTRAINT: You have multiple missing fields to collect. You are strictly forbidden from saying "last question" or "one last quick one". Instead, use phrases like "Just a couple of quick details..." or "Next...". Only refer to a "final question" when exactly one required field remains.
  ```
- **Field Grouping Directive**: Update `prompt.txt` to permit grouping related fields into a single natural question:
  ```
  - Collect lead information dynamically. You may group related fields naturally into a single friendly message to save time (e.g. asking for "budget and timeline" or "name, email, and business name" together), rather than interrogating the user with 6 separate turns.
  ```

### 3.4. Post-Qualification Seamless Booking (`internal/receptionist/handler.go`)
- **Status Guard**: In `completeWithPlanner`, if `contact.Status == "notified"`, the normal planner is bypassed.
- **Direct Calendar Run**: If the user sends scheduling text (e.g., `"Tuesday 3pm Singapore"`), we execute the booking tool directly.
- **Empathetic Confirmation**:
  - On a successful calendar tool run, reply with a warm, personal confirmation:
    `"Got it! I've updated your preferred slot to Tuesday 3pm SGT and notified Vignesh. We're all set! 👍"`
  - If no scheduling slots are resolved, respond on the fast path to warmly confirm receipt and pass the text to Vignesh without looping.

---

## 4. Test Plan & Verification
We will add four automated simulated WhatsApp scenarios in `handler_sim_test.go`:
1. **Greeting Turn**: Verify that sending `"hi"` does *not* trigger any background acknowledgment.
2. **Goldfish Memory Prevention**: Send `"Teddy, Marketing"`, then verify the subsequent turn's `missing_fields` does *not* ask for `"name"`, even if the previous parser failed.
3. **Email Collection**: Verify that `"email"` is requested and saved during the lead qualification flow.
4. **Post-Qualification Slot Update**: Simulate a qualified lead sending `"Tuesday 3pm Singapore"`, and verify that the calendar booking tool executes and confirms the time seamlessly.
