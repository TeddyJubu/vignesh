package receptionist

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"ai-receptionist/internal/agent"
	"ai-receptionist/internal/ai"
	"ai-receptionist/internal/config"
	"ai-receptionist/internal/intent"
	"ai-receptionist/internal/models"
	"ai-receptionist/internal/store"
	"ai-receptionist/internal/whatsapp"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// scriptProvider returns deterministic AI output for simulated WhatsApp turns.
type scriptProvider struct {
	name  string
	script func(messages []ai.ChatMessage, jsonMode bool) (string, error)
	mu    sync.Mutex
	calls int
}

func (p *scriptProvider) Name() string { return p.name }
func (p *scriptProvider) Ping(ctx context.Context) error { return nil }
func (p *scriptProvider) Complete(ctx context.Context, messages []ai.ChatMessage, jsonMode bool) (string, error) {
	p.mu.Lock()
	p.calls++
	p.mu.Unlock()
	return p.script(messages, jsonMode)
}

func (p *scriptProvider) callCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.calls
}

func testReceptionistConfig() *config.Config {
	t := true
	f := false
	return &config.Config{
		BusinessName:        "TestCo",
		OwnerNumber:         "6590000001",
		Model:               "test-model",
		Mode:                "receptionist",
		DebounceSeconds:     0,
		ReplyToSelfChat:     &t,
		EnableLeadTracking:  &f,
		EnableOwnerAlerts:   &f,
		QuietHours:          config.QuietHours{Enabled: false},
	}
}

func structuredReply(msg string) string {
	b, _ := json.Marshal(ai.StructuredResponse{
		Reply:       msg,
		LeadUpdates: map[string]string{},
		Qualified:   false,
		Summary:     "",
	})
	return string(b)
}

func newSimHandler(t *testing.T, main, planner, collate, intentAI *scriptProvider) (*Handler, *store.DB, func() []string) {
	t.Helper()
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	cfg := testReceptionistConfig()
	var sent []string
	var sentMu sync.Mutex
	whatsapp.SetTestHooks(
		func(_ context.Context, _ types.JID, text string) error {
			sentMu.Lock()
			sent = append(sent, text)
			sentMu.Unlock()
			return nil
		},
		func(context.Context, types.JID, bool) {},
	)
	t.Cleanup(func() {
		whatsapp.ClearTestHooks()
		db.Close()
	})
	h := New(cfg, db, main, intentAI, planner, collate, nil, nil, "{{business_name}} test", "", "")
	return h, db, func() []string {
		sentMu.Lock()
		defer sentMu.Unlock()
		out := append([]string(nil), sent...)
		return out
	}
}

func simInbound(convID, text string) (context.Context, *events.Message, whatsapp.InboundContext) {
	ctx := context.Background()
	phone := strings.TrimPrefix(convID, "self:")
	chat := whatsapp.PhoneToJID(phone)
	in := whatsapp.InboundContext{
		ConvID:  convID,
		Sender:  phone,
		Text:    text,
		IsGroup: false,
	}
	v := &events.Message{}
	v.Info.Chat = chat
	v.Info.Sender = chat
	return ctx, v, in
}

func TestSimulatedWhatsApp_FastPathHello(t *testing.T) {
	main := &scriptProvider{
		name: "main",
		script: func(_ []ai.ChatMessage, jsonMode bool) (string, error) {
			if !jsonMode {
				t.Fatal("fast path expected structured json mode")
			}
			return structuredReply("Hello! How can I help?"), nil
		},
	}
	planner := &scriptProvider{name: "planner", script: func(_ []ai.ChatMessage, _ bool) (string, error) {
		t.Fatal("planner should not run on hello")
		return "", nil
	}}
	collate := &scriptProvider{name: "collate", script: func(_ []ai.ChatMessage, _ bool) (string, error) {
		t.Fatal("collate should not run on hello")
		return "", nil
	}}
	intentAI := &scriptProvider{
		name: "intent",
		script: func(_ []ai.ChatMessage, _ bool) (string, error) {
			b, _ := json.Marshal(intent.Result{Intent: "general", Confidence: 0.9, Summary: "greeting"})
			return string(b), nil
		},
	}
	h, _, sent := newSimHandler(t, main, planner, collate, intentAI)
	ctx, v, in := simInbound("6591111111", "hello")

	if err := h.process(ctx, v, in, "hello"); err != nil {
		t.Fatal(err)
	}
	out := sent()
	if len(out) != 1 {
		t.Fatalf("sent=%v", out)
	}
	if !strings.Contains(out[0], "Hello") {
		t.Fatalf("reply=%q", out[0])
	}
}

func TestSimulatedWhatsApp_PlannerQuestionThenCollate(t *testing.T) {
	planner := &scriptProvider{
		name: "planner",
		script: func(_ []ai.ChatMessage, jsonMode bool) (string, error) {
			if jsonMode {
				t.Fatal("planner should not use json mode")
			}
			plan := agent.Plan{
				Goal:              "book call",
				Questions:         []string{"What day and time works for you (with timezone)?"},
				FinalResponseMode: "structured",
				Agents: []agent.AgentTask{
					{Name: "cal", Tool: "check_calendar_availability", Input: "next week"},
				},
			}
			b, _ := json.Marshal(plan)
			return string(b), nil
		},
	}
	collate := &scriptProvider{
		name: "collate",
		script: func(_ []ai.ChatMessage, jsonMode bool) (string, error) {
			if !jsonMode {
				t.Fatal("collate expected json mode")
			}
			return structuredReply("Tuesday 3pm SGT works — I'll confirm shortly."), nil
		},
	}
	main := &scriptProvider{
		name: "main",
		script: func(_ []ai.ChatMessage, _ bool) (string, error) {
			return structuredReply("fallback"), nil
		},
	}
	intentAI := &scriptProvider{
		name: "intent",
		script: func(_ []ai.ChatMessage, _ bool) (string, error) {
			b, _ := json.Marshal(intent.Result{Intent: "sales_qualify", Confidence: 0.88, Summary: "book"})
			return string(b), nil
		},
	}
	h, db, sent := newSimHandler(t, main, planner, collate, intentAI)
	convID := "6592222222"

	// Turn 1: triggers planner question
	ctx1, v1, in1 := simInbound(convID, "I want to book a meeting next week")
	if err := h.process(ctx1, v1, in1, "I want to book a meeting next week"); err != nil {
		t.Fatal(err)
	}
	s1 := sent()
	if len(s1) != 1 || !strings.Contains(s1[0], "day and time") {
		t.Fatalf("turn1 sent=%v", s1)
	}
	st, err := db.GetAgentState(convID)
	if err != nil {
		t.Fatal(err)
	}
	if st == nil || st.StateJSON == "" {
		t.Fatal("expected persisted agent state")
	}

	// Turn 2: answer → tools + collate
	ctx2, v2, in2 := simInbound(convID, "Tuesday 3pm Singapore")
	if err := h.process(ctx2, v2, in2, "Tuesday 3pm Singapore"); err != nil {
		t.Fatal(err)
	}
	s2 := sent()
	if len(s2) != 2 {
		t.Fatalf("sent messages=%v", s2)
	}
	if !strings.Contains(s2[1], "Tuesday") && !strings.Contains(s2[1], "3pm") {
		t.Fatalf("final reply=%q", s2[1])
	}
	if collate.callCount() < 1 {
		t.Fatal("collate was not called")
	}
}

func TestSimulatedWhatsApp_PostQualificationUpdate(t *testing.T) {
	convID := "6593333333"

	collate := &scriptProvider{
		name: "collate",
		script: func(_ []ai.ChatMessage, jsonMode bool) (string, error) {
			if !jsonMode {
				t.Fatal("collate expected structured json mode")
			}
			return structuredReply("Got it! I've updated your preferred slot to Tuesday 3pm SGT and notified Vignesh. We're all set! 👍"), nil
		},
	}
	planner := &scriptProvider{name: "planner", script: func(_ []ai.ChatMessage, _ bool) (string, error) {
		t.Fatal("planner should not run for post-qualification slot update")
		return "", nil
	}}
	main := &scriptProvider{name: "main", script: func(_ []ai.ChatMessage, _ bool) (string, error) {
		t.Fatal("main should not run for post-qualification slot update")
		return "", nil
	}}
	intentAI := &scriptProvider{name: "intent", script: func(_ []ai.ChatMessage, _ bool) (string, error) {
		b, _ := json.Marshal(intent.Result{Intent: "sales_qualify", Confidence: 0.9, Summary: "schedule update"})
		return string(b), nil
	}}

	h, db, sent := newSimHandler(t, main, planner, collate, intentAI)

	if _, err := db.GetOrCreateContact(convID); err != nil {
		t.Fatal(err)
	}
	leadData := map[string]string{
		"name":             "Teddy",
		"email":            "teddy@example.com",
		"business_type":    "SaaS",
		"service_needed":   "website",
		"budget":           "$5k",
		"timeline":         "Q2",
		"current_website":  "none",
	}
	leadJSON, _ := json.Marshal(leadData)
	if err := db.UpdateContactWithScore(convID, "Teddy", string(leadJSON), "notified", "high"); err != nil {
		t.Fatal(err)
	}
	if err := db.SetContactMode(convID, modeBooking); err != nil {
		t.Fatal(err)
	}

	ctx, v, in := simInbound(convID, "Tuesday 3pm Singapore")
	if err := h.process(ctx, v, in, "Tuesday 3pm Singapore"); err != nil {
		t.Fatal(err)
	}
	out := sent()
	if len(out) != 1 {
		t.Fatalf("sent=%v", out)
	}
	if !strings.Contains(out[0], "Tuesday") || !strings.Contains(out[0], "3pm") {
		t.Fatalf("reply=%q", out[0])
	}
	if planner.callCount() != 0 {
		t.Fatalf("planner calls=%d, want 0", planner.callCount())
	}
	if collate.callCount() < 1 {
		t.Fatal("collate was not called")
	}
}

func TestSimulatedWhatsApp_IntentEchoMode(t *testing.T) {
	t.Setenv("ECHO_INTENT", "1")
	main := &scriptProvider{name: "main", script: func(_ []ai.ChatMessage, _ bool) (string, error) {
		t.Fatal("main should not run in echo intent mode")
		return "", nil
	}}
	planner := &scriptProvider{name: "planner", script: func(_ []ai.ChatMessage, _ bool) (string, error) {
		return "", nil
	}}
	collate := &scriptProvider{name: "collate", script: func(_ []ai.ChatMessage, _ bool) (string, error) {
		return "", nil
	}}
	intentAI := &scriptProvider{
		name: "intent",
		script: func(_ []ai.ChatMessage, _ bool) (string, error) {
			b, _ := json.Marshal(intent.Result{Intent: "support", Confidence: 0.77, Summary: "pricing"})
			return string(b), nil
		},
	}
	h, _, sent := newSimHandler(t, main, planner, collate, intentAI)
	ctx, v, in := simInbound("6593333333", "What are your pricing plans?")
	if err := h.process(ctx, v, in, "What are your pricing plans?"); err != nil {
		t.Fatal(err)
	}
	out := sent()
	if len(out) != 1 || !strings.HasPrefix(out[0], "intent=support") {
		t.Fatalf("echo=%q", out[0])
	}
}

func TestSimulatedWhatsApp_ModelRoutingAnthropic(t *testing.T) {
	models.SetActiveProvider("anthropic")
	models.SetConfigModel("claude-sonnet-4-6")
	if got := models.GetModel("intent_classify"); got != models.AnthropicModelHaiku {
		t.Fatalf("intent model=%q", got)
	}
	if got := models.GetModel("planner"); got != models.AnthropicModelSonnet {
		t.Fatalf("planner model=%q", got)
	}
}

func TestSimulatedWhatsApp_GreetingDoesNotTriggerAck(t *testing.T) {
	prevAck := ackDelay
	ackDelay = 2 * time.Millisecond
	t.Cleanup(func() { ackDelay = prevAck })

	main := &scriptProvider{
		name: "main",
		script: func(_ []ai.ChatMessage, jsonMode bool) (string, error) {
			if !jsonMode {
				t.Fatal("fast path expected structured json mode")
			}
			time.Sleep(10 * time.Millisecond)
			return structuredReply("Hi there! How can I help?"), nil
		},
	}
	planner := &scriptProvider{name: "planner", script: func(_ []ai.ChatMessage, _ bool) (string, error) {
		t.Fatal("planner should not run on hi")
		return "", nil
	}}
	collate := &scriptProvider{name: "collate", script: func(_ []ai.ChatMessage, _ bool) (string, error) {
		t.Fatal("collate should not run on hi")
		return "", nil
	}}
	intentAI := &scriptProvider{
		name: "intent",
		script: func(_ []ai.ChatMessage, _ bool) (string, error) {
			b, _ := json.Marshal(intent.Result{Intent: "general", Confidence: 0.9, Summary: "greeting"})
			return string(b), nil
		},
	}
	h, _, sent := newSimHandler(t, main, planner, collate, intentAI)
	ctx, v, in := simInbound("6595555555", "hi")

	if err := h.process(ctx, v, in, "hi"); err != nil {
		t.Fatal(err)
	}
	out := sent()
	if len(out) != 1 {
		t.Fatalf("expected exactly one reply, sent=%v", out)
	}
	for _, msg := range out {
		if strings.Contains(msg, "checking now") || strings.Contains(msg, "Just a second") {
			t.Fatalf("ack should not be sent for greeting, sent=%v", out)
		}
	}
	if !strings.Contains(out[0], "Hi there") {
		t.Fatalf("reply=%q", out[0])
	}
}

func TestCompleteWithPlanner_UsesPlannerProvider(t *testing.T) {
	planner := &scriptProvider{
		name: "planner",
		script: func(_ []ai.ChatMessage, _ bool) (string, error) {
			return `{"goal":"x","agents":[],"questions":[],"final_response_mode":"structured"}`, nil
		},
	}
	main := &scriptProvider{name: "main", script: func(_ []ai.ChatMessage, _ bool) (string, error) {
		return structuredReply("fast"), nil
	}}
	h, _, _ := newSimHandler(t, main, planner, main, main)
	msgs := []ai.ChatMessage{{Role: "user", Content: "book a call tomorrow"}}
	_, _, _, _, err := h.completeWithPlanner(context.Background(), "6594444444", msgs, true, "main", modeBooking, "book a call tomorrow")
	if err != nil {
		t.Fatal(err)
	}
	if planner.callCount() < 1 {
		t.Fatal("expected planner provider call")
	}
}

// Avoid flaky ack goroutine leaking after tests.
func TestBuildSystemPrompt_InjectsLastQuestionConstraint(t *testing.T) {
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	tTrue := true
	cfg := testReceptionistConfig()
	cfg.EnableLeadTracking = &tTrue
	h := New(cfg, db, nil, nil, nil, nil, nil, nil, "System prompt", "", "")

	leadData := map[string]string{}
	prompt, err := h.buildSystemPrompt("6591234567", leadData, whatsapp.InboundContext{}, "en", "receptionist", "book a call", false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(prompt, "CRITICAL CONSTRAINT") {
		t.Fatalf("prompt missing CRITICAL CONSTRAINT:\n%s", prompt)
	}
	if !strings.Contains(prompt, "forbidden from saying") {
		t.Fatalf("prompt missing forbidden phrasing constraint:\n%s", prompt)
	}
}

func TestBuildSystemPrompt_BackfillsNameFromHistory(t *testing.T) {
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	convID := "6591234567"
	if err := db.InsertMessage(convID, "user", "hi, I am Teddy"); err != nil {
		t.Fatal(err)
	}

	tTrue := true
	cfg := testReceptionistConfig()
	cfg.EnableLeadTracking = &tTrue
	h := New(cfg, db, nil, nil, nil, nil, nil, nil, "System prompt", "", "")

	leadData := map[string]string{}
	prompt, err := h.buildSystemPrompt(convID, leadData, whatsapp.InboundContext{}, "en", "receptionist", "book a call", false)
	if err != nil {
		t.Fatal(err)
	}

	const prefix = "missing_fields: "
	idx := strings.Index(prompt, prefix)
	if idx < 0 {
		t.Fatal("system prompt missing missing_fields section")
	}
	rest := prompt[idx+len(prefix):]
	lineEnd := strings.IndexByte(rest, '\n')
	if lineEnd < 0 {
		lineEnd = len(rest)
	}
	var missing []string
	if err := json.Unmarshal([]byte(rest[:lineEnd]), &missing); err != nil {
		t.Fatalf("parse missing_fields: %v", err)
	}
	for _, f := range missing {
		if f == "name" {
			t.Errorf("system prompt still lists name as missing after being stated in history:\n%s", prompt)
		}
	}
}

func TestMain(m *testing.M) {
	ackDelay = 10 * time.Millisecond
	overallAITimeout = 5 * time.Second
	m.Run()
}
