package receptionist

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"ai-receptionist/internal/aiface"
	"ai-receptionist/internal/ai"
	"ai-receptionist/internal/config"
	"ai-receptionist/internal/intent"
	"ai-receptionist/internal/ops"
	"ai-receptionist/internal/store"
	"ai-receptionist/internal/whatsapp"

	"go.mau.fi/whatsmeow/types"
)

func day3Config() *config.Config {
	cfg := testReceptionistConfig()
	cfg.OwnerNumber = "6590013157"
	cfg.Capabilities = config.Capabilities{
		GroupAdmin:        true,
		Calendar:          true,
		OutboundBooking:   true,
		MarketingResearch: true,
		LeadScrape:        true,
	}
	return cfg
}

func intentFromMessage(messages []ai.ChatMessage) intent.Result {
	msg := ""
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			msg = strings.ToLower(messages[i].Content)
			break
		}
	}
	r := intent.Result{Intent: "general", Confidence: 0.9, Summary: "test"}
	switch {
	case strings.Contains(msg, "scrape"):
		r.Intent = "lead_scrape"
	case strings.Contains(msg, "research"):
		r.Intent = "research_request"
	case strings.Contains(msg, "book a meeting with") || strings.Contains(msg, "book a call with"):
		r.Intent = "outbound_book"
	case strings.Contains(msg, "create") && strings.Contains(msg, "group"):
		r.Intent = "group_manage"
	}
	return r
}

func mockScraperAI() aiface.Provider {
	return &aifaceScript{
		name: "scraper-ai",
		fn: func(msgs []aiface.Message, _ bool) (string, error) {
			user := lastUser(msgs)
			if strings.Contains(user, "JSON array") || strings.Contains(user, "Generate") {
				leads := make([]map[string]string, 0, 10)
				for i := 1; i <= 10; i++ {
					leads = append(leads, map[string]string{
						"name": fmt.Sprintf("Lead %d", i), "company": fmt.Sprintf("Co %d", i), "url": fmt.Sprintf("https://co%d.sg", i),
					})
				}
				b, _ := json.Marshal(leads)
				return string(b), nil
			}
			if strings.Contains(user, "Enrich") {
				return `{"name":"Lead 1","company":"Co 1","email":"a@co1.com","phone":"6500000001","url":"https://co1.sg","linkedin":""}`, nil
			}
			if strings.Contains(user, "Score each") {
				return `[{"name":"Lead 1","company":"Co 1","email":"a@co1.com","fit_score":8,"icp_match":"good fit"}]`, nil
			}
			if strings.Contains(user, "pitch angle") {
				return `{"pitch_angle":"Highlight conversion-focused landing pages"}`, nil
			}
			if strings.Contains(user, "QA this lead") {
				out := make([]map[string]any, 0, 10)
				for i := 1; i <= 10; i++ {
					out = append(out, map[string]any{
						"name": fmt.Sprintf("Lead %d", i), "company": fmt.Sprintf("Co %d", i),
						"email": fmt.Sprintf("lead%d@co%d.com", i, i), "fit_score": 8,
						"pitch_angle": "Strong fit for web redesign", "icp_match": "ICP match",
					})
				}
				b, _ := json.Marshal(out)
				return string(b), nil
			}
			return `[]`, nil
		},
	}
}

func mockResearchAI() aiface.Provider {
	return &aifaceScript{
		name: "research-ai",
		fn: func(msgs []aiface.Message, _ bool) (string, error) {
			user := lastUser(msgs)
			if strings.Contains(user, "Research query") || strings.Contains(user, "research angles") {
				return "- angle 1\n- angle 2", nil
			}
			return "## Executive Summary\nMeta ads for dental clinics focus on before/after and trust.\n\n## Key Findings\n- Video creatives outperform static\n- Local targeting wins\n\n## Sources\n- Meta Ads Library\n- Industry reports", nil
		},
	}
}

type aifaceScript struct {
	name string
	fn   func([]aiface.Message, bool) (string, error)
}

func (s *aifaceScript) Complete(_ context.Context, msgs []aiface.Message, jsonMode bool) (string, error) {
	return s.fn(msgs, jsonMode)
}

func lastUser(msgs []aiface.Message) string {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == "user" {
			return msgs[i].Content
		}
	}
	return ""
}

func newDay3Handler(t *testing.T) (*Handler, *store.DB, func() []string, *config.Config) {
	t.Helper()
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	cfg := day3Config()
	var sent []string
	whatsapp.SetTestHooks(
		func(_ context.Context, chat types.JID, text string) error {
			sent = append(sent, fmt.Sprintf("→%s: %s", chat.User, text))
			return nil
		},
		func(context.Context, types.JID, bool) {},
	)
	t.Cleanup(func() {
		whatsapp.ClearTestHooks()
		db.Close()
	})
	intentAI := &scriptProvider{
		name: "intent",
		script: func(msgs []ai.ChatMessage, _ bool) (string, error) {
			r := intentFromMessage(msgs)
			b, _ := json.Marshal(r)
			return string(b), nil
		},
	}
	main := &scriptProvider{name: "main", script: func(_ []ai.ChatMessage, _ bool) (string, error) {
		t.Fatal("main AI should not run for Day 3 dispatch intents")
		return "", nil
	}}
	h := New(cfg, db, main, intentAI, main, main, nil, nil, "test", "", "")
	return h, db, func() []string { return append([]string(nil), sent...) }, cfg
}

// TestDay3AcceptanceSim runs the Day 3 sprint matrix with simulated WhatsApp I/O.
func TestDay3AcceptanceSim(t *testing.T) {
	owner := "6590013157"
	guest := "6598765432"

	t.Run("1_lead_scrape_queues_and_worker_completes", func(t *testing.T) {
		h, db, sent, cfg := newDay3Handler(t)
		msg := "Scrape 10 F&B consultants in Singapore with email and pitch angle"
		ctx, v, in := simInbound(owner, msg)
		if err := h.process(ctx, v, in, msg); err != nil {
			t.Fatal(err)
		}
		out := sent()
		if len(out) != 1 || !strings.Contains(out[0], "Queued lead scrape") {
			t.Fatalf("owner ack=%v", out)
		}
		pending, _ := db.FirstPendingAsyncJob()
		if pending == nil || pending.JobType != "scrape_leads" {
			t.Fatalf("pending=%+v", pending)
		}
		env := ops.WorkerEnv{Store: db, Cfg: cfg, AI: mockScraperAI()}
		w := &ops.AsyncWorker{Store: db, Cfg: cfg, Handlers: ops.DefaultJobHandlers(env)}
		w.ProcessOneBatch(ctx)
		j, _ := db.GetAsyncJob(pending.ID)
		if j.Status != "completed" {
			t.Fatalf("job status=%q err=%q", j.Status, j.Error)
		}
		n, _ := db.CountLeadContactsByJob(pending.ID)
		if n != 10 {
			t.Fatalf("lead_contacts=%d want 10", n)
		}
		if !strings.Contains(j.Result, "Scrape done: 10 leads") {
			t.Fatalf("result=%q", j.Result)
		}
	})

	t.Run("3_research_queues_and_worker_reports", func(t *testing.T) {
		h, db, sent, cfg := newDay3Handler(t)
		msg := "Research what Meta ad angles are working for dental clinics in Singapore"
		ctx, v, in := simInbound(owner, msg)
		if err := h.process(ctx, v, in, msg); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(sent()[0], "Research queued") {
			t.Fatalf("ack=%v", sent())
		}
		pending, _ := db.FirstPendingAsyncJob()
		if pending == nil {
			t.Fatal("no pending research job")
		}
		env := ops.WorkerEnv{Store: db, Cfg: cfg, AI: mockResearchAI()}
		w := &ops.AsyncWorker{Store: db, Cfg: cfg, Handlers: ops.DefaultJobHandlers(env)}
		w.ProcessOneBatch(ctx)
		j, _ := db.GetAsyncJob(pending.ID)
		if j.Status != "completed" || !strings.Contains(j.Result, "Executive Summary") {
			t.Fatalf("job=%+v", j)
		}
	})

	t.Run("4_outbound_book_messaging_and_5_guest_slot_confirm", func(t *testing.T) {
		h, db, sent, cfg := newDay3Handler(t)
		msg := "Book a meeting with John Tan, +6598765432, about Epicware partnership"
		ctx, v, in := simInbound(owner, msg)
		if err := h.process(ctx, v, in, msg); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(sent()[0], "Outbound booking queued") {
			t.Fatalf("ack=%v", sent())
		}
		pending, _ := db.FirstPendingAsyncJob()
		if pending == nil {
			t.Fatal("no pending outbound job")
		}
		env := ops.WorkerEnv{Store: db, Cfg: cfg, WA: &whatsapp.Client{}}
		w := &ops.AsyncWorker{Store: db, Cfg: cfg, Handlers: ops.DefaultJobHandlers(env)}
		w.ProcessOneBatch(ctx)
		j, _ := db.GetAsyncJob(pending.ID)
		if j.Status != "completed" {
			t.Fatalf("job status=%q err=%q", j.Status, j.Error)
		}
		br, _ := db.GetActiveBookingByGuest(guest)
		if br == nil || br.Status != "awaiting_guest" {
			t.Fatalf("booking=%+v", br)
		}
		guestMessaged := false
		for _, line := range sent() {
			if strings.Contains(line, guest+":") && strings.Contains(line, "Pick a slot") {
				guestMessaged = true
				break
			}
		}
		if !guestMessaged {
			t.Fatalf("expected worker guest slot message, sent=%v", sent())
		}

		gctx, gv, gin := simInbound(guest, "2")
		gin.Sender = guest
		if !h.TryHandleGuestBookingReply(gctx, gv, gin) {
			t.Fatal("expected guest reply handled")
		}
		confirmed := false
		for _, line := range sent() {
			if strings.Contains(line, "Confirmed") {
				confirmed = true
				break
			}
		}
		if !confirmed {
			t.Fatalf("guest confirm=%v", sent())
		}
		br, _ = db.GetActiveBookingByGuest(guest)
		if br != nil && br.Status == "awaiting_guest" {
			t.Fatal("booking should be confirmed")
		}
	})

	t.Run("4b_guest_ambiguous_reply_not_confirmed", func(t *testing.T) {
		h, db, _, _ := newDay3Handler(t)
		slotsJSON, _ := json.Marshal([]string{"Mon 3pm", "Tue 10am", "Wed 2pm"})
		_, _ = db.InsertBookingRequest(store.BookingRequest{
			ID: "bk-1", GuestPhone: guest, Status: "awaiting_guest", GuestSlotsJSON: string(slotsJSON),
		})
		var guestSent []string
		whatsapp.SetTestHooks(
			func(_ context.Context, chat types.JID, text string) error {
				guestSent = append(guestSent, text)
				return nil
			},
			func(context.Context, types.JID, bool) {},
		)
		ctx, v, in := simInbound(guest, "who is this?")
		in.Sender = guest
		if !h.TryHandleGuestBookingReply(ctx, v, in) {
			t.Fatal("expected handler to intercept")
		}
		if len(guestSent) != 1 || !strings.Contains(guestSent[0], "1, 2, or 3") {
			t.Fatalf("guest=%v", guestSent)
		}
		br, _ := db.GetActiveBookingByGuest(guest)
		if br == nil || br.Status != "awaiting_guest" {
			t.Fatal("booking should remain awaiting")
		}
	})

	t.Run("6_group_nl_parses_multiword_name", func(t *testing.T) {
		a, ok := parseGroupNL("Create a WhatsApp group Epicware VIP and add +6591234567")
		if !ok || a.Name != "Epicware VIP" {
			t.Fatalf("action=%+v ok=%v", a, ok)
		}
	})
}
